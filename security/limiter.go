package security

import (
	"container/list"
	"net/netip"
	"sync"
	"time"
)

const (
	// failureLimiterWindow is the window in which the authentication failures of a client are counted
	failureLimiterWindow = time.Minute

	// failureLimiterMaximumFailures is the number of failures after which a client is blocked until its window ends
	failureLimiterMaximumFailures = 10

	// failureLimiterMaximumKeys is the maximum number of clients tracked by the failure limiter
	failureLimiterMaximumKeys = 10000
)

// failureLimiter counts the authentication failures of each client in a window of one minute starting at its first
// failure. A client with maximumFailures failures is blocked until its window ends, even with the right credentials.
// IPv4 clients are identified by address and IPv6 clients by /64 prefix. At most maximumKeys clients are tracked, the
// ones without recent failures being forgotten first, without background goroutine. The count is per process.
type failureLimiter struct {
	mutex           sync.Mutex
	maximumFailures int
	maximumKeys     int
	entries         map[netip.Prefix]*list.Element
	recency         *list.List // front: most recent failure
}

type failureLimiterEntry struct {
	key         netip.Prefix
	windowStart time.Time
	failures    int
}

func newFailureLimiter(maximumFailures, maximumKeys int) *failureLimiter {
	return &failureLimiter{
		maximumFailures: maximumFailures,
		maximumKeys:     maximumKeys,
		entries:         make(map[netip.Prefix]*list.Element),
		recency:         list.New(),
	}
}

// Blocked returns whether the client is blocked and, if so, how long it should wait before retrying. It counts nothing.
func (limiter *failureLimiter) Blocked(clientIP netip.Addr, now time.Time) (bool, time.Duration) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	element, exists := limiter.entries[failureLimiterKey(clientIP)]
	if !exists {
		return false, 0
	}
	entry := element.Value.(*failureLimiterEntry)
	windowEnd := entry.windowStart.Add(failureLimiterWindow)
	if entry.failures < limiter.maximumFailures || !now.Before(windowEnd) {
		return false, 0
	}
	retryAfter := windowEnd.Sub(now)
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	return true, retryAfter
}

// Failure counts an authentication failure of the client
func (limiter *failureLimiter) Failure(clientIP netip.Addr, now time.Time) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	key := failureLimiterKey(clientIP)
	if element, exists := limiter.entries[key]; exists {
		limiter.recency.MoveToFront(element)
		entry := element.Value.(*failureLimiterEntry)
		if !now.Before(entry.windowStart.Add(failureLimiterWindow)) {
			entry.windowStart, entry.failures = now, 0
		}
		entry.failures++
		return
	}
	limiter.entries[key] = limiter.recency.PushFront(&failureLimiterEntry{key: key, windowStart: now, failures: 1})
	for len(limiter.entries) > limiter.maximumKeys {
		oldest := limiter.recency.Back()
		delete(limiter.entries, oldest.Value.(*failureLimiterEntry).key)
		limiter.recency.Remove(oldest)
	}
}

// failureLimiterKey identifies an IPv4 client by address and an IPv6 client by /64 prefix
func failureLimiterKey(clientIP netip.Addr) netip.Prefix {
	clientIP = clientIP.Unmap()
	if !clientIP.IsValid() {
		return netip.Prefix{}
	}
	if clientIP.Is4() {
		return netip.PrefixFrom(clientIP, 32)
	}
	prefix, _ := clientIP.WithZone("").Prefix(64)
	return prefix
}
