package api

import (
	"bufio"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"time"

	"gatus/v5/config"
	"gatus/v5/liveupdates"
	"gatus/v5/managedendpoint"
	"gatus/v5/statuspage"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

const (
	// maximumEventStreams is the maximum number of event streams open at the same time, and maximumEventStreamsPerIP
	// the maximum per IP address of client (fork)
	maximumEventStreams      = 500
	maximumEventStreamsPerIP = 10

	eventStreamRetryAfterSeconds = "30"
	eventStreamUnavailableBody   = `{"error":"live updates temporarily unavailable"}`
)

// eventStreamTestHook is called at the beginning of every event stream when set by the tests
var eventStreamTestHook func()

// eventStreams limits the event streams open in total and per IP address of client
var eventStreams = &eventStreamLimiter{perIP: make(map[netip.Addr]int)}

type eventStreamLimiter struct {
	mutex sync.Mutex
	total int
	perIP map[netip.Addr]int
}

// acquire reserves a slot for a new event stream of the client, and returns false when a limit is reached
func (limiter *eventStreamLimiter) acquire(clientIP netip.Addr) bool {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	if limiter.total >= maximumEventStreams || limiter.perIP[clientIP] >= maximumEventStreamsPerIP {
		return false
	}
	limiter.total++
	limiter.perIP[clientIP]++
	return true
}

// release frees the slot of an event stream of the client
func (limiter *eventStreamLimiter) release(clientIP netip.Addr) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	limiter.total--
	if limiter.perIP[clientIP] <= 1 {
		delete(limiter.perIP, clientIP)
	} else {
		limiter.perIP[clientIP]--
	}
}

// endpointEventsHandler streams the notifications of the new results of an endpoint of the dashboard. The endpoint must
// be known in memory (configuration file, external endpoints or managed endpoints in any state), so that an endpoint
// without results yet can be watched without reading the storage.
func endpointEventsHandler(cfg *config.Config) fiber.Handler {
	trustedProxies := cfg.StatusPages.TrustedProxyPrefixes()
	return func(c *fiber.Ctx) error {
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil || (cfg.GetEndpointByKey(key) == nil && cfg.GetExternalEndpointByKey(key) == nil && managedendpoint.Get(key) == nil) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "endpoint not found"})
		}
		return streamEndpointEvents(c, key, eventStreamClientIP(c, trustedProxies), func(status int, body string) error {
			c.Set(fiber.HeaderCacheControl, "no-store")
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Status(status).SendString(body)
		})
	}
}

// statusPageEndpointEventsHandler streams the notifications of the new results of an endpoint of a published status
// page. Like the details of the endpoint, a page that is not published or does not show the endpoint gets the identical
// 404 of the status pages, before any limit and without reading the storage.
func statusPageEndpointEventsHandler(notFound fiber.Handler, trustedProxies []netip.Prefix) fiber.Handler {
	return func(c *fiber.Ctx) error {
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil || !statuspage.IsEndpointShownOf(published, key) {
			return notFound(c)
		}
		setPublicAPIHeaders(c)
		if published.Page.RequiresLogin() {
			c.Vary(fiber.HeaderAuthorization)
			c.Locals(localsProtectedEventStream, true)
		}
		return streamEndpointEvents(c, key, eventStreamClientIP(c, trustedProxies), func(status int, body string) error {
			return sendStatusPageError(c, status, body)
		})
	}
}

func eventStreamClientIP(c *fiber.Ctx, trustedProxies []netip.Prefix) netip.Addr {
	remoteIP, _ := netip.AddrFromSlice(c.Context().RemoteIP())
	forwardedFor := c.Request().Header.PeekAll(fiber.HeaderXForwardedFor)
	statuspage.ObserveConnection(remoteIP, len(forwardedFor) > 0, trustedProxies)
	return statuspage.ClientIP(remoteIP, forwardedFor, trustedProxies)
}

// streamEndpointEvents answers HEAD with the headers only, refuses the stream while the live updates are closed (503) or
// when a limit is reached (429), and otherwise streams the notifications of the endpoint until the client leaves, the
// live updates are closed or the maximum duration of a stream is reached
func streamEndpointEvents(c *fiber.Ctx, key string, clientIP netip.Addr, sendError func(status int, body string) error) error {
	if c.Method() == fiber.MethodHead {
		setEventStreamHeaders(c)
		return c.SendStatus(fiber.StatusOK)
	}
	notifications, currentSequence, cancel, err := liveupdates.Subscribe(key)
	if errors.Is(err, liveupdates.ErrClosed) {
		return sendError(fiber.StatusServiceUnavailable, eventStreamUnavailableBody)
	}
	if !eventStreams.acquire(clientIP) {
		cancel()
		c.Set(fiber.HeaderRetryAfter, eventStreamRetryAfterSeconds)
		return sendError(fiber.StatusTooManyRequests, statusPageTooManyRequestsBody)
	}
	// The values are copied before SetBodyStreamWriter: the writer runs in its own goroutine and must not use the context
	lastEventID := parseLastEventID(c.Get("Last-Event-ID"), c.Query("lastEventId"))
	setEventStreamHeaders(c)
	c.Status(fiber.StatusOK)
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logr.Errorf("[api.streamEndpointEvents] Recovered from a panic in the event stream of key=%s: %v", key, recovered)
			}
			cancel()
			eventStreams.release(clientIP)
		}()
		if eventStreamTestHook != nil {
			eventStreamTestHook()
		}
		writeEndpointEvents(writer, key, notifications, currentSequence, lastEventID)
	})
	return nil
}

func setEventStreamHeaders(c *fiber.Ctx) {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	// Fork: the stream of a page that requires a login is private, so that no shared cache keeps it
	if protected, _ := c.Locals(localsProtectedEventStream).(bool); protected {
		c.Set(fiber.HeaderCacheControl, "private, no-cache, no-store, no-transform")
	} else {
		c.Set(fiber.HeaderCacheControl, "no-cache, no-store, no-transform")
	}
	c.Set("X-Accel-Buffering", "no")
}

// parseLastEventID returns the sequence of the Last-Event-ID header, or of the lastEventId parameter of a new
// EventSource, and 0 when it is missing or invalid
func parseLastEventID(header, parameter string) uint64 {
	value := header
	if len(value) == 0 {
		value = parameter
	}
	lastEventID, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return lastEventID
}

// writeEndpointEvents writes the event stream: the retry delay and the current sequence, a result event right away when
// the client missed one, then a result event per notification and a comment every PingInterval
func writeEndpointEvents(writer *bufio.Writer, key string, notifications <-chan struct{}, currentSequence, lastEventID uint64) {
	if _, err := fmt.Fprintf(writer, "retry: 3000\nid: %d\n\n", currentSequence); err != nil || writer.Flush() != nil {
		return
	}
	if lastEventID < currentSequence && !writeResultEvent(writer, currentSequence) {
		return
	}
	ping := time.NewTicker(liveupdates.PingInterval)
	defer ping.Stop()
	maximumDuration := time.NewTimer(liveupdates.MaximumStreamDuration)
	defer maximumDuration.Stop()
	for {
		select {
		case _, open := <-notifications:
			if !open || !writeResultEvent(writer, liveupdates.Sequence(key)) {
				return
			}
		case <-ping.C:
			if _, err := writer.WriteString(": ping\n\n"); err != nil || writer.Flush() != nil {
				return
			}
		case <-maximumDuration.C:
			return
		}
	}
}

// writeResultEvent writes a result event and returns whether the client is still connected
func writeResultEvent(writer *bufio.Writer, sequence uint64) bool {
	if _, err := fmt.Fprintf(writer, "event: result\nid: %d\ndata: {}\n\n", sequence); err != nil {
		return false
	}
	return writer.Flush() == nil
}
