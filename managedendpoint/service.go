package managedendpoint

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"gatus/v5/alerting/provider"
	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/lifecycle"
	"gatus/v5/liveupdates"
	"gatus/v5/metrics"
	"gatus/v5/pushkey"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
	"gatus/v5/watchdog"
	"github.com/TwiN/logr"
	"gopkg.in/yaml.v3"
)

const (
	// SourceConfig identifies endpoints defined in the configuration file
	SourceConfig = "config"

	// SourceAdmin identifies endpoints managed through the administration API
	SourceAdmin = "admin"

	testTimeout                    = 10 * time.Second
	maximumConcurrentTests         = 2
	maximumResolvedConditionLength = 512
)

var (
	// ErrNotFound is returned when no endpoint exists with the given key
	ErrNotFound = errors.New("endpoint not found")

	// ErrReadOnly is returned when trying to change an endpoint defined in the configuration file
	ErrReadOnly = errors.New("the endpoint is defined in the configuration file and cannot be changed through the administration")

	// ErrCycleInProgress is returned when a start or configuration reload is in progress
	ErrCycleInProgress = errors.New("a start or configuration reload is in progress, try again later")

	// ErrTooManyTests is returned when too many endpoint tests are in progress
	ErrTooManyTests = errors.New("too many endpoint tests in progress, try again later")

	// ErrStorageNotSupported is returned when the storage does not support managed endpoints
	ErrStorageNotSupported = errors.New("the storage does not support managed endpoints")

	// ErrApplyFailed is returned when a change could not be applied to the monitoring
	ErrApplyFailed = errors.New("failed to apply the change to the monitoring")

	testSlots = make(chan struct{}, maximumConcurrentTests)
)

// Item summarizes an endpoint for the administration list
type Item struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	Group          string `json:"group"`
	Type           string `json:"type,omitempty"`
	URL            string `json:"url,omitempty"`
	Interval       string `json:"interval,omitempty"`
	Enabled        bool   `json:"enabled"`
	Source         string `json:"source"`
	Conflict       bool   `json:"conflict"`
	ConflictOrigin string `json:"conflictOrigin,omitempty"`
	Error          string `json:"error,omitempty"`
	// AcceptsPush is whether the endpoint receives push (fork)
	AcceptsPush bool       `json:"acceptsPush,omitempty"`
	Version     int64      `json:"version,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
}

// Definition is an endpoint definition with its secrets masked, in YAML and as a JSON document with the YAML keys
type Definition struct {
	YAML string         `json:"yaml"`
	JSON map[string]any `json:"json"`
}

// Detail is an endpoint with its definitions
type Detail struct {
	Item
	// Definition is the persisted definition for managed endpoints, or the definition from the configuration file
	Definition *Definition `json:"definition"`
	// Effective is the definition with default values, when the endpoint is valid
	Effective *Definition `json:"effective,omitempty"`
	// AffectedConfigStatusPages are the status pages of the configuration file that still select the old key, after a
	// rename
	AffectedConfigStatusPages []AffectedStatusPage `json:"affectedConfigStatusPages,omitempty"`
	// PushToken is the push token of a managed endpoint that receives push, masked in its definition (fork)
	PushToken string `json:"pushToken,omitempty"`
}

// Validation is the result of a successful validation
type Validation struct {
	Definition *Definition `json:"definition"`
	Effective  *Definition `json:"effective"`
}

// TestResult is the result of a single evaluation of an endpoint definition
type TestResult struct {
	Success          bool                  `json:"success"`
	DurationMs       int64                 `json:"durationMs"`
	HTTPStatus       int                   `json:"status,omitempty"`
	Errors           []string              `json:"errors,omitempty"`
	ConditionResults []TestConditionResult `json:"conditionResults"`
}

// TestConditionResult is the result of a condition of a tested endpoint
type TestConditionResult struct {
	Condition string `json:"condition"`
	Success   bool   `json:"success"`
}

// Metadata describes what a managed endpoint can use
type Metadata struct {
	AlertTypes  []string `json:"alertTypes"`
	Tunnels     []string `json:"tunnels"`
	ExtraLabels []string `json:"extraLabels"`
}

// Service administers the managed endpoints against a loaded configuration
type Service struct {
	cfg *config.Config
}

// NewService returns the administration service for the given configuration
func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

// List returns the endpoints and external endpoints of the configuration file and the managed endpoints, ordered by key
func (s *Service) List() []*Item {
	items := make([]*Item, 0, len(s.cfg.Endpoints)+len(s.cfg.ExternalEndpoints))
	for _, ep := range s.cfg.Endpoints {
		items = append(items, s.configItem(ep))
	}
	for _, externalEndpoint := range s.cfg.ExternalEndpoints {
		items = append(items, configExternalItem(externalEndpoint))
	}
	for _, state := range List() {
		items = append(items, adminItem(state))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
	return items
}

// Get returns the endpoint with the given key, with its definitions
func (s *Service) Get(key string) (*Detail, error) {
	if state := Get(key); state != nil {
		return adminDetail(state)
	}
	if ep := s.cfg.GetEndpointByKey(key); ep != nil {
		effective := configDefinition(ep.Key())
		if effective == nil {
			effective = []byte("{}")
		}
		definition, err := maskedDefinition(effective)
		if err != nil {
			return nil, err
		}
		return &Detail{Item: *s.configItem(ep), Definition: definition, Effective: definition}, nil
	}
	if externalEndpoint := s.cfg.GetExternalEndpointByKey(key); externalEndpoint != nil {
		marshalled, err := yaml.Marshal(externalEndpoint)
		if err != nil {
			return nil, err
		}
		definition, err := maskedDefinition(marshalled)
		if err != nil {
			return nil, err
		}
		return &Detail{Item: *configExternalItem(externalEndpoint), Definition: definition, Effective: definition}, nil
	}
	return nil, ErrNotFound
}

// Validate validates a definition without persisting it. When key is not empty, the definition is validated as an
// update of that managed endpoint: its own key is not a conflict and masked secrets are restored.
func (s *Service) Validate(raw []byte, key string) (*Validation, error) {
	prepared, err := s.prepare(raw, key)
	if err != nil {
		return nil, err
	}
	definition, err := maskedDefinition(prepared.Definition)
	if err != nil {
		return nil, err
	}
	effective, err := effectiveDefinition(&prepared.Parsed)
	if err != nil {
		return nil, err
	}
	maskedEffective, err := maskedDefinition(effective)
	if err != nil {
		return nil, err
	}
	return &Validation{Definition: definition, Effective: maskedEffective}, nil
}

// ParseDefinition decodes a definition strictly and returns it as a document with the YAML keys. It neither validates
// the endpoint nor reads stored data, so it only returns what was submitted: it is used to convert a definition being
// edited without exposing or masking secrets.
func (s *Service) ParseDefinition(raw []byte) (map[string]any, error) {
	if _, err := ParseDefinition(raw); err != nil {
		return nil, err
	}
	return ToDocument(raw)
}

// Create creates a managed endpoint and starts monitoring it. A push endpoint without token gets a generated one.
func (s *Service) Create(raw []byte, author string) (*Detail, error) {
	return s.create(raw, author, true)
}

// Update replaces the definition of a managed endpoint whose current version is expectedVersion
func (s *Service) Update(key string, raw []byte, expectedVersion int64, author string) (*Detail, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return nil, ErrCycleInProgress
	}
	defer end()
	statesMutex.Lock()
	defer statesMutex.Unlock()
	return s.update(key, raw, expectedVersion, author, "updated")
}

// SetEnabled enables or disables a managed endpoint whose current version is expectedVersion
func (s *Service) SetEnabled(key string, enabled bool, expectedVersion int64, author string) (*Detail, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return nil, ErrCycleInProgress
	}
	defer end()
	statesMutex.Lock()
	defer statesMutex.Unlock()
	state, err := s.managedState(key)
	if err != nil {
		return nil, err
	}
	document, err := ToDocument([]byte(state.Stored.Definition))
	if err != nil {
		return nil, err
	}
	if enabled {
		delete(document, "enabled")
	} else {
		document["enabled"] = false
	}
	raw, err := FromDocument(document)
	if err != nil {
		return nil, err
	}
	operation := "disabled"
	if enabled {
		operation = "enabled"
	}
	return s.update(key, raw, expectedVersion, author, operation)
}

// update must be called with a lifecycle change in progress and statesMutex held
func (s *Service) update(key string, raw []byte, expectedVersion int64, author, operation string) (*Detail, error) {
	state, err := s.managedState(key)
	if err != nil {
		return nil, err
	}
	if state.Stored.Version != expectedVersion {
		return nil, common.ErrManagedEndpointVersionMismatch
	}
	managedEndpointStore, ok := store.GetManagedEndpointStore()
	if !ok {
		return nil, ErrStorageNotSupported
	}
	submitted, err := ParseDefinition(raw)
	if err != nil {
		return nil, err
	}
	storedParsed, storedErr := ParseDefinition([]byte(state.Stored.Definition))
	if storedErr == nil && storedParsed.IsPush() != submitted.IsPush() {
		return nil, ErrTypeChanged
	}
	// The key of the definition is validated against the configuration file and the other managed endpoints
	prepared, err := s.prepare(raw, key)
	if err != nil {
		return nil, err
	}
	newKey := prepared.Key()
	// A changed name or group renames the endpoint, even when the key stays the same (e.g. "My API" and "my-api")
	renaming := newKey != key || (storedErr == nil && (storedParsed.Name() != submitted.Name() || storedParsed.Group() != submitted.Group()))
	// The data of the key of a managed endpoint in conflict belongs to the endpoint of the configuration file
	conflict := state.InConflict()
	// Generated before the endpoint starts being monitored (see State.Effective)
	effective, err := effectiveDefinition(&prepared.Parsed)
	if err != nil {
		return nil, err
	}
	previous := state.Parsed()
	wasMonitored := previous.IsValid() && watchdog.IsEndpointMonitored(key)
	// The monitoring is stopped before the transaction: with SQLite, an in-flight execution writing its result would
	// otherwise wait for the connection held by the transaction
	if wasMonitored {
		if err := watchdog.StopEndpoint(key, watchdog.SourceAdmin); err != nil && !errors.Is(err, watchdog.ErrEndpointNotMonitored) {
			return nil, fmt.Errorf("%w: %w", ErrApplyFailed, err)
		}
	}
	restartPrevious := func() {
		if wasMonitored {
			_ = startMonitoring(previous)
		}
	}
	// Uses the storage outside of the transaction, so it must run before it. The triggered alerts are still stored under
	// the old key until the rename is committed.
	switch {
	case !renaming:
		restoreTriggeredAlerts(&prepared.Parsed)
	case conflict:
	case storedErr == nil:
		restorePersistedTriggeredAlertsOfKey(&prepared.Parsed, storedParsed.Name(), storedParsed.Group())
	case previous.IsValid():
		restorePersistedTriggeredAlertsOfKey(&prepared.Parsed, previous.Name(), previous.Group())
	}
	var plan *KeyRenamePlan
	if renaming && !conflict && newKey != key {
		if participant := registeredKeyRenameParticipant(); participant != nil {
			if plan, err = participant.PrepareKeyRename(key, newKey, author); err != nil {
				restartPrevious()
				return nil, err
			}
		}
	}
	updated := &common.ManagedEndpoint{Key: newKey, Definition: string(prepared.Definition), UpdatedBy: author}
	started := false
	apply := func() error {
		if err := startMonitoring(&prepared.Parsed); err != nil {
			return err
		}
		started = prepared.IsEnabled()
		return nil
	}
	if renaming {
		rename := &common.ManagedEndpointRename{OldKey: key, Name: prepared.Name(), Group: prepared.Group(), MoveHistory: !conflict}
		if plan != nil {
			rename.StatusPages = plan.StatusPages
		}
		err = managedEndpointStore.RenameManagedEndpoint(updated, expectedVersion, rename, apply)
	} else {
		err = managedEndpointStore.UpdateManagedEndpoint(updated, expectedVersion, apply)
	}
	if err != nil {
		if started {
			_ = watchdog.StopEndpoint(newKey, watchdog.SourceAdmin)
		}
		restartPrevious()
		if plan != nil {
			plan.Discard()
		}
		return nil, err
	}
	if plan != nil {
		plan.Commit()
	}
	if !conflict {
		metrics.DeleteMetricsForEndpointKey(key)
	}
	newState := newStateFromPrepared(updated, prepared, effective)
	replaceState(key, newState)
	if renaming && newKey != key {
		// Fork: the time of the last push and the retries used of the old key are not carried to the new key
		watchdog.ForgetExternalEndpoint(key)
		liveupdates.Forget(key)
	}
	if renaming {
		logr.Infof("[managedendpoint.Update] Managed endpoint with key=%s renamed to key=%s by %s", key, newKey, auditAuthor(author))
	} else {
		logr.Infof("[managedendpoint.Update] Managed endpoint with key=%s %s by %s", key, operation, auditAuthor(author))
	}
	detail, err := adminDetail(newState)
	if err == nil && plan != nil {
		detail.AffectedConfigStatusPages = plan.AffectedConfigStatusPages
	}
	return detail, err
}

// Delete deletes a managed endpoint whose current version is expectedVersion. Unless it is in conflict with the
// configuration file, its monitoring is stopped and its data is deleted. It returns the number of alerts that were
// triggered; alert providers are not notified.
func (s *Service) Delete(key string, expectedVersion int64, author string) (int, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return 0, ErrCycleInProgress
	}
	defer end()
	statesMutex.Lock()
	defer statesMutex.Unlock()
	state, err := s.managedState(key)
	if err != nil {
		return 0, err
	}
	if state.Stored.Version != expectedVersion {
		return 0, common.ErrManagedEndpointVersionMismatch
	}
	managedEndpointStore, ok := store.GetManagedEndpointStore()
	if !ok {
		return 0, ErrStorageNotSupported
	}
	conflict := state.InConflict()
	previous := state.Parsed()
	wasMonitored := !conflict && previous.IsValid() && watchdog.IsEndpointMonitored(key)
	if wasMonitored {
		if err := watchdog.StopEndpoint(key, watchdog.SourceAdmin); err != nil && !errors.Is(err, watchdog.ErrEndpointNotMonitored) {
			return 0, fmt.Errorf("%w: %w", ErrApplyFailed, err)
		}
	}
	triggeredAlerts := 0
	if !conflict {
		// Safe to read: the monitoring of the endpoint has stopped
		for _, endpointAlert := range previous.alerts() {
			if endpointAlert.Triggered {
				triggeredAlerts++
			}
		}
	}
	if err := managedEndpointStore.DeleteManagedEndpoint(key, expectedVersion, !conflict, nil); err != nil {
		if wasMonitored {
			_ = startMonitoring(previous)
		}
		return 0, err
	}
	if !conflict {
		metrics.DeleteMetricsForEndpointKey(key)
	}
	removeState(key)
	watchdog.ForgetExternalEndpoint(key)
	liveupdates.Forget(key)
	logr.Infof("[managedendpoint.Delete] Managed endpoint with key=%s deleted by %s (triggered alerts: %d)", key, auditAuthor(author), triggeredAlerts)
	return triggeredAlerts, nil
}

// Test validates a definition and evaluates it once, without persisting results, sending alerts or publishing
// metrics. See Validate for the meaning of key. Push endpoints cannot be tested.
func (s *Service) Test(raw []byte, key string) (*TestResult, error) {
	select {
	case testSlots <- struct{}{}:
	default:
		return nil, ErrTooManyTests
	}
	defer func() { <-testSlots }()
	prepared, err := s.prepare(raw, key)
	if err != nil {
		return nil, err
	}
	if prepared.Push != nil {
		return nil, ErrPushNotTestable
	}
	ep := prepared.Endpoint
	if ep.ClientConfig != nil && (ep.ClientConfig.Timeout <= 0 || ep.ClientConfig.Timeout > testTimeout) {
		ep.ClientConfig.Timeout = testTimeout
	}
	result := ep.EvaluateHealth()
	ep.Close()
	testResult := &TestResult{
		Success:          result.Success,
		DurationMs:       result.Duration.Milliseconds(),
		HTTPStatus:       result.HTTPStatus,
		Errors:           result.Errors,
		ConditionResults: make([]TestConditionResult, 0, len(result.ConditionResults)),
	}
	for _, conditionResult := range result.ConditionResults {
		condition := conditionResult.Condition
		if len(condition) > maximumResolvedConditionLength {
			condition = condition[:maximumResolvedConditionLength] + "…"
		}
		testResult.ConditionResults = append(testResult.ConditionResults, TestConditionResult{Condition: condition, Success: conditionResult.Success})
	}
	return testResult, nil
}

// Metadata returns the configured alert types, the tunnels and the extra labels a managed endpoint can use
func (s *Service) Metadata() *Metadata {
	metadata := &Metadata{AlertTypes: []string{}, Tunnels: []string{}, ExtraLabels: []string{}}
	if s.cfg.Alerting != nil {
		value := reflect.ValueOf(s.cfg.Alerting).Elem()
		for i := 0; i < value.NumField(); i++ {
			field, fieldType := value.Field(i), value.Type().Field(i)
			if !fieldType.IsExported() || field.Kind() != reflect.Ptr || field.IsNil() {
				continue
			}
			if _, isProvider := field.Interface().(provider.AlertProvider); !isProvider {
				continue
			}
			if tag := strings.Split(fieldType.Tag.Get("yaml"), ",")[0]; len(tag) > 0 && tag != "-" {
				metadata.AlertTypes = append(metadata.AlertTypes, tag)
			}
		}
		sort.Strings(metadata.AlertTypes)
	}
	if s.cfg.Tunneling != nil {
		for name := range s.cfg.Tunneling.Tunnels {
			metadata.Tunnels = append(metadata.Tunnels, name)
		}
		sort.Strings(metadata.Tunnels)
	}
	if labels := metrics.RegisteredExtraLabels(); labels != nil {
		metadata.ExtraLabels = labels
	}
	return metadata
}

// managedState returns the state of a managed endpoint, or ErrReadOnly/ErrNotFound
func (s *Service) managedState(key string) (*State, error) {
	if state := Get(key); state != nil {
		return state, nil
	}
	if _, used := ConfigKeyOrigin(s.cfg, key); used {
		return nil, ErrReadOnly
	}
	return nil, ErrNotFound
}

// prepare restores the masked secrets of the stored definition of key, if any, and validates the definition
func (s *Service) prepare(raw []byte, key string) (*Prepared, error) {
	managedKeys := Keys()
	if len(key) > 0 {
		managedKeys = keysExcept(managedKeys, key)
		if state := Get(key); state != nil {
			if _, err := ParseDefinition(raw); err != nil {
				return nil, err
			}
			submitted, err := ToDocument(raw)
			if err != nil {
				return nil, err
			}
			stored, err := ToDocument([]byte(state.Stored.Definition))
			if err != nil {
				return nil, err
			}
			RestoreMaskedSecrets(submitted, stored)
			if raw, err = FromDocument(submitted); err != nil {
				return nil, err
			}
		}
	}
	return Prepare(raw, Context{
		Config:             s.cfg,
		ManagedKeys:        managedKeys,
		AllowedExtraLabels: metrics.RegisteredExtraLabels(),
		PushTokens:         s.pushTokens(key),
		IsPushKeyToken:     isPushKeyToken,
	})
}

// pushTokens returns the push tokens used by the configuration file and by the managed endpoints other than
// excludedKey, with a description of what uses them
func (s *Service) pushTokens(excludedKey string) map[string]string {
	tokens := make(map[string]string)
	for _, externalEndpoint := range s.cfg.ExternalEndpoints {
		if len(externalEndpoint.Token) > 0 {
			tokens[externalEndpoint.Token] = "an external endpoint of the configuration file"
		}
	}
	if s.cfg.Push != nil {
		for _, token := range s.cfg.Push.Tokens() {
			tokens[token] = "the push configuration of the configuration file"
		}
	}
	for _, state := range List() {
		if state.Stored.Key == excludedKey {
			continue
		}
		if token := state.Parsed().PushToken(); len(token) > 0 {
			tokens[token] = "another managed endpoint"
		}
	}
	return tokens
}

func isPushKeyToken(token string) bool {
	_, isKey := pushkey.Lookup(token)
	return isKey
}

// configItem describes an endpoint of the configuration file
func (s *Service) configItem(ep *endpoint.Endpoint) *Item {
	item := &Item{Key: ep.Key(), Name: ep.Name, Group: ep.Group, Type: string(ep.Type()), URL: maskURL(ep.URL), Enabled: ep.IsEnabled(), Source: SourceConfig}
	if ep.Interval > 0 {
		item.Interval = ep.Interval.String()
	}
	if s.cfg.Push != nil {
		for _, pushEndpoint := range s.cfg.Push.Endpoints {
			if pushEndpoint.Key == item.Key {
				item.AcceptsPush = true
			}
		}
	}
	return item
}

// configExternalItem describes an external endpoint of the configuration file, which receives push
func configExternalItem(externalEndpoint *endpoint.ExternalEndpoint) *Item {
	item := &Item{Key: externalEndpoint.Key(), Name: externalEndpoint.Name, Group: externalEndpoint.Group, Type: ItemTypePush, Enabled: externalEndpoint.IsEnabled(), Source: SourceConfig, AcceptsPush: true}
	if externalEndpoint.Heartbeat.Interval > 0 {
		item.Interval = externalEndpoint.Heartbeat.Interval.String()
	}
	return item
}

func adminItem(state *State) *Item {
	item := &Item{Key: state.Stored.Key, Source: SourceAdmin, Version: state.Stored.Version, UpdatedBy: state.Stored.UpdatedBy}
	createdAt, updatedAt := state.Stored.CreatedAt, state.Stored.UpdatedAt
	item.CreatedAt, item.UpdatedAt = &createdAt, &updatedAt
	parsed := state.Parsed()
	if !parsed.IsValid() {
		// In conflict or invalid: describe it from its definition, as far as possible
		if storedParsed, err := ParseDefinition([]byte(state.Stored.Definition)); err == nil {
			parsed = storedParsed
		}
	}
	switch {
	case parsed.Push != nil:
		item.Name, item.Group, item.Type, item.Enabled, item.AcceptsPush = parsed.Push.Name, parsed.Push.Group, ItemTypePush, parsed.Push.IsEnabled(), true
		if parsed.Push.Heartbeat.Interval > 0 {
			item.Interval = parsed.Push.Heartbeat.Interval.String()
		}
	case parsed.Endpoint != nil:
		ep := parsed.Endpoint
		item.Name, item.Group, item.URL, item.Enabled, item.AcceptsPush = ep.Name, ep.Group, maskURL(ep.URL), ep.IsEnabled(), parsed.ReceivesPush()
		if len(ep.URL) > 0 {
			item.Type = string(ep.Type())
		}
		if ep.Interval > 0 {
			item.Interval = ep.Interval.String()
		}
	}
	if state.InConflict() {
		item.Conflict, item.ConflictOrigin = true, state.ConflictOrigin
		item.Enabled = false
	} else if state.Err != nil {
		item.Error = state.Err.Error()
		item.Enabled = false
	}
	return item
}

func adminDetail(state *State) (*Detail, error) {
	definition, err := maskedDefinition([]byte(state.Stored.Definition))
	if err != nil {
		return nil, err
	}
	detail := &Detail{Item: *adminItem(state), Definition: definition}
	if state.Effective != nil {
		if detail.Effective, err = maskedDefinition(state.Effective); err != nil {
			return nil, err
		}
	}
	parsed := state.Parsed()
	if !parsed.IsValid() {
		if storedParsed, err := ParseDefinition([]byte(state.Stored.Definition)); err == nil {
			parsed = storedParsed
		}
	}
	detail.PushToken = parsed.PushToken()
	return detail, nil
}

func maskedDefinition(definition []byte) (*Definition, error) {
	document, err := ToDocument(definition)
	if err != nil {
		return nil, err
	}
	MaskSecrets(document)
	yamlDefinition, err := FromDocument(document)
	if err != nil {
		return nil, err
	}
	return &Definition{YAML: string(yamlDefinition), JSON: document}, nil
}

func auditAuthor(author string) string {
	if len(author) == 0 {
		return "unknown"
	}
	return author
}
