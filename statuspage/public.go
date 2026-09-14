package statuspage

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
	"github.com/TwiN/gocache/v2"
	"github.com/TwiN/logr"
	"golang.org/x/sync/singleflight"
)

const (
	publicCacheTTL      = 30 * time.Second
	unavailableCacheTTL = 5 * time.Second

	maximumConcurrentAssemblies = 4
)

var (
	// ErrPageNotFound is returned when the status page does not exist or is not published
	ErrPageNotFound = errors.New("status page not found")

	// ErrPageUnavailable is returned when the status page could not be assembled
	ErrPageUnavailable = errors.New("status page temporarily unavailable")

	errStorageNotSupported = errors.New("the storage does not support public status pages")
)

// summaryReader is the part of the storage used to assemble the public status pages
type summaryReader interface {
	GetEndpointSummaries(keys []string, maximumResults int, now time.Time) (map[string]*common.EndpointSummary, error)
}

var (
	// getSummaryReader resolves the reader once per assembly: the storage is closed and replaced on reload, so no reader is
	// kept between assemblies. It is replaced in tests.
	getSummaryReader = func() (summaryReader, bool) {
		return store.GetEndpointSummaryBatchReader()
	}

	// publicCache holds the JSON payloads, and the unavailability of the pages that failed to be assembled, by
	// slug|revision|generation
	publicCache = gocache.NewCache().WithMaxSize(1000).WithEvictionPolicy(gocache.LeastRecentlyUsed)

	// assemblies deduplicates the concurrent assemblies of the same page revision
	assemblies singleflight.Group

	// assemblySemaphore limits the concurrent assemblies of the public status pages
	assemblySemaphore = make(chan struct{}, maximumConcurrentAssemblies)

	// semaphoreTimeout is how long an assembly waits for a slot of its semaphore
	semaphoreTimeout = 5 * time.Second
)

// unavailableMarker is cached when a page could not be assembled, so that a failing storage is not queried by every request
type unavailableMarker struct{}

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
	if body, cached, err := cachedPublicPage(cacheKey); cached {
		return body, err
	}
	value, err, _ := assemblies.Do(cacheKey, func() (any, error) {
		if body, cached, err := cachedPublicPage(cacheKey); cached {
			return body, err
		}
		release, acquired := acquireSlot(assemblySemaphore)
		if !acquired {
			logr.Warnf("[statuspage.PublicPage] Timed out waiting to assemble status page with slug=%s", slug)
			return nil, ErrPageUnavailable
		}
		defer release()
		body, err := assemble(published.Page, published.MaximumResults, time.Now())
		if err != nil {
			logr.Errorf("[statuspage.PublicPage] Failed to assemble status page with slug=%s: %s", slug, err.Error())
			publicCache.SetWithTTL(cacheKey, unavailableMarker{}, unavailableCacheTTL)
			return nil, ErrPageUnavailable
		}
		publicCache.SetWithTTL(cacheKey, body, publicCacheTTL)
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
