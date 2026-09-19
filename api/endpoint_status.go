package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"gatus/v5/client"
	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/config/remote"
	"gatus/v5/internal/httpx"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
	"gatus/v5/storage/store/common/paging"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

// EndpointStatuses handles requests to retrieve all EndpointStatus
// Due to how intensive this operation can be on the storage, this function leverages a cache.
func EndpointStatuses(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)
		value, exists := cache.Get(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize))
		var data []byte
		if !exists {
			endpointStatuses, err := store.Get().GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(page, pageSize))
			if err != nil {
				logr.Errorf("[api.EndpointStatuses] Failed to retrieve endpoint statuses: %s", err.Error())
				return httpx.SendString(c, 500, err.Error())
			}
			// ALPHA: Retrieve endpoint statuses from remote instances
			if endpointStatusesFromRemote, err := getEndpointStatusesFromRemoteInstances(cfg.Remote); err != nil {
				logr.Errorf("[handler.EndpointStatuses] Silently failed to retrieve endpoint statuses from remote: %s", err.Error())
			} else if endpointStatusesFromRemote != nil {
				endpointStatuses = append(endpointStatuses, endpointStatusesFromRemote...)
			}
			// Marshal endpoint statuses to JSON
			data, err = json.Marshal(endpointStatuses)
			if err != nil {
				logr.Errorf("[api.EndpointStatuses] Unable to marshal object to JSON: %s", err.Error())
				return httpx.SendString(c, 500, "unable to marshal object to JSON")
			}
			cache.SetWithTTL(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize), data, cacheTTL)
		} else {
			data = value.([]byte)
		}
		httpx.SetHeader(c, "Content-Type", "application/json")
		return httpx.Send(c, 200, data)
	}
}

func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config) ([]*endpoint.Status, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}
	var endpointStatusesFromAllRemotes []*endpoint.Status
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)
	for _, instance := range remoteConfig.Instances {
		response, err := httpClient.Get(instance.URL)
		if err != nil {
			// Log the error but continue with other instances
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		var endpointStatuses []*endpoint.Status
		if err = json.NewDecoder(response.Body).Decode(&endpointStatuses); err != nil {
			_ = response.Body.Close()
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to decode endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		_ = response.Body.Close()
		for _, endpointStatus := range endpointStatuses {
			endpointStatus.Name = instance.EndpointPrefix + endpointStatus.Name
		}
		endpointStatusesFromAllRemotes = append(endpointStatusesFromAllRemotes, endpointStatuses...)
	}
	// Only return nil, error if no remote instances were successfully processed
	if len(endpointStatusesFromAllRemotes) == 0 && remoteConfig.Instances != nil {
		return nil, fmt.Errorf("failed to retrieve endpoint statuses from all remote instances")
	}
	return endpointStatusesFromAllRemotes, nil
}

// EndpointStatus retrieves a single endpoint.Status by group and endpoint name
func EndpointStatus(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil {
			logr.Errorf("[api.EndpointStatus] Failed to decode key: %s", err.Error())
			return httpx.SendString(c, 400, "invalid key encoding")
		}
		endpointStatus, err := store.Get().GetEndpointStatusByKey(key, paging.NewEndpointStatusParams().WithResults(page, pageSize).WithEvents(1, cfg.Storage.MaximumNumberOfEvents))
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return httpx.SendString(c, 404, err.Error())
			}
			logr.Errorf("[api.EndpointStatus] Failed to retrieve endpoint status: %s", err.Error())
			return httpx.SendString(c, 500, err.Error())
		}
		if endpointStatus == nil { // XXX: is this check necessary?
			logr.Errorf("[api.EndpointStatus] Endpoint with key=%s not found", key)
			return httpx.SendString(c, 404, "not found")
		}
		output, err := json.Marshal(newEndpointStatusResponse(cfg, key, endpointStatus))
		if err != nil {
			logr.Errorf("[api.EndpointStatus] Unable to marshal object to JSON: %s", err.Error())
			return httpx.SendString(c, 500, "unable to marshal object to JSON")
		}
		httpx.SetHeader(c, "Content-Type", "application/json")
		return httpx.Send(c, 200, output)
	}
}
