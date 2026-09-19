package statuspage

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	pageconfig "gatus/v5/internal/config/statuspage"
	"gatus/v5/internal/lifecycle"
	"gatus/v5/internal/storage/store"
	"gatus/v5/internal/storage/store/common"
	"github.com/TwiN/logr"
	"gopkg.in/yaml.v3"
)

// warningTypeCharts is the type of the warning of a page with the deprecated charts, which are ignored
const warningTypeCharts = "charts"

var (
	// ErrReadOnly is returned when trying to change a status page defined in the configuration file
	ErrReadOnly = errors.New("the status page is defined in the configuration file and cannot be changed through the administration")

	// ErrCycleInProgress is returned when a start or configuration reload is in progress
	ErrCycleInProgress = errors.New("a start or configuration reload is in progress, try again later")

	// ErrSlugChanged is returned when an update changes the slug of the status page
	ErrSlugChanged = errors.New("the slug of a status page cannot be changed")

	// ErrSlugInUse is returned when creating a status page whose slug is already used
	ErrSlugInUse = errors.New("the slug is already used by another status page")

	// ErrStorageNotSupported is returned when the storage does not support managed status pages
	ErrStorageNotSupported = errors.New("the storage does not support managed status pages")

	// ErrExposureQueryRequired is returned when the exposure of an endpoint is requested without group and key
	ErrExposureQueryRequired = errors.New("the group or the key of the endpoint is required")

	// getManagedStatusPageStore is replaced in tests
	getManagedStatusPageStore = store.GetManagedStatusPageStore

	// previewSemaphore limits the previews of the administration, which are not cached, without taking the slots of the
	// public status pages
	previewSemaphore = make(chan struct{}, 1)
)

// Item summarizes a status page for the administration list
type Item struct {
	Slug           string `json:"slug"`
	Title          string `json:"title,omitempty"`
	Origin         Origin `json:"origin"`
	Enabled        bool   `json:"enabled"`
	Published      bool   `json:"published"`
	Conflict       bool   `json:"conflict"`
	ConflictOrigin string `json:"conflictOrigin,omitempty"`
	Error          string `json:"error,omitempty"`
	Endpoints      int    `json:"endpoints"`
	// RequiresLogin is whether the page has a login of its own (fork)
	RequiresLogin bool       `json:"requiresLogin"`
	Path          string     `json:"path"`
	Version       int64      `json:"version,omitempty"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     string     `json:"updatedBy,omitempty"`
}

// Listing is the administration list of the status pages
type Listing struct {
	// PublicationEnabled is status-pages.enabled: when false, no page is published
	PublicationEnabled bool `json:"publicationEnabled"`

	// ManagedUnavailable is whether the managed status pages could not be loaded
	ManagedUnavailable bool `json:"managedUnavailable"`

	// SharedRateLimitWarning is the untrusted proxy IP address from which every visitor seems to come, if any
	SharedRateLimitWarning string `json:"sharedRateLimitWarning,omitempty"`

	StatusPages []*Item `json:"statusPages"`
}

// Detail is a status page with its definition
type Detail struct {
	Item

	// Definition is the normalized definition, or nil if the stored definition is invalid
	Definition *pageconfig.Page `json:"definition"`

	// YAML is the stored definition for managed status pages, or the definition from the configuration file
	YAML string `json:"yaml"`
}

// Warning is a group or an endpoint key selected by a page without match
type Warning struct {
	// Type is "group" or "endpoint"
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Validation is the result of a successful validation
type Validation struct {
	Definition *pageconfig.Page `json:"definition"`
	Warnings   []Warning        `json:"warnings"`
	Endpoints  int              `json:"endpoints"`
}

// Options lists what a status page can select
type Options struct {
	Groups    []GroupOption    `json:"groups"`
	Endpoints []EndpointOption `json:"endpoints"`
}

// GroupOption is a group with the number of endpoints that can be published
type GroupOption struct {
	Name      string `json:"name"`
	Endpoints int    `json:"endpoints"`
}

// EndpointOption is an endpoint that can be published
type EndpointOption struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

// Exposure lists the status pages on which an endpoint would appear
type Exposure struct {
	StatusPages []ExposureItem `json:"statusPages"`
}

// ExposureItem is a status page on which an endpoint would appear, by group or by key
type ExposureItem struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Origin    Origin `json:"origin"`
	Published bool   `json:"published"`
	Reason    string `json:"reason"`
}

// Service administers the status pages
type Service struct{}

// NewService returns the administration service of the status pages
func NewService() *Service {
	return &Service{}
}

// List returns the status pages of the configuration file and the managed status pages
func (s *Service) List() *Listing {
	refs := Endpoints()
	states := List()
	listing := &Listing{PublicationEnabled: IsEnabled(), ManagedUnavailable: IsManagedUnavailable(), StatusPages: make([]*Item, 0, len(states))}
	listing.SharedRateLimitWarning, _ = SharedRateLimitWarning()
	for _, state := range states {
		listing.StatusPages = append(listing.StatusPages, newItem(state, refs))
	}
	return listing
}

// Get returns the status page with the given slug, the managed one when both origins use the slug
func (s *Service) Get(slug string) (*Detail, error) {
	state := findState(slug)
	if state == nil {
		return nil, ErrPageNotFound
	}
	return newDetail(state)
}

// Options returns the groups and the endpoints that a status page can select
func (s *Service) Options() *Options {
	refs := Endpoints()
	counts := make(map[string]int)
	options := &Options{Groups: []GroupOption{}, Endpoints: make([]EndpointOption, 0, len(refs))}
	for _, ref := range refs {
		if group := pageconfig.NormalizeGroup(ref.Group); len(group) > 0 {
			counts[group]++
		}
		options.Endpoints = append(options.Endpoints, EndpointOption{Key: ref.Key, Name: ref.Name, Group: ref.Group})
	}
	for group, count := range counts {
		options.Groups = append(options.Groups, GroupOption{Name: group, Endpoints: count})
	}
	sort.Slice(options.Groups, func(i, j int) bool {
		return options.Groups[i].Name < options.Groups[j].Name
	})
	sort.Slice(options.Endpoints, func(i, j int) bool {
		a, b := options.Endpoints[i], options.Endpoints[j]
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		return a.Key < b.Key
	})
	return options
}

// Exposure returns the status pages on which an endpoint with the given group or key would appear
func (s *Service) Exposure(group, key string) (*Exposure, error) {
	group, key = pageconfig.NormalizeGroup(group), strings.ToLower(strings.TrimSpace(key))
	if len(group) == 0 && len(key) == 0 {
		return nil, ErrExposureQueryRequired
	}
	exposure := &Exposure{StatusPages: []ExposureItem{}}
	for _, state := range List() {
		if state.Page == nil {
			continue
		}
		var reason string
		switch {
		case len(group) > 0 && slices.Contains(state.Page.Groups, group):
			reason = "group"
		case len(key) > 0 && (slices.Contains(state.Page.Endpoints, key) || slices.Contains(state.Page.Featured, key)):
			reason = "key"
		default:
			continue
		}
		exposure.StatusPages = append(exposure.StatusPages, ExposureItem{
			Slug:      state.Slug,
			Title:     state.Page.Title,
			Origin:    state.Origin,
			Published: IsEnabled() && state.IsPublished(),
			Reason:    reason,
		})
	}
	return exposure, nil
}

// Validate validates a definition without persisting it. When slug is not empty, the definition is validated as an
// update of that managed status page.
func (s *Service) Validate(raw []byte, slug string) (*Validation, error) {
	page, err := parseSubmitted(raw, slug)
	if err != nil {
		return nil, err
	}
	refs := Endpoints()
	// Fork: like every read of the administration, the validation answers with the hash of the credential masked
	return &Validation{Definition: maskCredential(page), Warnings: selectionWarnings(page, refs), Endpoints: len(Select(page, refs).Keys())}, nil
}

// Create creates a managed status page. Without enabled, it is created disabled.
func (s *Service) Create(raw []byte, author string) (*Detail, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return nil, ErrCycleInProgress
	}
	defer end()
	mutex.Lock()
	defer mutex.Unlock()
	managedStatusPageStore, ok := getManagedStatusPageStore()
	if !ok {
		return nil, ErrStorageNotSupported
	}
	page, err := parseSubmitted(raw, "")
	if err != nil {
		return nil, err
	}
	definition, err := yaml.Marshal(page)
	if err != nil {
		return nil, err
	}
	stored := &common.ManagedStatusPage{Slug: page.Slug, Definition: string(definition), UpdatedBy: author}
	if err := managedStatusPageStore.CreateManagedStatusPage(stored, nil); err != nil {
		return nil, err
	}
	state := publishManaged(stored)
	logr.Infof("[statuspage.Create] Managed status page with slug=%s created by %s", page.Slug, auditAuthor(author))
	return newDetail(state)
}

// Update replaces the definition of a managed status page whose current version is expectedVersion
func (s *Service) Update(slug string, raw []byte, expectedVersion int64, author string) (*Detail, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return nil, ErrCycleInProgress
	}
	defer end()
	mutex.Lock()
	defer mutex.Unlock()
	managedStatusPageStore, err := managedStoreForChange(slug, expectedVersion)
	if err != nil {
		return nil, err
	}
	page, err := parseSubmitted(raw, slug)
	if err != nil {
		return nil, err
	}
	return save(managedStatusPageStore, page, expectedVersion, author, "updated")
}

// SetEnabled enables or disables a managed status page whose current version is expectedVersion
func (s *Service) SetEnabled(slug string, enabled bool, expectedVersion int64, author string) (*Detail, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return nil, ErrCycleInProgress
	}
	defer end()
	mutex.Lock()
	defer mutex.Unlock()
	managedStatusPageStore, err := managedStoreForChange(slug, expectedVersion)
	if err != nil {
		return nil, err
	}
	page, err := Parse([]byte(current.Load().managedStates[slug].Stored.Definition))
	if err != nil {
		return nil, err
	}
	page.Enabled = &enabled
	operation := "disabled"
	if enabled {
		operation = "enabled"
	}
	return save(managedStatusPageStore, page, expectedVersion, author, operation)
}

// Delete deletes a managed status page whose current version is expectedVersion
func (s *Service) Delete(slug string, expectedVersion int64, author string) error {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return ErrCycleInProgress
	}
	defer end()
	mutex.Lock()
	defer mutex.Unlock()
	managedStatusPageStore, err := managedStoreForChange(slug, expectedVersion)
	if err != nil {
		return err
	}
	if err := managedStatusPageStore.DeleteManagedStatusPage(slug, expectedVersion, nil); err != nil {
		return err
	}
	// Unpublished only after the commit
	next := cloneCurrentSnapshot()
	delete(next.managedStates, slug)
	publish(next)
	_ = publicCache.DeleteKeysByPattern(slug + "|*")
	logr.Infof("[statuspage.Delete] Managed status page with slug=%s deleted by %s", slug, auditAuthor(author))
	return nil
}

// Preview returns the public payload of any status page, including disabled, in conflict and configuration file ones,
// without cache and without rate limit
func (s *Service) Preview(slug string) ([]byte, error) {
	state := findState(slug)
	if state == nil {
		return nil, ErrPageNotFound
	}
	page := state.Page
	if page == nil {
		var err error
		if page, err = Parse([]byte(state.Stored.Definition)); err != nil {
			return nil, err
		}
	}
	release, acquired := acquireSlot(previewSemaphore)
	if !acquired {
		return nil, ErrPageUnavailable
	}
	defer release()
	maximumResults := MaximumPublicResults
	if snap := current.Load(); snap != nil {
		maximumResults = snap.maximumResults
	}
	body, err := assemble(page, maximumResults, time.Now())
	if err != nil {
		logr.Errorf("[statuspage.Preview] Failed to assemble status page with slug=%s: %s", slug, err.Error())
		return nil, ErrPageUnavailable
	}
	return body, nil
}

// parseSubmitted parses a submitted definition, in which a missing enabled means disabled. slug is the managed status
// page being updated, or empty for a creation, in which case the slug must not be used.
func parseSubmitted(raw []byte, slug string) (*pageconfig.Page, error) {
	// Fork: the plaintext password of the login of the page becomes a bcrypt hash before anything is parsed, and a
	// password that was not sent keeps the stored hash
	resolved, err := resolveSubmittedCredential(raw, storedPasswordHash(slug))
	if err != nil {
		return nil, err
	}
	page, err := Parse(resolved)
	if err != nil {
		return nil, err
	}
	if page.Enabled == nil {
		disabled := false
		page.Enabled = &disabled
	}
	if len(slug) > 0 {
		if page.Slug != slug {
			return nil, ErrSlugChanged
		}
		return page, nil
	}
	if snap := current.Load(); snap != nil {
		if _, used := snap.configStates[page.Slug]; used {
			return nil, fmt.Errorf("%w: %s is used by the configuration file", ErrSlugInUse, page.Slug)
		}
		if _, used := snap.managedStates[page.Slug]; used {
			return nil, fmt.Errorf("%w: %s is used by another managed status page", ErrSlugInUse, page.Slug)
		}
	}
	return page, nil
}

// managedStoreForChange checks that the managed status page exists at expectedVersion and returns the store. mutex must
// be held.
func managedStoreForChange(slug string, expectedVersion int64) (store.ManagedStatusPageStore, error) {
	var state *State
	if snap := current.Load(); snap != nil {
		state = snap.managedStates[slug]
		if state == nil {
			if _, exists := snap.configStates[slug]; exists {
				return nil, ErrReadOnly
			}
		}
	}
	if state == nil {
		return nil, ErrPageNotFound
	}
	if state.Stored.Version != expectedVersion {
		return nil, common.ErrManagedStatusPageVersionMismatch
	}
	managedStatusPageStore, ok := getManagedStatusPageStore()
	if !ok {
		return nil, ErrStorageNotSupported
	}
	return managedStatusPageStore, nil
}

// save persists the definition of a managed status page and publishes it after the commit. mutex must be held.
func save(managedStatusPageStore store.ManagedStatusPageStore, page *pageconfig.Page, expectedVersion int64, author, operation string) (*Detail, error) {
	definition, err := yaml.Marshal(page)
	if err != nil {
		return nil, err
	}
	updated := &common.ManagedStatusPage{Slug: page.Slug, Definition: string(definition), UpdatedBy: author}
	if err := managedStatusPageStore.UpdateManagedStatusPage(updated, expectedVersion, nil); err != nil {
		return nil, err
	}
	state := publishManaged(updated)
	logr.Infof("[statuspage.Update] Managed status page with slug=%s %s by %s", page.Slug, operation, auditAuthor(author))
	return newDetail(state)
}

// publishManaged publishes, with a new revision, the state of a managed status page whose change was committed. mutex
// must be held.
func publishManaged(stored *common.ManagedStatusPage) *State {
	next := cloneCurrentSnapshot()
	state := newManagedState(stored, next.configStates)
	next.managedStates[stored.Slug] = state
	publish(next)
	_ = publicCache.DeleteKeysByPattern(stored.Slug + "|*")
	return state
}

// cloneCurrentSnapshot returns a copy of the current snapshot, to be changed and published. mutex must be held.
func cloneCurrentSnapshot() *snapshot {
	next := &snapshot{enabled: true, maximumResults: MaximumPublicResults, configStates: make(map[string]*State), managedStates: make(map[string]*State)}
	if snap := current.Load(); snap != nil {
		next.generation, next.maximumResults, next.enabled = snap.generation, snap.maximumResults, snap.enabled
		next.managedUnavailable, next.configEndpoints = snap.managedUnavailable, snap.configEndpoints
		for slug, state := range snap.configStates {
			next.configStates[slug] = state
		}
		for slug, state := range snap.managedStates {
			next.managedStates[slug] = state
		}
	}
	return next
}

// findState returns the state of the status page with the given slug, the managed one when both origins use the slug
func findState(slug string) *State {
	snap := current.Load()
	if snap == nil {
		return nil
	}
	if state := snap.managedStates[slug]; state != nil {
		return state
	}
	return snap.configStates[slug]
}

// selectionWarnings returns the groups, endpoint keys and featured endpoint keys selected by the page without match
// among refs, and a warning of type charts when the page still has the deprecated charts
func selectionWarnings(page *pageconfig.Page, refs []EndpointRef) []Warning {
	groups := make(map[string]struct{}, len(refs))
	keys := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		groups[pageconfig.NormalizeGroup(ref.Group)] = struct{}{}
		keys[ref.Key] = struct{}{}
	}
	warnings := []Warning{}
	for _, group := range page.Groups {
		if _, exists := groups[group]; !exists {
			warnings = append(warnings, Warning{Type: "group", Value: group})
		}
	}
	for _, key := range page.Endpoints {
		if _, exists := keys[key]; !exists {
			warnings = append(warnings, Warning{Type: "endpoint", Value: key})
		}
	}
	for _, key := range page.Featured {
		if _, exists := keys[key]; !exists {
			warnings = append(warnings, Warning{Type: "featured", Value: key})
		}
	}
	if len(page.Charts) > 0 {
		warnings = append(warnings, Warning{Type: warningTypeCharts, Value: strings.Join(page.Charts, ", ")})
	}
	return warnings
}

func newItem(state *State, refs []EndpointRef) *Item {
	item := &Item{Slug: state.Slug, Origin: state.Origin, Path: "/status/" + state.Slug, Published: IsEnabled() && state.IsPublished()}
	page := state.Page
	if page == nil && state.Stored != nil {
		// In conflict or invalid: describe it from its definition, as far as possible
		page, _ = Parse([]byte(state.Stored.Definition))
	}
	if page != nil {
		item.Title = page.Title
		item.Enabled = page.IsEnabled()
		if state.Origin == OriginAdmin {
			item.Enabled = page.Enabled != nil && *page.Enabled
		}
		item.Endpoints = len(Select(page, refs).Keys())
		item.RequiresLogin = page.RequiresLogin()
	}
	if stored := state.Stored; stored != nil {
		createdAt, updatedAt := stored.CreatedAt, stored.UpdatedAt
		item.Version, item.CreatedAt, item.UpdatedAt, item.UpdatedBy = stored.Version, &createdAt, &updatedAt, stored.UpdatedBy
	}
	if state.InConflict() {
		item.Conflict, item.ConflictOrigin = true, state.ConflictOrigin
	}
	if state.Err != nil {
		item.Error = state.Err.Error()
	}
	return item
}

// newDetail describes a status page with its definition. Fork: the hash of the credential of the page is masked, in
// the definition and in the YAML, and submitting the mask back keeps the stored hash.
func newDetail(state *State) (*Detail, error) {
	detail := &Detail{Item: *newItem(state, Endpoints())}
	if state.Stored != nil {
		detail.YAML = maskCredentialInDefinition(state.Stored.Definition)
		if page, err := Parse([]byte(state.Stored.Definition)); err == nil {
			detail.Definition = maskCredential(page)
		}
		return detail, nil
	}
	definition, err := yaml.Marshal(maskCredential(state.Page))
	if err != nil {
		return nil, err
	}
	detail.YAML, detail.Definition = string(definition), maskCredential(state.Page)
	return detail, nil
}

func auditAuthor(author string) string {
	if len(author) == 0 {
		return "unknown"
	}
	return author
}
