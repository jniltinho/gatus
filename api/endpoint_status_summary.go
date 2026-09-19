package api

import (
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/managedendpoint"
	"gatus/v5/statuspage"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"

	"github.com/TwiN/logr"
)

// endpointStatusResponse is the status of an endpoint with the fields of the fork used by the details page of the
// dashboard: whether the endpoint is a push endpoint, its uptimes and average response times, and the response time of
// its last result, whatever the page of results requested
type endpointStatusResponse struct {
	*endpoint.Status
	Push                bool                           `json:"push"`
	Uptime              statuspage.UptimePayload       `json:"uptime"`
	ResponseTime        statuspage.ResponseTimePayload `json:"responseTime"`
	CurrentResponseTime *int64                         `json:"currentResponseTime"`
}

type endpointSummaryReader interface {
	GetEndpointSummaries(keys []string, maximumResults int, now time.Time) (map[string]*common.EndpointSummary, error)
}

func newEndpointStatusResponse(cfg *config.Config, key string, status *endpoint.Status) *endpointStatusResponse {
	response := &endpointStatusResponse{Status: status, Push: isPushEndpoint(cfg, key)}
	reader, ok := store.Get().(endpointSummaryReader)
	if !ok {
		return response
	}
	summaries, err := reader.GetEndpointSummaries([]string{key}, 1, time.Now())
	if err != nil {
		logr.Errorf("[api.EndpointStatus] Failed to retrieve the summary of endpoint with key=%s: %s", key, err.Error())
		return response
	}
	summary := summaries[key]
	if summary == nil {
		return response
	}
	response.Uptime, response.ResponseTime = statuspage.UptimePayloads(summary.Uptimes)
	if numberOfResults := len(summary.Results); numberOfResults > 0 {
		if milliseconds := summary.Results[numberOfResults-1].Duration.Milliseconds(); milliseconds > 0 {
			response.CurrentResponseTime = &milliseconds
		}
	}
	return response
}

// isPushEndpoint returns whether the key is the key of a push endpoint managed through the administration, in any state,
// or of an external endpoint of the configuration file, enabled or not
func isPushEndpoint(cfg *config.Config, key string) bool {
	if state := managedendpoint.Get(key); state != nil && state.Push != nil {
		return true
	}
	for _, externalEndpoint := range cfg.ExternalEndpoints {
		if externalEndpoint.Key() == key {
			return true
		}
	}
	return false
}
