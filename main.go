package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/controller"
	"github.com/TwiN/gatus/v5/lifecycle"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
	"github.com/TwiN/logr"
)

const (
	GatusConfigPathEnvVar = "GATUS_CONFIG_PATH"
	GatusConfigFileEnvVar = "GATUS_CONFIG_FILE" // Deprecated in favor of GatusConfigPathEnvVar
	GatusLogLevelEnvVar   = "GATUS_LOG_LEVEL"
)

func main() {
	if delayInSeconds, _ := strconv.Atoi(os.Getenv("GATUS_DELAY_START_SECONDS")); delayInSeconds > 0 {
		logr.Infof("Delaying start by %d seconds", delayInSeconds)
		time.Sleep(time.Duration(delayInSeconds) * time.Second)
	}
	configureLogging()
	cfg, err := loadConfiguration()
	if err != nil {
		panic(err)
	}
	lifecycle.BeginCycle() // ended by start
	initializeStorage(cfg)
	start(cfg)
	// Wait for termination signal
	signalChannel := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		logr.Info("Received termination signal, attempting to gracefully shut down")
		lifecycle.BeginCycle() // never ended: the application is shutting down
		stop(cfg)
		save()
		done <- true
	}()
	<-done
	logr.Info("Shutting down")
}

// start must be called with a lifecycle cycle in progress, which it ends once monitoring has started
func start(cfg *config.Config) {
	go controller.Handle(cfg)
	metrics.InitializePrometheusMetrics(cfg, nil)
	watchdog.Monitor(cfg)
	lifecycle.EndCycle()
	go listenToConfigurationFileChanges(cfg)
}

func stop(cfg *config.Config) {
	watchdog.Shutdown(cfg)
	controller.Shutdown()
	metrics.UnregisterPrometheusMetrics()
	closeTunnels(cfg)
}

func save() {
	if err := store.Get().Save(); err != nil {
		logr.Errorf("Failed to save storage provider: %s", err.Error())
	}
}

func configureLogging() {
	logLevelAsString := os.Getenv(GatusLogLevelEnvVar)
	if logLevel, err := logr.LevelFromString(logLevelAsString); err != nil {
		logr.SetThreshold(logr.LevelInfo)
		if len(logLevelAsString) == 0 {
			logr.Infof("[main.configureLogging] Defaulting log level to %s", logr.LevelInfo)
		} else {
			logr.Warnf("[main.configureLogging] Invalid log level '%s', defaulting to %s", logLevelAsString, logr.LevelInfo)
		}
	} else {
		logr.SetThreshold(logLevel)
		logr.Infof("[main.configureLogging] Log Level is set to %s", logr.GetThreshold())
	}
}

func loadConfiguration() (*config.Config, error) {
	configPath := os.Getenv(GatusConfigPathEnvVar)
	// Backwards compatibility
	if len(configPath) == 0 {
		if configPath = os.Getenv(GatusConfigFileEnvVar); len(configPath) > 0 {
			logr.Warnf("WARNING: %s is deprecated. Please use %s instead.", GatusConfigFileEnvVar, GatusConfigPathEnvVar)
		}
	}
	return config.LoadConfiguration(configPath)
}

// initializeStorage initializes the storage provider
//
// Q: "TwiN, why are you putting this here? Wouldn't it make more sense to have this in the config?!"
// A: Yes. Yes it would make more sense to have it in the config package. But I don't want to import
// the massive SQL dependencies just because I want to import the config, so here we are.
func initializeStorage(cfg *config.Config) {
	err := store.Initialize(cfg.Storage)
	if err != nil {
		panic(err)
	}
	// Remove all SuiteStatuses that represent suites which no longer exist in the configuration
	var suiteKeys []string
	for _, suite := range cfg.Suites {
		suiteKeys = append(suiteKeys, suite.Key())
	}
	numberOfSuiteStatusesDeleted := store.Get().DeleteAllSuiteStatusesNotInKeys(suiteKeys)
	if numberOfSuiteStatusesDeleted > 0 {
		logr.Infof("[main.initializeStorage] Deleted %d suite statuses because their matching suites no longer existed", numberOfSuiteStatusesDeleted)
	}
	// Remove all EndpointStatus that represent endpoints which no longer exist in the configuration
	var keys []string
	for _, ep := range cfg.Endpoints {
		keys = append(keys, ep.Key())
	}
	for _, ee := range cfg.ExternalEndpoints {
		keys = append(keys, ee.Key())
	}
	// Also add endpoints that are part of suites
	for _, suite := range cfg.Suites {
		for _, ep := range suite.Endpoints {
			keys = append(keys, ep.Key())
		}
	}
	logr.Infof("[main.initializeStorage] Total endpoint keys to preserve: %d", len(keys))
	numberOfEndpointStatusesDeleted := store.Get().DeleteAllEndpointStatusesNotInKeys(keys)
	if numberOfEndpointStatusesDeleted > 0 {
		logr.Infof("[main.initializeStorage] Deleted %d endpoint statuses because their matching endpoints no longer existed", numberOfEndpointStatusesDeleted)
	}
	// Clean up the triggered alerts from the storage provider and load valid triggered endpoint alerts
	numberOfPersistedTriggeredAlertsLoaded := 0
	for _, ep := range cfg.Endpoints {
		numberOfPersistedTriggeredAlertsLoaded += watchdog.RestorePersistedTriggeredAlerts(ep)
	}
	for _, ee := range cfg.ExternalEndpoints {
		convertedEndpoint := ee.ToEndpoint()
		if restored := watchdog.RestorePersistedTriggeredAlerts(convertedEndpoint); restored > 0 {
			ee.NumberOfSuccessesInARow, ee.NumberOfFailuresInARow = convertedEndpoint.NumberOfSuccessesInARow, convertedEndpoint.NumberOfFailuresInARow
			numberOfPersistedTriggeredAlertsLoaded += restored
		}
	}
	// Load persisted triggered alerts for suite endpoints
	for _, suite := range cfg.Suites {
		for _, ep := range suite.Endpoints {
			numberOfPersistedTriggeredAlertsLoaded += watchdog.RestorePersistedTriggeredAlerts(ep)
		}
	}
	if numberOfPersistedTriggeredAlertsLoaded > 0 {
		logr.Infof("[main.initializeStorage] Loaded %d persisted triggered alerts", numberOfPersistedTriggeredAlertsLoaded)
	}
}

func closeTunnels(cfg *config.Config) {
	if cfg.Tunneling != nil {
		if err := cfg.Tunneling.Close(); err != nil {
			logr.Errorf("[main.closeTunnels] Error closing SSH tunnels: %v", err)
		}
	}
}

func listenToConfigurationFileChanges(cfg *config.Config) {
	for {
		time.Sleep(30 * time.Second)
		if cfg.HasLoadedConfigurationBeenModified() {
			logr.Info("[main.listenToConfigurationFileChanges] Configuration file has been modified")
			// The new configuration is validated before anything is stopped (see main_reload.go)
			updatedConfig, ok := loadUpdatedConfiguration(cfg, loadConfiguration)
			if !ok {
				continue
			}
			lifecycle.BeginCycle() // ended by start
			stop(cfg)
			time.Sleep(time.Second) // Wait a bit to make sure everything is done.
			save()
			store.Get().Close()
			initializeStorage(updatedConfig)
			start(updatedConfig)
			return
		}
	}
}
