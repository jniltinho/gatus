package managedendpoint

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
	"gatus/v5/watchdog"
	"github.com/TwiN/logr"
)

// State is the runtime state of a persisted managed endpoint
type State struct {
	// Stored is the persisted managed endpoint
	Stored *common.ManagedEndpoint

	// Endpoint is the validated endpoint, or nil if the managed endpoint is in conflict or invalid
	Endpoint *endpoint.Endpoint

	// ConflictOrigin describes what uses the same key in the configuration file, when the managed endpoint is in conflict
	ConflictOrigin string

	// Err is the validation error, when the managed endpoint is invalid
	Err error

	// Effective is the YAML definition of Endpoint with its default values, generated before the endpoint started being
	// monitored. Monitored endpoints must not be serialized: copying their structs races with the watchdog (e.g. the
	// HTTP client created lazily by the first evaluation).
	Effective []byte
}

// InConflict returns whether the key of the managed endpoint is used by the configuration file
func (state *State) InConflict() bool {
	return len(state.ConflictOrigin) > 0
}

var (
	// states is an immutable snapshot of the managed endpoints, replaced on every change (copy-on-write)
	states atomic.Pointer[map[string]*State]

	// statesMutex serializes the changes of the snapshot
	statesMutex sync.Mutex

	// configDefinitions holds the effective YAML definitions of the endpoints of the configuration file, generated
	// before they started being monitored (see State.Effective)
	configDefinitions atomic.Pointer[map[string][]byte]
)

// Load reads the managed endpoints from the storage, validates them against cfg and publishes their state, replacing
// the previous one. It returns the keys of every persisted managed endpoint, valid or not, so that their history is
// preserved. With a storage that does not support managed endpoints (memory), there is nothing to load.
//
// It does not depend on admin.enabled: disabling the administration does not stop the managed endpoints.
func Load(cfg *config.Config) ([]string, error) {
	statesMutex.Lock()
	defer statesMutex.Unlock()
	definitions := make(map[string][]byte, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		if effective, err := Effective(ep); err == nil {
			definitions[ep.Key()] = effective
		}
	}
	configDefinitions.Store(&definitions)
	managedEndpointStore, ok := store.GetManagedEndpointStore()
	if !ok {
		publish(map[string]*State{})
		return nil, nil
	}
	storedEndpoints, err := managedEndpointStore.ListManagedEndpoints()
	if err != nil {
		publish(map[string]*State{})
		return nil, err
	}
	keys := make([]string, 0, len(storedEndpoints))
	for _, stored := range storedEndpoints {
		keys = append(keys, stored.Key)
	}
	allowedExtraLabels := cfg.GetUniqueExtraMetricLabels()
	loaded := make(map[string]*State, len(storedEndpoints))
	for _, stored := range storedEndpoints {
		loaded[stored.Key] = newState(stored, cfg, keysExcept(keys, stored.Key), allowedExtraLabels)
	}
	publish(loaded)
	if len(loaded) > 0 {
		logr.Infof("[managedendpoint.Load] Loaded %d managed endpoints", len(loaded))
	}
	return keys, nil
}

func newState(stored *common.ManagedEndpoint, cfg *config.Config, managedKeys, allowedExtraLabels []string) *State {
	state := &State{Stored: stored}
	if origin, used := ConfigKeyOrigin(cfg, stored.Key); used {
		state.ConflictOrigin = origin
		logr.Warnf("[managedendpoint.Load] Managed endpoint with key=%s is not monitored because its key is used by %s", stored.Key, origin)
		return state
	}
	prepared, err := Prepare([]byte(stored.Definition), Context{Config: cfg, ManagedKeys: managedKeys, AllowedExtraLabels: allowedExtraLabels})
	if err == nil && prepared.Endpoint.Key() != stored.Key {
		err = fmt.Errorf("%w: the key of the definition (%s) does not match the stored key", ErrInvalidDefinition, prepared.Endpoint.Key())
	}
	var effective []byte
	if err == nil {
		effective, err = Effective(prepared.Endpoint)
	}
	if err != nil {
		state.Err = err
		logr.Errorf("[managedendpoint.Load] Managed endpoint with key=%s is not monitored because it is invalid: %s", stored.Key, err.Error())
		return state
	}
	state.Endpoint, state.Effective = prepared.Endpoint, effective
	watchdog.RestorePersistedTriggeredAlerts(prepared.Endpoint)
	return state
}

// StartMonitoring starts monitoring every valid and enabled managed endpoint. It must be called after watchdog.Monitor,
// while the lifecycle cycle is still in progress.
func StartMonitoring() {
	for _, state := range List() {
		if state.Endpoint == nil || !state.Endpoint.IsEnabled() {
			continue
		}
		// Same spacing as watchdog.Monitor, to prevent many requests from running at the same time
		time.Sleep(222 * time.Millisecond)
		if err := watchdog.StartEndpoint(state.Endpoint, watchdog.SourceAdmin); err != nil {
			logr.Errorf("[managedendpoint.StartMonitoring] Failed to start monitoring managed endpoint with key=%s: %s", state.Stored.Key, err.Error())
		}
	}
}

// List returns the state of every managed endpoint, ordered by key
func List() []*State {
	snapshot := states.Load()
	if snapshot == nil {
		return nil
	}
	list := make([]*State, 0, len(*snapshot))
	for _, state := range *snapshot {
		list = append(list, state)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Stored.Key < list[j].Stored.Key
	})
	return list
}

// Get returns the state of the managed endpoint with the given key, or nil
func Get(key string) *State {
	snapshot := states.Load()
	if snapshot == nil {
		return nil
	}
	return (*snapshot)[key]
}

// EndpointByKey returns the monitored endpoint of the managed endpoint with the given key, ignoring case, or nil if
// there is none or if it is in conflict or invalid
func EndpointByKey(key string) *endpoint.Endpoint {
	if state := Get(strings.ToLower(key)); state != nil {
		return state.Endpoint
	}
	return nil
}

// Keys returns the keys of every managed endpoint
func Keys() []string {
	list := List()
	keys := make([]string, 0, len(list))
	for _, state := range list {
		keys = append(keys, state.Stored.Key)
	}
	return keys
}

// configDefinition returns the effective YAML definition of an endpoint of the configuration file, or nil
func configDefinition(key string) []byte {
	if definitions := configDefinitions.Load(); definitions != nil {
		return (*definitions)[key]
	}
	return nil
}

func publish(snapshot map[string]*State) {
	states.Store(&snapshot)
}

// putState replaces the state of a managed endpoint in a new snapshot. statesMutex must be held.
func putState(state *State) {
	replaceState(state.Stored.Key, state)
}

// replaceState removes the state stored under oldKey and adds state under its own key, in a single new snapshot, so
// that a renamed managed endpoint is never listed under both keys. statesMutex must be held.
func replaceState(oldKey string, state *State) {
	next := make(map[string]*State)
	if current := states.Load(); current != nil {
		for key, value := range *current {
			if key != oldKey {
				next[key] = value
			}
		}
	}
	next[state.Stored.Key] = state
	publish(next)
}

// removeState removes the state of a managed endpoint in a new snapshot. statesMutex must be held.
func removeState(key string) {
	next := make(map[string]*State)
	if current := states.Load(); current != nil {
		for existingKey, value := range *current {
			if existingKey != key {
				next[existingKey] = value
			}
		}
	}
	publish(next)
}

func keysExcept(keys []string, excluded string) []string {
	others := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != excluded {
			others = append(others, k)
		}
	}
	return others
}
