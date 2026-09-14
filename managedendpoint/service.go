package managedendpoint

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/provider"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/lifecycle"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/watchdog"
	"github.com/TwiN/logr"
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

	// ErrKeyChanged is returned when an update changes the name or the group of the endpoint
	ErrKeyChanged = errors.New("the name and the group of a managed endpoint cannot be changed")

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
	Key            string     `json:"key"`
	Name           string     `json:"name"`
	Group          string     `json:"group"`
	Type           string     `json:"type,omitempty"`
	URL            string     `json:"url,omitempty"`
	Interval       string     `json:"interval,omitempty"`
	Enabled        bool       `json:"enabled"`
	Source         string     `json:"source"`
	Conflict       bool       `json:"conflict"`
	ConflictOrigin string     `json:"conflictOrigin,omitempty"`
	Error          string     `json:"error,omitempty"`
	Version        int64      `json:"version,omitempty"`
	CreatedAt      *time.Time `json:"createdAt,omitempty"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy      string     `json:"updatedBy,omitempty"`
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

// List returns the endpoints of the configuration file and the managed endpoints, ordered by key
func (s *Service) List() []*Item {
	items := make([]*Item, 0, len(s.cfg.Endpoints))
	for _, ep := range s.cfg.Endpoints {
		items = append(items, configItem(ep))
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
		return &Detail{Item: *configItem(ep), Definition: definition, Effective: definition}, nil
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
	effective, err := endpointDefinition(prepared.Endpoint)
	if err != nil {
		return nil, err
	}
	return &Validation{Definition: definition, Effective: effective}, nil
}

// ParseDefinition decodes a definition strictly and returns it as a document with the YAML keys. It neither validates
// the endpoint nor reads stored data, so it only returns what was submitted: it is used to convert a definition being
// edited without exposing or masking secrets.
func (s *Service) ParseDefinition(raw []byte) (map[string]any, error) {
	if _, err := Parse(raw); err != nil {
		return nil, err
	}
	return ToDocument(raw)
}

// Create creates a managed endpoint and starts monitoring it
func (s *Service) Create(raw []byte, author string) (*Detail, error) {
	end, ok := lifecycle.TryBeginChange()
	if !ok {
		return nil, ErrCycleInProgress
	}
	defer end()
	statesMutex.Lock()
	defer statesMutex.Unlock()
	managedEndpointStore, ok := store.GetManagedEndpointStore()
	if !ok {
		return nil, ErrStorageNotSupported
	}
	prepared, err := s.prepare(raw, "")
	if err != nil {
		return nil, err
	}
	key := prepared.Endpoint.Key()
	// Generated before the endpoint starts being monitored (see State.Effective)
	effective, err := Effective(prepared.Endpoint)
	if err != nil {
		return nil, err
	}
	// Uses the storage outside of the transaction, so it must run before it
	watchdog.RestorePersistedTriggeredAlerts(prepared.Endpoint)
	stored := &common.ManagedEndpoint{Key: key, Definition: string(prepared.Definition), UpdatedBy: author}
	started := false
	err = managedEndpointStore.CreateManagedEndpoint(stored, func() error {
		if err := startMonitoring(prepared.Endpoint); err != nil {
			return err
		}
		started = prepared.Endpoint.IsEnabled()
		return nil
	})
	if err != nil {
		if started {
			_ = watchdog.StopEndpoint(key, watchdog.SourceAdmin)
		}
		return nil, err
	}
	state := &State{Stored: stored, Endpoint: prepared.Endpoint, Effective: effective}
	putState(state)
	logr.Infof("[managedendpoint.Create] Managed endpoint with key=%s created by %s", key, auditAuthor(author))
	return adminDetail(state)
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
	submitted, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	if storedEndpoint, err := Parse([]byte(state.Stored.Definition)); err == nil && (storedEndpoint.Name != submitted.Name || storedEndpoint.Group != submitted.Group) {
		return nil, ErrKeyChanged
	}
	prepared, err := s.prepare(raw, key)
	if err != nil {
		return nil, err
	}
	// Generated before the endpoint starts being monitored (see State.Effective)
	effective, err := Effective(prepared.Endpoint)
	if err != nil {
		return nil, err
	}
	previousEndpoint := state.Endpoint
	wasMonitored := previousEndpoint != nil && watchdog.IsEndpointMonitored(key)
	// The monitoring is stopped before the transaction: with SQLite, an in-flight execution writing its result would
	// otherwise wait for the connection held by the transaction
	if wasMonitored {
		if err := watchdog.StopEndpoint(key, watchdog.SourceAdmin); err != nil && !errors.Is(err, watchdog.ErrEndpointNotMonitored) {
			return nil, fmt.Errorf("%w: %w", ErrApplyFailed, err)
		}
	}
	watchdog.RestorePersistedTriggeredAlerts(prepared.Endpoint)
	updated := &common.ManagedEndpoint{Key: key, Definition: string(prepared.Definition), UpdatedBy: author}
	started := false
	err = managedEndpointStore.UpdateManagedEndpoint(updated, expectedVersion, func() error {
		if err := startMonitoring(prepared.Endpoint); err != nil {
			return err
		}
		started = prepared.Endpoint.IsEnabled()
		return nil
	})
	if err != nil {
		if started {
			_ = watchdog.StopEndpoint(key, watchdog.SourceAdmin)
		}
		if wasMonitored {
			_ = watchdog.StartEndpoint(previousEndpoint, watchdog.SourceAdmin)
		}
		return nil, err
	}
	metrics.DeleteMetricsForEndpointKey(key)
	newState := &State{Stored: updated, Endpoint: prepared.Endpoint, Effective: effective}
	putState(newState)
	logr.Infof("[managedendpoint.Update] Managed endpoint with key=%s %s by %s", key, operation, auditAuthor(author))
	return adminDetail(newState)
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
	wasMonitored := !conflict && state.Endpoint != nil && watchdog.IsEndpointMonitored(key)
	if wasMonitored {
		if err := watchdog.StopEndpoint(key, watchdog.SourceAdmin); err != nil && !errors.Is(err, watchdog.ErrEndpointNotMonitored) {
			return 0, fmt.Errorf("%w: %w", ErrApplyFailed, err)
		}
	}
	triggeredAlerts := 0
	if !conflict && state.Endpoint != nil {
		// Safe to read: the monitoring of the endpoint has stopped
		for _, endpointAlert := range state.Endpoint.Alerts {
			if endpointAlert.Triggered {
				triggeredAlerts++
			}
		}
	}
	if err := managedEndpointStore.DeleteManagedEndpoint(key, expectedVersion, !conflict, nil); err != nil {
		if wasMonitored {
			_ = watchdog.StartEndpoint(state.Endpoint, watchdog.SourceAdmin)
		}
		return 0, err
	}
	if !conflict {
		metrics.DeleteMetricsForEndpointKey(key)
	}
	removeState(key)
	logr.Infof("[managedendpoint.Delete] Managed endpoint with key=%s deleted by %s (triggered alerts: %d)", key, auditAuthor(author), triggeredAlerts)
	return triggeredAlerts, nil
}

// Test validates a definition and evaluates it once, without persisting results, sending alerts or publishing
// metrics. See Validate for the meaning of key.
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
			if _, err := Parse(raw); err != nil {
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
	return Prepare(raw, Context{Config: s.cfg, ManagedKeys: managedKeys, AllowedExtraLabels: metrics.RegisteredExtraLabels()})
}

func startMonitoring(ep *endpoint.Endpoint) error {
	if !ep.IsEnabled() {
		return nil
	}
	if err := watchdog.StartEndpoint(ep, watchdog.SourceAdmin); err != nil {
		return fmt.Errorf("%w: %w", ErrApplyFailed, err)
	}
	return nil
}

func configItem(ep *endpoint.Endpoint) *Item {
	item := &Item{Key: ep.Key(), Name: ep.Name, Group: ep.Group, Type: string(ep.Type()), URL: maskURL(ep.URL), Enabled: ep.IsEnabled(), Source: SourceConfig}
	if ep.Interval > 0 {
		item.Interval = ep.Interval.String()
	}
	return item
}

func adminItem(state *State) *Item {
	item := &Item{Key: state.Stored.Key, Source: SourceAdmin, Version: state.Stored.Version, UpdatedBy: state.Stored.UpdatedBy}
	createdAt, updatedAt := state.Stored.CreatedAt, state.Stored.UpdatedAt
	item.CreatedAt, item.UpdatedAt = &createdAt, &updatedAt
	ep := state.Endpoint
	if ep == nil {
		// In conflict or invalid: describe it from its definition, as far as possible
		ep, _ = Parse([]byte(state.Stored.Definition))
	}
	if ep != nil {
		item.Name, item.Group, item.URL, item.Enabled = ep.Name, ep.Group, maskURL(ep.URL), ep.IsEnabled()
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
	return detail, nil
}

func endpointDefinition(ep *endpoint.Endpoint) (*Definition, error) {
	effective, err := Effective(ep)
	if err != nil {
		return nil, err
	}
	return maskedDefinition(effective)
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
