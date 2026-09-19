package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"time"

	"gatus/v5/config"
	"gatus/v5/internal/httpx"
	"gatus/v5/liveupdates"
	"gatus/v5/managedendpoint"
	"gatus/v5/statuspage"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
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
func endpointEventsHandler(cfg *config.Config) echo.HandlerFunc {
	trustedProxies := cfg.StatusPages.TrustedProxyPrefixes()
	return func(c *echo.Context) error {
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil || (cfg.GetEndpointByKey(key) == nil && cfg.GetExternalEndpointByKey(key) == nil && managedendpoint.Get(key) == nil) {
			return httpx.JSON(c, http.StatusNotFound, map[string]any{"error": "endpoint not found"})
		}
		return streamEndpointEvents(c, key, eventStreamClientIP(c, trustedProxies), func(status int, body string) error {
			httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
			httpx.SetHeader(c, echo.HeaderContentType, echo.MIMEApplicationJSON)
			return httpx.SendString(c, status, body)
		})
	}
}

// statusPageEndpointEventsHandler streams the notifications of the new results of an endpoint of a published status
// page. Like the details of the endpoint, a page that is not published or does not show the endpoint gets the identical
// 404 of the status pages, before any limit and without reading the storage.
func statusPageEndpointEventsHandler(notFound echo.HandlerFunc, trustedProxies []netip.Prefix) echo.HandlerFunc {
	return func(c *echo.Context) error {
		published, captured := publishedStatusPage(c)
		if !captured {
			return notFound(c)
		}
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil || !statuspage.IsEndpointShownOf(published, key) {
			return notFound(c)
		}
		setPublicAPIHeaders(c)
		if published.Page.RequiresLogin() {
			httpx.Vary(c, echo.HeaderAuthorization)
			c.Set(localsProtectedEventStream, true)
		}
		return streamEndpointEvents(c, key, eventStreamClientIP(c, trustedProxies), func(status int, body string) error {
			return sendStatusPageError(c, status, body)
		})
	}
}

func eventStreamClientIP(c *echo.Context, trustedProxies []netip.Prefix) netip.Addr {
	remoteIP := httpx.RemoteIP(c)
	forwardedFor := httpx.HeaderValues(c, echo.HeaderXForwardedFor)
	statuspage.ObserveConnection(remoteIP, len(forwardedFor) > 0, trustedProxies)
	return statuspage.ClientIP(remoteIP, forwardedFor, trustedProxies)
}

// streamEndpointEvents answers HEAD with the headers only, refuses the stream while the live updates are closed (503) or
// when a limit is reached (429), and otherwise streams the notifications of the endpoint until the client leaves, the
// live updates are closed or the maximum duration of a stream is reached
func streamEndpointEvents(c *echo.Context, key string, clientIP netip.Addr, sendError func(status int, body string) error) error {
	if c.Request().Method == http.MethodHead {
		setEventStreamHeaders(c)
		return httpx.SendStatus(c, http.StatusOK)
	}
	notifications, currentSequence, cancel, err := liveupdates.Subscribe(key)
	if errors.Is(err, liveupdates.ErrClosed) {
		return sendError(http.StatusServiceUnavailable, eventStreamUnavailableBody)
	}
	if !eventStreams.acquire(clientIP) {
		cancel()
		httpx.SetHeader(c, echo.HeaderRetryAfter, eventStreamRetryAfterSeconds)
		return sendError(http.StatusTooManyRequests, statusPageTooManyRequestsBody)
	}
	lastEventID := parseLastEventID(httpx.Header(c, "Last-Event-ID"), c.QueryParam("lastEventId"))
	// The handler is the loop of the stream: there is no writer goroutine anymore, so the slot and the subscription are
	// released on every way out of it, a panic included
	defer func() {
		if recovered := recover(); recovered != nil {
			logr.Errorf("[api.streamEndpointEvents] Recovered from a panic in the event stream of key=%s: %v", key, recovered)
		}
		cancel()
		eventStreams.release(clientIP)
	}()
	stream := newEventStream(c)
	// Before the first byte: the write timeout of the server is for ordinary answers, and would cut the stream after a few
	// seconds. The cut would look like a client that left, because it also ends the context of the request.
	stream.extendWriteDeadline()
	setEventStreamHeaders(c)
	c.Response().WriteHeader(http.StatusOK)
	if eventStreamTestHook != nil {
		eventStreamTestHook()
	}
	writeEndpointEvents(c.Request().Context(), stream, key, notifications, currentSequence, lastEventID)
	return nil
}

// eventStream writes the events to the client and flushes each one, so that none waits in a buffer
type eventStream struct {
	writer     io.Writer
	controller *http.ResponseController
}

func newEventStream(c *echo.Context) *eventStream {
	return &eventStream{writer: c.Response(), controller: http.NewResponseController(c.Response())}
}

// extendWriteDeadline gives the stream a write deadline longer than its maximum duration. A writer without deadlines,
// like the recorder of a test, is left as it is.
func (stream *eventStream) extendWriteDeadline() {
	if err := stream.controller.SetWriteDeadline(time.Now().Add(liveupdates.StreamWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		logr.Debugf("[api.eventStream] Failed to extend the write deadline of an event stream: %s", err.Error())
	}
}

// send writes and flushes, and returns whether the client is still there
func (stream *eventStream) send(text string) bool {
	if _, err := io.WriteString(stream.writer, text); err != nil {
		return false
	}
	return stream.controller.Flush() == nil
}

func setEventStreamHeaders(c *echo.Context) {
	httpx.SetHeader(c, echo.HeaderContentType, "text/event-stream")
	// Fork: the stream of a page that requires a login is private, so that no shared cache keeps it
	if protected, _ := c.Get(localsProtectedEventStream).(bool); protected {
		httpx.SetHeader(c, echo.HeaderCacheControl, "private, no-cache, no-store, no-transform")
	} else {
		httpx.SetHeader(c, echo.HeaderCacheControl, "no-cache, no-store, no-transform")
	}
	httpx.SetHeader(c, "X-Accel-Buffering", "no")
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
func writeEndpointEvents(ctx context.Context, stream *eventStream, key string, notifications <-chan struct{}, currentSequence, lastEventID uint64) {
	if !stream.send(fmt.Sprintf("retry: 3000\nid: %d\n\n", currentSequence)) {
		return
	}
	if lastEventID < currentSequence && !writeResultEvent(stream, currentSequence) {
		return
	}
	ping := time.NewTicker(liveupdates.PingInterval)
	defer ping.Stop()
	maximumDuration := time.NewTimer(liveupdates.MaximumStreamDuration)
	defer maximumDuration.Stop()
	for {
		select {
		case _, open := <-notifications:
			if !open || !writeResultEvent(stream, liveupdates.Sequence(key)) {
				return
			}
		case <-ping.C:
			if !stream.send(": ping\n\n") {
				return
			}
		case <-maximumDuration.C:
			return
		case <-ctx.Done():
			// The client left, or the server is shutting down
			return
		}
	}
}

// writeResultEvent writes a result event and returns whether the client is still connected
func writeResultEvent(stream *eventStream, sequence uint64) bool {
	return stream.send(fmt.Sprintf("event: result\nid: %d\ndata: {}\n\n", sequence))
}
