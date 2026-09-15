package watchdog

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/metrics"
	"gatus/v5/storage/store"
	"github.com/TwiN/logr"
)

var (
	// resultLocks serializes, per endpoint key, the processing of the results of an endpoint: its checks, its pushes and
	// its heartbeat, so that the counters of its alerts are never updated concurrently (fork)
	resultLocks sync.Map // map[string]*sync.Mutex

	// lastPushes keeps, per external endpoint key, the time of the last accepted push, in Unix nanoseconds. It is kept
	// across reloads, and starts at the first time the key is used, so that a heartbeat only fails after a full interval.
	lastPushes sync.Map // map[string]*atomic.Int64
)

// lockEndpointResults locks the processing of the results of the endpoint with the given key and returns the function
// that unlocks it
func lockEndpointResults(key string) func() {
	value, _ := resultLocks.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

// lastPush returns the time of the last accepted push of the external endpoint with the given key
func lastPush(key string) *atomic.Int64 {
	initial := new(atomic.Int64)
	initial.Store(time.Now().UnixNano())
	value, _ := lastPushes.LoadOrStore(key, initial)
	return value.(*atomic.Int64)
}

// startExternal starts monitoring the heartbeat of an external endpoint
func (r *endpointRegistry) startExternal(ee *endpoint.ExternalEndpoint, source Source) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrMonitoringStopped
	}
	key := ee.Key()
	if _, exists := r.entries[key]; exists {
		return ErrEndpointAlreadyMonitored
	}
	ctx, cancel := context.WithCancel(r.ctx)
	entry := &monitoredEndpoint{external: ee, source: source, ctx: ctx, cancel: cancel, done: make(chan struct{})}
	r.entries[key] = entry
	go func() {
		defer close(entry.done)
		monitorExternalEndpointHeartbeat(ctx, ee, r.cfg)
	}()
	return nil
}

// StartExternalEndpoint starts monitoring the heartbeat of an external endpoint for the given source (fork). It is
// stopped with StopEndpoint.
func StartExternalEndpoint(ee *endpoint.ExternalEndpoint, source Source) error {
	endpoints := registry.Load()
	if endpoints == nil {
		return ErrMonitoringStopped
	}
	return endpoints.startExternal(ee, source)
}

// ProcessExternalEndpointResult stores a result of an external endpoint, publishes its metrics and handles its alerts,
// serialized with the other results of the endpoint (fork). A pushed result also restarts the interval of the heartbeat.
// It returns the error of the storage, in which case nothing else is done.
func ProcessExternalEndpointResult(ee *endpoint.ExternalEndpoint, result *endpoint.Result, cfg *config.Config, pushed bool) error {
	unlock := lockEndpointResults(ee.Key())
	defer unlock()
	return processExternalEndpointResult(ee, result, cfg, pushed)
}

// processExternalEndpointResult is ProcessExternalEndpointResult with the lock of the key already held
func processExternalEndpointResult(ee *endpoint.ExternalEndpoint, result *endpoint.Result, cfg *config.Config, pushed bool) error {
	key := ee.Key()
	convertedEndpoint := ee.ToEndpoint()
	if err := store.Get().InsertEndpointResult(convertedEndpoint, result); err != nil {
		return err
	}
	if pushed {
		lastPush(key).Store(time.Now().UnixNano())
	}
	if cfg.Metrics {
		metrics.PublishMetricsForEndpoint(convertedEndpoint, result, metrics.RegisteredExtraLabels())
	}
	if cfg.Maintenance.IsUnderMaintenance() || isUnderMaintenanceWindow(ee.MaintenanceWindows) {
		logr.Debugf("[watchdog.ProcessExternalEndpointResult] Not handling alerting of key=%s because currently in the maintenance window", key)
		return nil
	}
	HandleAlerting(convertedEndpoint, result, cfg.Alerting)
	ee.NumberOfSuccessesInARow = convertedEndpoint.NumberOfSuccessesInARow
	ee.NumberOfFailuresInARow = convertedEndpoint.NumberOfFailuresInARow
	return nil
}

// SubmitEndpointResult processes a result pushed to an endpoint monitored by the watchdog (fork): it is stored in the
// history of the endpoint, and its metrics and alerts are handled like those of its checks, one result at a time. It
// returns ErrEndpointNotMonitored when the endpoint is not monitored, or stopped while waiting for its lock.
func SubmitEndpointResult(key string, result *endpoint.Result) error {
	endpoints := registry.Load()
	if endpoints == nil {
		return ErrMonitoringStopped
	}
	endpoints.mu.Lock()
	entry, exists := endpoints.entries[key]
	endpoints.mu.Unlock()
	if !exists {
		return ErrEndpointNotMonitored
	}
	unlock := lockEndpointResults(key)
	defer unlock()
	if entry.ctx.Err() != nil {
		return ErrEndpointNotMonitored
	}
	// Fork: an external endpoint whose heartbeat is monitored, e.g. a push endpoint managed through the administration
	if entry.external != nil {
		return processExternalEndpointResult(entry.external, result, endpoints.cfg, true)
	}
	ep, cfg := entry.endpoint, endpoints.cfg
	if err := store.Get().InsertEndpointResult(ep, result); err != nil {
		return err
	}
	if cfg.Metrics {
		metrics.PublishMetricsForEndpoint(ep, result, metrics.RegisteredExtraLabels())
	}
	if cfg.Maintenance.IsUnderMaintenance() || isUnderMaintenanceWindow(ep.MaintenanceWindows) {
		logr.Debugf("[watchdog.SubmitEndpointResult] Not handling alerting of key=%s because currently in the maintenance window", key)
		return nil
	}
	HandleAlerting(ep, result, cfg.Alerting)
	return nil
}

func isUnderMaintenanceWindow[T interface{ IsUnderMaintenance() bool }](maintenanceWindows []T) bool {
	for _, maintenanceWindow := range maintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			return true
		}
	}
	return false
}
