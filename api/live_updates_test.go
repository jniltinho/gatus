package api

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"gatus/v5/config/endpoint"
	"gatus/v5/liveupdates"
	"gatus/v5/watchdog"
)

// newEventStreamTestServer starts the router of the status page tests on a real listener, because app.Test reads the
// whole body and cannot test a stream
func newEventStreamTestServer(t *testing.T) string {
	t.Helper()
	liveupdates.Open()
	previousPing, previousMaximum := liveupdates.PingInterval, liveupdates.MaximumStreamDuration
	liveupdates.PingInterval, liveupdates.MaximumStreamDuration = 100*time.Millisecond, 5*time.Second
	enabled, rateLimit := true, 0
	statusPages := statusPagesTestConfig(enabled, rateLimit)
	app := newStatusPageTestApp(t, statusPageBasicSecurity(t), statusPages)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(listener) }()
	t.Cleanup(func() {
		liveupdates.Close()
		_ = app.ShutdownWithTimeout(2 * time.Second)
		liveupdates.Open()
		liveupdates.PingInterval, liveupdates.MaximumStreamDuration = previousPing, previousMaximum
	})
	return "http://" + listener.Addr().String()
}

func eventStreamRequest(t *testing.T, method, url string, headers map[string]string, authenticated bool) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Accept", "text/event-stream")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	if authenticated {
		request.SetBasicAuth("admin", "secret")
	}
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	return response
}

// readEventMessage returns the next message of the stream that is not a ping, or fails after the timeout
func readEventMessage(t *testing.T, reader *bufio.Reader, timeout time.Duration) string {
	t.Helper()
	messages := make(chan string, 1)
	go func() {
		for {
			var lines []string
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					messages <- "EOF"
					return
				}
				line = strings.TrimRight(line, "\n")
				if len(line) == 0 {
					break
				}
				lines = append(lines, line)
			}
			if message := strings.Join(lines, "\n"); message != ": ping" {
				messages <- message
				return
			}
		}
	}()
	select {
	case message := <-messages:
		return message
	case <-time.After(timeout):
		t.Fatal("timed out waiting for a message of the event stream")
		return ""
	}
}

func currentSequenceHeader(key string) map[string]string {
	return map[string]string{"Last-Event-ID": strconv.FormatUint(liveupdates.Sequence(key), 10)}
}

func TestEndpointEvents_Protected(t *testing.T) {
	base := newEventStreamTestServer(t)
	response := eventStreamRequest(t, http.MethodGet, base+"/api/v1/endpoints/core_api/events", map[string]string{"Accept-Encoding": "br, gzip", "Last-Event-ID": strconv.FormatUint(liveupdates.Sequence("core_api"), 10)}, true)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" || response.Header.Get("Cache-Control") != "no-cache, no-store, no-transform" || response.Header.Get("X-Accel-Buffering") != "no" || response.Header.Get("Content-Encoding") != "" {
		t.Fatalf("unexpected response: %d %v", response.StatusCode, response.Header)
	}
	reader := bufio.NewReader(response.Body)
	if first := readEventMessage(t, reader, 2*time.Second); !strings.HasPrefix(first, "retry: 3000\nid: ") {
		t.Fatalf("expected the retry and the current sequence first, got %q", first)
	}
	ep := &endpoint.Endpoint{Name: "api", Group: "core"}
	watchdog.UpdateEndpointStatus(ep, &endpoint.Result{Pending: true, Message: "secret pending message", Timestamp: time.Now()})
	message := readEventMessage(t, reader, 2*time.Second)
	if !strings.HasPrefix(message, "event: result\nid: ") || !strings.HasSuffix(message, "\ndata: {}") || strings.Contains(message, "secret") {
		t.Fatalf("expected a result event without data, got %q", message)
	}
	if id := strings.Split(strings.Split(message, "\n")[1], ": ")[1]; id != strconv.FormatUint(liveupdates.Sequence("core_api"), 10) {
		t.Errorf("expected the id to be the sequence of the endpoint, got %s", id)
	}

	unauthenticated := eventStreamRequest(t, http.MethodGet, base+"/api/v1/endpoints/core_api/events", nil, false)
	_ = unauthenticated.Body.Close()
	if unauthenticated.StatusCode != http.StatusUnauthorized || unauthenticated.Header.Get("WWW-Authenticate") != "" {
		t.Errorf("expected 401 without the Basic challenge, got %d %v", unauthenticated.StatusCode, unauthenticated.Header)
	}
	unknown := eventStreamRequest(t, http.MethodGet, base+"/api/v1/endpoints/core_missing/events", nil, true)
	_ = unknown.Body.Close()
	if unknown.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for an unknown endpoint, got %d", unknown.StatusCode)
	}
}

func TestEndpointEvents_LastEventID(t *testing.T) {
	base := newEventStreamTestServer(t)
	liveupdates.Publish("core_api")
	missed := strconv.FormatUint(liveupdates.Sequence("core_api")-1, 10)
	for name, headers := range map[string]map[string]string{"header": {"Last-Event-ID": missed}, "parameter": nil} {
		url := base + "/api/v1/endpoints/core_api/events"
		if headers == nil {
			url += "?lastEventId=" + missed
		}
		response := eventStreamRequest(t, http.MethodGet, url, headers, true)
		reader := bufio.NewReader(response.Body)
		readEventMessage(t, reader, 2*time.Second)
		if message := readEventMessage(t, reader, 2*time.Second); !strings.HasPrefix(message, "event: result") {
			t.Errorf("%s: expected a result event right away, got %q", name, message)
		}
		_ = response.Body.Close()
	}
}

func TestEndpointEvents_Public(t *testing.T) {
	base := newEventStreamTestServer(t)
	response := eventStreamRequest(t, http.MethodGet, base+"/api/v1/status-pages/infra/endpoints/core_api/events", currentSequenceHeader("core_api"), false)
	reader := bufio.NewReader(response.Body)
	if response.StatusCode != http.StatusOK || response.Header.Get("X-Robots-Tag") != "noindex, nofollow" || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected public event stream: %d %v", response.StatusCode, response.Header)
	}
	readEventMessage(t, reader, 2*time.Second)
	_ = response.Body.Close()
	reference := eventStreamRequest(t, http.MethodGet, base+"/api/v1/status-pages/missing", nil, false)
	referenceBody, _ := io.ReadAll(reference.Body)
	_ = reference.Body.Close()
	for _, path := range []string{"/api/v1/status-pages/infra/endpoints/core_missing/events", "/api/v1/status-pages/hidden/endpoints/core_api/events"} {
		notFound := eventStreamRequest(t, http.MethodGet, base+path, nil, false)
		body, _ := io.ReadAll(notFound.Body)
		_ = notFound.Body.Close()
		if notFound.StatusCode != http.StatusNotFound || string(body) != string(referenceBody) {
			t.Errorf("%s: expected the identical 404, got %d %s", path, notFound.StatusCode, body)
		}
	}
}

func waitForEventStreams(t *testing.T, expected int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		eventStreams.mutex.Lock()
		total := eventStreams.total
		eventStreams.mutex.Unlock()
		if total == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %d open event streams, got %d", expected, total)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestEndpointEvents_Limits(t *testing.T) {
	base := newEventStreamTestServer(t)
	url := base + "/api/v1/status-pages/infra/endpoints/core_api/events"
	for i := 0; i < 20; i++ {
		head := eventStreamRequest(t, http.MethodHead, url, nil, false)
		_ = head.Body.Close()
		if head.StatusCode != http.StatusOK || head.Header.Get("Content-Type") != "text/event-stream" {
			t.Fatalf("expected HEAD to answer the headers, got %d %v", head.StatusCode, head.Header)
		}
	}
	waitForEventStreams(t, 0)
	var streams []*http.Response
	for i := 0; i < maximumEventStreamsPerIP; i++ {
		stream := eventStreamRequest(t, http.MethodGet, url, currentSequenceHeader("core_api"), false)
		if stream.StatusCode != http.StatusOK {
			t.Fatalf("expected the stream %d to open, got %d", i+1, stream.StatusCode)
		}
		readEventMessage(t, bufio.NewReader(stream.Body), 2*time.Second)
		streams = append(streams, stream)
	}
	refused := eventStreamRequest(t, http.MethodGet, url, nil, false)
	body, _ := io.ReadAll(refused.Body)
	_ = refused.Body.Close()
	if refused.StatusCode != http.StatusTooManyRequests || refused.Header.Get("Retry-After") != "30" || refused.Header.Get("Cache-Control") != "no-store" || string(body) != statusPageTooManyRequestsBody {
		t.Errorf("expected 429 above the limit per IP, got %d %v %s", refused.StatusCode, refused.Header, body)
	}
	// A client that leaves frees its slot at the next ping
	_ = streams[0].Body.Close()
	waitForEventStreams(t, maximumEventStreamsPerIP-1)
	reopened := eventStreamRequest(t, http.MethodGet, url, nil, false)
	if reopened.StatusCode != http.StatusOK {
		t.Errorf("expected a stream to open once a slot is free, got %d", reopened.StatusCode)
	}
	_ = reopened.Body.Close()
	for _, stream := range streams[1:] {
		_ = stream.Body.Close()
	}
	waitForEventStreams(t, 0)
}

func TestEndpointEvents_PanicReleasesTheSlot(t *testing.T) {
	base := newEventStreamTestServer(t)
	eventStreamTestHook = func() { panic("test panic") }
	defer func() { eventStreamTestHook = nil }()
	response := eventStreamRequest(t, http.MethodGet, base+"/api/v1/status-pages/infra/endpoints/core_api/events", nil, false)
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	waitForEventStreams(t, 0)
}

func TestEndpointEvents_CloseEndsTheStreams(t *testing.T) {
	base := newEventStreamTestServer(t)
	url := base + "/api/v1/status-pages/infra/endpoints/core_api/events"
	response := eventStreamRequest(t, http.MethodGet, url, currentSequenceHeader("core_api"), false)
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	readEventMessage(t, reader, 2*time.Second)
	liveupdates.Close()
	defer liveupdates.Open()
	if message := readEventMessage(t, reader, 2*time.Second); message != "EOF" {
		t.Errorf("expected the stream to end, got %q", message)
	}
	refused := eventStreamRequest(t, http.MethodGet, url, nil, false)
	_ = refused.Body.Close()
	if refused.StatusCode != http.StatusServiceUnavailable || refused.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("expected 503 while the live updates are closed, got %d %v", refused.StatusCode, refused.Header)
	}
	waitForEventStreams(t, 0)
}

func TestPublicEndpointDetails_RenewedByNewResult(t *testing.T) {
	app := newStatusPageTestApp(t, statusPageBasicSecurity(t), statusPagesTestConfig(true, 0))
	_, before := doStatusPageRequest(t, app, http.MethodGet, "/api/v1/status-pages/infra/endpoints/core_api")
	watchdog.UpdateEndpointStatus(&endpoint.Endpoint{Name: "api", Group: "core"}, &endpoint.Result{Success: false, Timestamp: time.Now()})
	_, after := doStatusPageRequest(t, app, http.MethodGet, "/api/v1/status-pages/infra/endpoints/core_api")
	if strings.Count(after, `"timestamp"`) <= strings.Count(before, `"timestamp"`) || !strings.Contains(after, `"status":"down"`) {
		t.Errorf("expected the cached details to be renewed by the new result, got %s", after)
	}
}
