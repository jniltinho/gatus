package statuspage

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/logr"
	"golang.org/x/sync/singleflight"
)

const (
	publicCacheTTL        = 30 * time.Second
	responseTimesCacheTTL = 5 * time.Minute
	unavailableCacheTTL   = 5 * time.Second

	maximumConcurrentAssemblies = 4
)

var (
	// ErrPageNotFound is returned when the status page does not exist or is not published
	ErrPageNotFound = errors.New("status page not found")

	// ErrPageUnavailable is returned when the status page could not be assembled
	ErrPageUnavailable = errors.New("status page temporarily unavailable")

	errStorageNotSupported = errors.New("the storage does not support public status pages")

	// responseTimeDurations are the periods of the response time charts
	responseTimeDurations = map[string]time.Duration{
		"24h": 24 * time.Hour,
		"7d":  7 * 24 * time.Hour,
		"30d": 30 * 24 * time.Hour,
	}
)

// summaryReader is the part of the storage used to assemble the public status pages
type summaryReader interface {
	GetEndpointSummaries(keys []string, maximumResults int, now time.Time) (map[string]*common.EndpointSummary, error)
}

// responseTimeReader is the part of the storage used to assemble the response time charts
type responseTimeReader interface {
	GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error)
}

var (
	// getSummaryReader resolves the reader once per assembly: the storage is closed and replaced on reload, so no reader is
	// kept between assemblies. It is replaced in tests.
	getSummaryReader = func() (summaryReader, bool) {
		return store.GetEndpointSummaryBatchReader()
	}

	// getResponseTimeReader resolves the reader of the response time charts once per assembly. It is replaced in tests.
	getResponseTimeReader = func() (responseTimeReader, bool) {
		return store.Get(), true
	}

	// publicCache holds the JSON payloads, and the unavailability of the payloads that failed to be assembled, by
	// slug|revision|generation (and the duration for the response times)
	publicCache = gocache.NewCache().WithMaxSize(1000).WithEvictionPolicy(gocache.LeastRecentlyUsed)

	// assemblies deduplicates the concurrent assemblies of the same payload
	assemblies singleflight.Group

	// assemblySemaphore limits the concurrent assemblies of the public status pages and of their response time charts
	assemblySemaphore = make(chan struct{}, maximumConcurrentAssemblies)

	// semaphoreTimeout is how long an assembly waits for a slot of its semaphore
	semaphoreTimeout = 5 * time.Second
)

// unavailableMarker is cached when a payload could not be assembled, so that a failing storage is not queried by every
// request
type unavailableMarker struct{}

// ResponseTimesPayload is the public representation of the response time charts of a status page
type ResponseTimesPayload struct {
	Duration  string               `json:"duration"`
	Endpoints []ResponseTimeSeries `json:"endpoints"`
}

// ResponseTimeSeries is the response time chart of an endpoint, identified by its name and group (never by its key)
type ResponseTimeSeries struct {
	Name   string              `json:"name"`
	Group  string              `json:"group"`
	Points []ResponseTimePoint `json:"points"`
}

// ResponseTimePoint is the average response time of an hour, or of a day for the entries merged by the SQL storage
type ResponseTimePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	Milliseconds int       `json:"ms"`
}

// PublicPage returns the JSON payload of the published status page with the given slug. The payload is assembled at most
// once per page revision every 30 seconds, whatever the number of concurrent requests.
//
// It returns ErrPageNotFound, without reading the storage, when the page is not published, and ErrPageUnavailable when
// the storage could not be read or the assembly waited too long for a slot.
func PublicPage(slug string) ([]byte, error) {
	published, ok := Lookup(slug)
	if !ok {
		return nil, ErrPageNotFound
	}
	cacheKey := fmt.Sprintf("%s|%d|%d", slug, published.Revision, published.Generation)
	return cachedAssembly(cacheKey, publicCacheTTL, slug, func() ([]byte, error) {
		return assemble(published.Page, published.MaximumResults, time.Now())
	})
}

// PublicResponseTimes returns the JSON payload of the response time charts of the published status page with the given
// slug over the given duration (24h, 7d or 30d). It is assembled at most once per page revision and duration every 5
// minutes. It returns ErrPageNotFound, without reading the storage, when the page is not published or the duration is
// invalid.
func PublicResponseTimes(slug, duration string) ([]byte, error) {
	period, valid := responseTimeDurations[duration]
	if !valid {
		return nil, ErrPageNotFound
	}
	published, ok := Lookup(slug)
	if !ok {
		return nil, ErrPageNotFound
	}
	cacheKey := fmt.Sprintf("%s|%d|%d|response-times|%s", slug, published.Revision, published.Generation, duration)
	return cachedAssembly(cacheKey, responseTimesCacheTTL, slug, func() ([]byte, error) {
		return assembleResponseTimes(published.Page, duration, period, time.Now())
	})
}

// cachedAssembly returns the payload cached under cacheKey or assembles it once for all concurrent requests, in a slot of
// assemblySemaphore. A failed assembly is cached for unavailableCacheTTL; a timeout waiting for a slot is not cached.
func cachedAssembly(cacheKey string, ttl time.Duration, slug string, assembleFunc func() ([]byte, error)) ([]byte, error) {
	if body, cached, err := cachedPublicPage(cacheKey); cached {
		return body, err
	}
	value, err, _ := assemblies.Do(cacheKey, func() (any, error) {
		if body, cached, err := cachedPublicPage(cacheKey); cached {
			return body, err
		}
		release, acquired := acquireSlot(assemblySemaphore)
		if !acquired {
			logr.Warnf("[statuspage.cachedAssembly] Timed out waiting to assemble status page with slug=%s", slug)
			return nil, ErrPageUnavailable
		}
		defer release()
		body, err := assembleFunc()
		if err != nil {
			logr.Errorf("[statuspage.cachedAssembly] Failed to assemble status page with slug=%s: %s", slug, err.Error())
			publicCache.SetWithTTL(cacheKey, unavailableMarker{}, unavailableCacheTTL)
			return nil, ErrPageUnavailable
		}
		publicCache.SetWithTTL(cacheKey, body, ttl)
		return body, nil
	})
	if err != nil {
		return nil, err
	}
	return value.([]byte), nil
}

func cachedPublicPage(cacheKey string) ([]byte, bool, error) {
	value, exists := publicCache.Get(cacheKey)
	if !exists {
		return nil, false, nil
	}
	switch cached := value.(type) {
	case []byte:
		return cached, true, nil
	case unavailableMarker:
		return nil, true, ErrPageUnavailable
	}
	return nil, false, nil
}

// assemble reads the summaries of the endpoints selected by the page and encodes its public payload
func assemble(page *pageconfig.Page, maximumResults int, now time.Time) ([]byte, error) {
	reader, ok := getSummaryReader()
	if !ok {
		return nil, errStorageNotSupported
	}
	selection := Select(page, Endpoints())
	if selection.Truncated {
		logr.Warnf("[statuspage.assemble] Status page with slug=%s selects more than %d endpoints, only the first %d are shown", page.Slug, pageconfig.MaximumEndpoints, pageconfig.MaximumEndpoints)
	}
	summaries, err := reader.GetEndpointSummaries(selection.Keys(), maximumResults, now)
	if err != nil {
		return nil, err
	}
	return json.Marshal(BuildPayload(page, selection, summaries, now))
}

// assembleResponseTimes reads the hourly average response times of the endpoints of the page with a chart, in display
// order, and encodes them. A page without chart does not read the storage.
func assembleResponseTimes(page *pageconfig.Page, duration string, period time.Duration, now time.Time) ([]byte, error) {
	payload := ResponseTimesPayload{Duration: duration, Endpoints: []ResponseTimeSeries{}}
	charts := make(map[string]struct{}, len(page.Charts))
	for _, key := range page.Charts {
		charts[key] = struct{}{}
	}
	if len(charts) == 0 {
		return json.Marshal(payload)
	}
	reader, ok := getResponseTimeReader()
	if !ok {
		return nil, errStorageNotSupported
	}
	for _, ref := range Select(page, Endpoints()).Refs() {
		if _, chart := charts[ref.Key]; !chart {
			continue
		}
		series := ResponseTimeSeries{Name: ref.Name, Group: ref.Group, Points: []ResponseTimePoint{}}
		hourlyAverages, err := reader.GetHourlyAverageResponseTimeByKey(ref.Key, now.Add(-period), now)
		if err != nil && !errors.Is(err, common.ErrEndpointNotFound) {
			return nil, err
		}
		for hourUnixTimestamp, milliseconds := range hourlyAverages {
			series.Points = append(series.Points, ResponseTimePoint{Timestamp: time.Unix(hourUnixTimestamp, 0).UTC(), Milliseconds: milliseconds})
		}
		sort.Slice(series.Points, func(i, j int) bool {
			return series.Points[i].Timestamp.Before(series.Points[j].Timestamp)
		})
		payload.Endpoints = append(payload.Endpoints, series)
	}
	return json.Marshal(payload)
}

// acquireSlot waits at most semaphoreTimeout for a slot of the semaphore, and returns the function releasing it
func acquireSlot(semaphore chan struct{}) (func(), bool) {
	timer := time.NewTimer(semaphoreTimeout)
	defer timer.Stop()
	select {
	case semaphore <- struct{}{}:
		return func() { <-semaphore }, true
	case <-timer.C:
		return nil, false
	}
}
