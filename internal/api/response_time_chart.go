package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"gatus/v5/internal/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/internal/managedendpoint"
	"gatus/v5/internal/statuspage"
	"gatus/v5/internal/storage/store"
	"gatus/v5/internal/storage/store/common"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

const (
	// maximumRecentChartResults is the maximum number of results of the recent period of the protected chart (fork), like
	// the heartbeats of the chart of the Uptime Kuma
	maximumRecentChartResults = 100

	responseTimeChartInvalidPeriodBody = `{"error":"invalid period"}`
)

// responseTimeChartPeriod is a period of the response time chart made of buckets
type responseTimeChartPeriod struct {
	bucketSeconds int
	buckets       int
}

// responseTimeChartPeriods are the periods of buckets of the chart, the recent period being the latest results
var responseTimeChartPeriods = map[string]responseTimeChartPeriod{
	"3h":  {bucketSeconds: common.MinuteBucketSeconds, buckets: 180},
	"6h":  {bucketSeconds: common.MinuteBucketSeconds, buckets: 360},
	"24h": {bucketSeconds: common.MinuteBucketSeconds, buckets: 1440},
	"1w":  {bucketSeconds: common.HourBucketSeconds, buckets: 168},
}

var errResponseTimeChartNotSupported = errors.New("the storage does not support the response time chart")

type responseTimeChartPayload struct {
	Period          string    `json:"period"`
	IntervalSeconds *int64    `json:"intervalSeconds"`
	BucketSeconds   int       `json:"bucketSeconds,omitempty"`
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	// The list of the period is always present, even when empty, and the other one is absent
	Results *[]responseTimeChartResult `json:"results,omitempty"`
	Buckets *[]responseTimeChartBucket `json:"buckets,omitempty"`
}

type responseTimeChartResult struct {
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"`
	DurationMs int64     `json:"durationMs"`
}

type responseTimeChartBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Up        int       `json:"up"`
	Down      int       `json:"down"`
	Pending   int       `json:"pending"`
	AvgMs     *int64    `json:"avgMs"`
	MinMs     *int64    `json:"minMs"`
	MaxMs     *int64    `json:"maxMs"`
}

// isResponseTimeChartPeriod returns whether period is recent or a period of buckets
func isResponseTimeChartPeriod(period string) bool {
	_, isBucketPeriod := responseTimeChartPeriods[period]
	return period == "recent" || isBucketPeriod
}

// endpointIntervalSeconds returns the interval of the endpoint with the given key, used by the chart to break its line
// over long gaps: the interval of the endpoint of the configuration file, of the heartbeat of the external endpoint of
// the configuration file or of the managed endpoint, in this order, and nil when it is zero or unknown
func endpointIntervalSeconds(cfg *config.Config, key string) *int64 {
	var interval time.Duration
	if ep := cfg.GetEndpointByKey(key); ep != nil {
		interval = ep.Interval
	} else if externalEndpoint := cfg.GetExternalEndpointByKey(key); externalEndpoint != nil {
		interval = externalEndpoint.Heartbeat.Interval
	} else if state := managedendpoint.Get(key); state != nil {
		if state.Endpoint != nil {
			interval = state.Endpoint.Interval
		} else if state.Push != nil {
			interval = state.Push.Heartbeat.Interval
		}
	}
	seconds := int64(interval / time.Second)
	if seconds <= 0 {
		return nil
	}
	return &seconds
}

// buildResponseTimeChart encodes the payload of the chart of the endpoint for a valid period, with at most
// maximumResults recent results
func buildResponseTimeChart(cfg *config.Config, key, period string, maximumResults int, now time.Time) ([]byte, error) {
	reader, ok := store.GetResponseTimeChartReader()
	if !ok {
		return nil, errResponseTimeChartNotSupported
	}
	payload := responseTimeChartPayload{Period: period, IntervalSeconds: endpointIntervalSeconds(cfg, key), To: now.UTC()}
	if period == "recent" {
		results, err := reader.GetRecentResponseTimeResults(key, maximumResults)
		if err != nil {
			return nil, err
		}
		payload.From = payload.To
		encodedResults := make([]responseTimeChartResult, 0, len(results))
		for _, result := range results {
			status := "down"
			if result.Pending {
				status = "pending"
			} else if result.Success {
				status = "up"
			}
			encodedResults = append(encodedResults, responseTimeChartResult{Timestamp: result.Timestamp.UTC(), Status: status, DurationMs: result.Duration.Milliseconds()})
		}
		if len(encodedResults) > 0 {
			payload.From, payload.To = encodedResults[0].Timestamp, encodedResults[len(encodedResults)-1].Timestamp
		}
		payload.Results = &encodedResults
		return json.Marshal(payload)
	}
	chartPeriod := responseTimeChartPeriods[period]
	bucketDuration := time.Duration(chartPeriod.bucketSeconds) * time.Second
	lastBucket := now.UTC().Truncate(bucketDuration)
	payload.BucketSeconds = chartPeriod.bucketSeconds
	payload.From = lastBucket.Add(-time.Duration(chartPeriod.buckets-1) * bucketDuration)
	buckets, err := reader.GetResponseTimeBuckets(key, chartPeriod.bucketSeconds, payload.From, lastBucket)
	if err != nil {
		return nil, err
	}
	encodedBuckets := make([]responseTimeChartBucket, 0, len(buckets))
	for _, bucket := range buckets {
		encoded := responseTimeChartBucket{Timestamp: bucket.Timestamp.UTC(), Up: bucket.Up, Down: bucket.Down, Pending: bucket.Pending, MinMs: bucket.MinMs, MaxMs: bucket.MaxMs}
		if bucket.TimedUp > 0 {
			average := bucket.TotalMs / int64(bucket.TimedUp)
			encoded.AvgMs = &average
		}
		encodedBuckets = append(encodedBuckets, encoded)
	}
	payload.Buckets = &encodedBuckets
	return json.Marshal(payload)
}

// endpointResponseTimeChartHandler serves the response time chart of an endpoint of the dashboard. The endpoint must be
// known in memory, like for the event streams, so that an unknown key does not read the storage.
func endpointResponseTimeChartHandler(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil || (cfg.GetEndpointByKey(key) == nil && cfg.GetExternalEndpointByKey(key) == nil && managedendpoint.Get(key) == nil) {
			return httpx.JSON(c, http.StatusNotFound, map[string]any{"error": "endpoint not found"})
		}
		period := httpx.Query(c, "period")
		if !isResponseTimeChartPeriod(period) {
			httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
			return httpx.SendString(c, http.StatusBadRequest, responseTimeChartInvalidPeriodBody)
		}
		maximumResults := maximumRecentChartResults
		if cfg.Storage != nil && cfg.Storage.MaximumNumberOfResults > 0 && cfg.Storage.MaximumNumberOfResults < maximumResults {
			maximumResults = cfg.Storage.MaximumNumberOfResults
		}
		body, err := buildResponseTimeChart(cfg, key, period, maximumResults, time.Now())
		if err != nil {
			logr.Errorf("[api.endpointResponseTimeChartHandler] Failed to build the response time chart of endpoint with key=%s: %s", key, err.Error())
			return httpx.JSON(c, http.StatusInternalServerError, map[string]any{"error": "failed to load the response time chart"})
		}
		httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
		return httpx.Send(c, http.StatusOK, body)
	}
}

// statusPageResponseTimeChartHandler serves the response time chart of an endpoint of a published status page. The
// identical 404 is answered before the period is validated, and the payload is cached, see
// statuspage.PublicResponseTimeChart.
func statusPageResponseTimeChartHandler(cfg *config.Config, notFound echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		slug := c.Param("slug")
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil || !statuspage.IsEndpointShownOf(published, key) {
			return notFound(c)
		}
		period := httpx.Query(c, "period")
		if !isResponseTimeChartPeriod(period) {
			return sendStatusPageError(c, http.StatusBadRequest, responseTimeChartInvalidPeriodBody)
		}
		now := time.Now()
		body, err := statuspage.PublicResponseTimeChart(slug, key, period, now, func(maximumResults int) ([]byte, error) {
			return buildResponseTimeChart(cfg, key, period, maximumResults, now)
		})
		switch {
		case err == nil:
			setPublicAPIHeaders(c)
			setProtectedPageCacheControl(c, published.Page.RequiresLogin(), "no-cache")
			httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
			return httpx.Send(c, http.StatusOK, body)
		case errors.Is(err, statuspage.ErrPageNotFound):
			return notFound(c)
		default:
			return sendStatusPageError(c, http.StatusServiceUnavailable, statusPageUnavailableBody)
		}
	}
}
