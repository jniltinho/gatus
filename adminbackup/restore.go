package adminbackup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	pushconfig "gatus/v5/config/push"
	"gatus/v5/managedendpoint"
	"gatus/v5/pushkey"
	"gatus/v5/statuspage"
	"gatus/v5/storage/store/common"
	"github.com/TwiN/logr"
)

const (
	// Types of the items of a restore, in the order in which they are planned and applied
	TypePushKey    = "pushKey"
	TypeEndpoint   = "endpoint"
	TypeStatusPage = "statusPage"

	// Actions of the items of a plan
	ActionCreate    = "create"
	ActionUpdate    = "update"
	ActionUnchanged = "unchanged"
	ActionSkip      = "skip"

	// Results of the items of an applied restore
	ResultCreated   = "created"
	ResultUpdated   = "updated"
	ResultUnchanged = "unchanged"
	ResultSkipped   = "skipped"
	ResultFailed    = "failed"

	reasonAlreadyExists  = "already exists"
	reasonNameOrGroup    = "name or group changes"
	reasonReloadProgress = "configuration reload in progress"
	reasonChanged        = "changed during the restore"
)

// ErrFingerprintMismatch is returned when the plan changed since the preview
var ErrFingerprintMismatch = errors.New("the backup, the options or the registered items changed since the preview: preview the restore again")

// Options are the options of a restore
type Options struct {
	// Overwrite updates the items that already exist with a different definition
	Overwrite bool

	// DisableEndpoints creates and updates the endpoints disabled
	DisableEndpoints bool
}

// Plan is what a restore would do, without effects
type Plan struct {
	Summary     Summary `json:"summary"`
	Notices     Notices `json:"notices"`
	Fingerprint string  `json:"fingerprint"`
	Items       []*Item `json:"items"`
}

// Summary counts the items of a plan by action
type Summary struct {
	Create    int `json:"create"`
	Update    int `json:"update"`
	Unchanged int `json:"unchanged"`
	Skip      int `json:"skip"`
}

// Notices tell what the restore will start
type Notices struct {
	// MonitoringStarts is the number of enabled endpoints that will be created or updated, and so monitored
	MonitoringStarts int `json:"monitoringStarts"`

	// WithAlerts is the number of those endpoints with alerts
	WithAlerts int `json:"withAlerts"`
}

// Item is an item of a plan
type Item struct {
	Type     string   `json:"type"`
	ID       string   `json:"id"`
	Action   string   `json:"action"`
	Reason   string   `json:"reason"`
	Warnings []string `json:"warnings"`

	// What is applied, as previewed
	definition    []byte
	version       int64
	currentSHA256 string
	tokenHash     string
	hint          string
}

// Result is the result of an applied restore
type Result struct {
	Summary ResultSummary `json:"summary"`
	Results []*ItemResult `json:"results"`
}

// ResultSummary counts the items of an applied restore by result
type ResultSummary struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
}

// ItemResult is the result of an item of an applied restore
type ItemResult struct {
	Type     string   `json:"type"`
	ID       string   `json:"id"`
	Result   string   `json:"result"`
	Message  string   `json:"message"`
	Warnings []string `json:"warnings"`
}

// Restorer plans and applies the restores of backups with the services of the administration
type Restorer struct {
	Endpoints   *managedendpoint.Service
	StatusPages *statuspage.Service
}

// fingerprintInput is encoded with its fields in this order to compute the fingerprint of a plan
type fingerprintInput struct {
	PlaintextSHA256  string             `json:"plaintextSHA256"`
	Overwrite        bool               `json:"overwrite"`
	DisableEndpoints bool               `json:"disableEndpoints"`
	Generation       uint64             `json:"generation"`
	Items            []fingerprintEntry `json:"items"`
}

type fingerprintEntry struct {
	Type          string `json:"type"`
	ID            string `json:"id"`
	Action        string `json:"action"`
	Version       int64  `json:"version"`
	CurrentSHA256 string `json:"currentSHA256"`
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (item *Item) skip(reason string) *Item {
	item.Action, item.Reason = ActionSkip, reason
	return item
}

// Plan returns what restoring the backup would do, without effects. plaintext is the backup file as decoded, whose
// hash is part of the fingerprint. The plan simulates the items in the order in which they are applied: push keys,
// endpoints and status pages.
func (r *Restorer) Plan(plaintext []byte, options Options) (*Plan, error) {
	file, err := Decode(plaintext)
	if err != nil {
		return nil, err
	}
	if isAnyRegistryUnavailable() {
		return nil, ErrUnavailable
	}
	plan := &Plan{Items: []*Item{}}
	// Every push token of the endpoints, including the invalid managed endpoints and the endpoints of the backup, so that
	// a push key never accepts the token of an endpoint
	endpointTokenHashes := make(map[[sha256.Size]byte]bool)
	for _, token := range r.Endpoints.EndpointPushTokens() {
		endpointTokenHashes[pushconfig.HashToken(token)] = true
	}
	for _, item := range file.Endpoints {
		if token := managedendpoint.DefinitionPushToken([]byte(item.Definition)); len(token) > 0 {
			endpointTokenHashes[pushconfig.HashToken(token)] = true
		}
	}
	ctx := &managedendpoint.RestoreContext{PlannedTokens: make(map[string]string), PlannedKeyHashes: make(map[[sha256.Size]byte]bool)}
	sort.Slice(file.PushKeys, func(i, j int) bool { return file.PushKeys[i].Name < file.PushKeys[j].Name })
	for _, pushKey := range file.PushKeys {
		plan.Items = append(plan.Items, planPushKey(pushKey, endpointTokenHashes, ctx))
	}
	sort.Slice(file.Endpoints, func(i, j int) bool { return file.Endpoints[i].Key < file.Endpoints[j].Key })
	var plannedRefs []statuspage.EndpointRef
	for _, backupEndpoint := range file.Endpoints {
		item, prepared := r.planEndpoint(backupEndpoint, options, ctx)
		plan.Items = append(plan.Items, item)
		if prepared != nil && (item.Action == ActionCreate || item.Action == ActionUpdate) && prepared.IsEnabled() {
			plan.Notices.MonitoringStarts++
			if prepared.AlertCount() > 0 {
				plan.Notices.WithAlerts++
			}
			plannedRefs = append(plannedRefs, statuspage.EndpointRef{Key: prepared.Key(), Name: prepared.Name(), Group: prepared.Group()})
		}
	}
	refs := append(statuspage.Endpoints(), plannedRefs...)
	sort.Slice(file.StatusPages, func(i, j int) bool { return file.StatusPages[i].Slug < file.StatusPages[j].Slug })
	for _, backupStatusPage := range file.StatusPages {
		plan.Items = append(plan.Items, r.planStatusPage(backupStatusPage, options, refs))
	}
	fingerprint := fingerprintInput{PlaintextSHA256: sha256Hex(plaintext), Overwrite: options.Overwrite, DisableEndpoints: options.DisableEndpoints, Generation: statuspage.Generation(), Items: []fingerprintEntry{}}
	for _, item := range plan.Items {
		switch item.Action {
		case ActionCreate:
			plan.Summary.Create++
		case ActionUpdate:
			plan.Summary.Update++
		case ActionUnchanged:
			plan.Summary.Unchanged++
		default:
			plan.Summary.Skip++
		}
		fingerprint.Items = append(fingerprint.Items, fingerprintEntry{Type: item.Type, ID: item.ID, Action: item.Action, Version: item.version, CurrentSHA256: item.currentSHA256})
	}
	encoded, err := json.Marshal(fingerprint)
	if err != nil {
		return nil, err
	}
	plan.Fingerprint = sha256Hex(encoded)
	return plan, nil
}

func planPushKey(backupKey PushKey, endpointTokenHashes map[[sha256.Size]byte]bool, ctx *managedendpoint.RestoreContext) *Item {
	item := &Item{Type: TypePushKey, ID: backupKey.Name, Warnings: []string{}, tokenHash: backupKey.TokenHash, hint: backupKey.Hint}
	hash, err := pushkey.ValidateRestoredKey(backupKey.Name, backupKey.TokenHash, backupKey.Hint)
	if err != nil {
		return item.skip(err.Error())
	}
	if existing, exists := pushkey.FindByName(backupKey.Name, hash); exists {
		switch {
		case existing.Origin == pushkey.OriginConfig:
			return item.skip("name in use by a push key of the configuration file")
		case existing.SameHash:
			item.Action = ActionUnchanged
			return item
		default:
			item.currentSHA256 = hex.EncodeToString(existing.Hash[:])
			return item.skip("name in use by another push key")
		}
	}
	if pushkey.IsHashInUse(hash) || endpointTokenHashes[hash] || ctx.PlannedKeyHashes[hash] {
		return item.skip(pushkey.ErrHashInUse.Error())
	}
	ctx.PlannedKeyHashes[hash] = true
	item.Action = ActionCreate
	return item
}

func (r *Restorer) planEndpoint(backupEndpoint Endpoint, options Options, ctx *managedendpoint.RestoreContext) (*Item, *managedendpoint.Prepared) {
	item := &Item{Type: TypeEndpoint, ID: backupEndpoint.Key, Warnings: []string{}}
	raw := []byte(backupEndpoint.Definition)
	if options.DisableEndpoints {
		document, err := managedendpoint.ToDocument(raw)
		if err != nil {
			return item.skip(err.Error()), nil
		}
		document["enabled"] = false
		if raw, err = managedendpoint.FromDocument(document); err != nil {
			return item.skip(err.Error()), nil
		}
	}
	parsed, err := managedendpoint.ParseDefinition(raw)
	if err != nil {
		return item.skip(err.Error()), nil
	}
	if parsed.Key() != backupEndpoint.Key {
		return item.skip(fmt.Sprintf("the key does not match the definition (%s)", parsed.Key())), nil
	}
	state := managedendpoint.Get(backupEndpoint.Key)
	if state != nil {
		item.version, item.currentSHA256 = state.Stored.Version, sha256Hex([]byte(state.Stored.Definition))
	}
	prepared, err := r.Endpoints.ValidateRestore(raw, ctx)
	if err != nil {
		return item.skip(err.Error()), nil
	}
	item.definition = prepared.Definition
	if state != nil {
		stored, err := managedendpoint.NormalizeDefinition([]byte(state.Stored.Definition))
		if err == nil && string(stored) == string(prepared.Definition) {
			item.Action = ActionUnchanged
			return item, prepared
		}
		if !options.Overwrite {
			return item.skip(reasonAlreadyExists), prepared
		}
		item.Action = ActionUpdate
		if managedendpoint.NameOrGroupChanges(&prepared.Parsed) {
			item.Reason = reasonNameOrGroup
		}
	} else {
		item.Action = ActionCreate
		ctx.PlannedKeys = append(ctx.PlannedKeys, prepared.Key())
	}
	if token := prepared.PushToken(); len(token) > 0 {
		ctx.PlannedTokens[token] = "another endpoint of the backup"
	}
	return item, prepared
}

func (r *Restorer) planStatusPage(backupStatusPage StatusPage, options Options, refs []statuspage.EndpointRef) *Item {
	item := &Item{Type: TypeStatusPage, ID: backupStatusPage.Slug, Warnings: []string{}}
	storedDefinition, version, exists := statuspage.ManagedDefinition(backupStatusPage.Slug)
	if exists {
		item.version, item.currentSHA256 = version, sha256Hex([]byte(storedDefinition))
	}
	page, definition, warnings, err := r.StatusPages.ValidateRestore([]byte(backupStatusPage.Definition), refs)
	if err != nil {
		return item.skip(err.Error())
	}
	if page.Slug != backupStatusPage.Slug {
		return item.skip(fmt.Sprintf("the slug does not match the definition (%s)", page.Slug))
	}
	item.definition, item.Warnings = definition, describeWarnings(warnings)
	if exists {
		if _, stored, err := statuspage.NormalizeDefinition([]byte(storedDefinition)); err == nil && string(stored) == string(definition) {
			item.Action = ActionUnchanged
			return item
		}
		if !options.Overwrite {
			return item.skip(reasonAlreadyExists)
		}
		item.Action = ActionUpdate
		return item
	}
	item.Action = ActionCreate
	return item
}

func describeWarnings(warnings []statuspage.Warning) []string {
	described := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		described = append(described, fmt.Sprintf("%s %q selects nothing", warning.Type, warning.Value))
	}
	return described
}

// Apply applies the restore previewed with the given fingerprint, item by item, with the services of the
// administration and author as the author of every change. The failure of an item does not stop the others; when a
// reload of the configuration starts, the remaining items are skipped.
func (r *Restorer) Apply(plaintext []byte, options Options, fingerprint, author string) (*Result, error) {
	plan, err := r.Plan(plaintext, options)
	if err != nil {
		return nil, err
	}
	if plan.Fingerprint != fingerprint {
		return nil, ErrFingerprintMismatch
	}
	result := &Result{Results: make([]*ItemResult, 0, len(plan.Items))}
	reloading := false
	for _, item := range plan.Items {
		itemResult := &ItemResult{Type: item.Type, ID: item.ID, Message: item.Reason, Warnings: item.Warnings}
		switch {
		case item.Action == ActionUnchanged:
			itemResult.Result = ResultUnchanged
		case item.Action == ActionSkip:
			itemResult.Result = ResultSkipped
		case reloading:
			itemResult.Result, itemResult.Message = ResultSkipped, reasonReloadProgress
		default:
			err := r.applyItem(item, author, itemResult)
			switch {
			case err == nil:
			case isCycleInProgress(err):
				reloading = true
				itemResult.Result, itemResult.Message = ResultSkipped, reasonReloadProgress
			case errors.Is(err, common.ErrManagedEndpointVersionMismatch) || errors.Is(err, common.ErrManagedStatusPageVersionMismatch):
				itemResult.Result, itemResult.Message = ResultFailed, reasonChanged
			default:
				itemResult.Result, itemResult.Message = ResultFailed, err.Error()
			}
		}
		result.count(itemResult.Result)
		result.Results = append(result.Results, itemResult)
	}
	logr.Infof("[adminbackup.Apply] Restore by %s: %d created, %d updated, %d unchanged, %d skipped, %d failed", auditAuthor(author), result.Summary.Created, result.Summary.Updated, result.Summary.Unchanged, result.Summary.Skipped, result.Summary.Failed)
	return result, nil
}

func (r *Restorer) applyItem(item *Item, author string, itemResult *ItemResult) error {
	switch item.Type {
	case TypePushKey:
		if _, err := pushkey.Restore(item.ID, item.tokenHash, item.hint, author, r.Endpoints.EndpointTokenGuard()); err != nil {
			return err
		}
		itemResult.Result = ResultCreated
	case TypeEndpoint:
		var err error
		if item.Action == ActionCreate {
			_, err = r.Endpoints.RestoreCreate(item.definition, author)
			itemResult.Result = ResultCreated
		} else {
			_, err = r.Endpoints.Update(item.ID, item.definition, item.version, author)
			itemResult.Result = ResultUpdated
		}
		if err != nil {
			return err
		}
	case TypeStatusPage:
		var detail *statuspage.Detail
		var err error
		if item.Action == ActionCreate {
			detail, err = r.StatusPages.Create(item.definition, author)
			itemResult.Result = ResultCreated
		} else {
			detail, err = r.StatusPages.Update(item.ID, item.definition, item.version, author)
			itemResult.Result = ResultUpdated
		}
		if err != nil {
			return err
		}
		if detail != nil && detail.Definition != nil {
			itemResult.Warnings = describeWarnings(statuspage.SelectionWarnings(detail.Definition))
		}
	}
	return nil
}

func isCycleInProgress(err error) bool {
	return errors.Is(err, managedendpoint.ErrCycleInProgress) || errors.Is(err, statuspage.ErrCycleInProgress) || errors.Is(err, pushkey.ErrCycleInProgress)
}

func (result *Result) count(itemResult string) {
	switch itemResult {
	case ResultCreated:
		result.Summary.Created++
	case ResultUpdated:
		result.Summary.Updated++
	case ResultUnchanged:
		result.Summary.Unchanged++
	case ResultSkipped:
		result.Summary.Skipped++
	default:
		result.Summary.Failed++
	}
}
