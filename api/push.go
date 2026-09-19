package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/internal/httpx"
	"gatus/v5/push"
	"gatus/v5/statuspage"
	"gatus/v5/watchdog"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
)

const (
	// pushNotFoundMessage is the message of the Uptime Kuma for an unknown or inactive monitor
	pushNotFoundMessage = "Monitor not found or not active."

	// pushTooManyRequestsMessage is the message of the rejections above the limit of the client
	pushTooManyRequestsMessage = "Too many requests"

	// pushStorageErrorMessage is the message of a push that could not be stored
	pushStorageErrorMessage = "Failed to store the push."

	// maximumPushPing is the maximum ping accepted by the Uptime Kuma, in milliseconds
	maximumPushPing = 100000000000

	// pushRejectionsPerMinute is the number of rejected pushes accepted per minute from a client before answering 429
	pushRejectionsPerMinute = 30

	// maximumPushLimiterKeys is the maximum number of clients tracked by the push limiter
	maximumPushLimiterKeys = 50000
)

var (
	// errInvalidPushPing has the message of the Uptime Kuma for a ping out of range
	errInvalidPushPing = errors.New("Invalid ping value. Must be between 0 and 100000000000 ms.") //nolint:staticcheck // message of the Uptime Kuma

	// javaScriptFloatPrefix matches what parseFloat reads at the beginning of a string in JavaScript
	javaScriptFloatPrefix = regexp.MustCompile(`^[+-]?(Infinity|(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?)`)

	// pushLimiter counts the rejected pushes of each client. It is reused across reloads.
	pushLimiter = statuspage.NewLimiter(pushRejectionsPerMinute, maximumPushLimiterKeys)
)

// pushResponse is the body of the responses of /api/push, in the format of the Uptime Kuma
type pushResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"msg,omitempty"`
}

// registerPushRoutes registers the push routes compatible with the Uptime Kuma (fork). They must be registered before
// the static files and the security middleware: the catch-alls make sure that no path under /api/push asks for
// authentication.
func registerPushRoutes(unprotectedAPIRouter httpx.Router, cfg *config.Config) {
	resolver := push.NewResolver(cfg)
	trustedProxies := cfg.StatusPages.TrustedProxyPrefixes()
	handler := pushHandler(cfg, resolver, trustedProxies)
	unprotectedAPIRouter.Any("/push/:token", handler)
	unprotectedAPIRouter.Any("/push/:token/:key", handler)
	notFound := func(c *echo.Context) error {
		return rejectPush(c, trustedProxies, pushNotFoundMessage)
	}
	unprotectedAPIRouter.Any("/push", notFound)
	unprotectedAPIRouter.Any("/push/*", notFound)
}

// pushHandler receives a push with the parameters of the Uptime Kuma: status (up or anything else), msg and ping. The
// status pending is an extension of the fork: it records a pending result, while the Uptime Kuma records a failure.
func pushHandler(cfg *config.Config, resolver *push.Resolver, trustedProxies []netip.Prefix) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// Like the Uptime Kuma, the ping is checked before the monitor
		ping, err := parsePushPing(httpx.Query(c, "ping"))
		if err != nil {
			return rejectPush(c, trustedProxies, err.Error())
		}
		endpointKey, err := url.PathUnescape(c.Param("key"))
		if err != nil {
			return rejectPush(c, trustedProxies, pushNotFoundMessage)
		}
		target, globalKeyName, ok := resolver.Resolve(c.Param("token"), endpointKey)
		if !ok {
			return rejectPush(c, trustedProxies, pushNotFoundMessage)
		}
		status := httpx.Query(c, "status")
		if len(status) == 0 {
			status = "up"
		}
		message := httpx.Query(c, "msg")
		if len(message) == 0 {
			message = "OK"
		}
		result := &endpoint.Result{
			Timestamp: time.Now(),
			Success:   status == "up",
			Pending:   status == "pending",
			Message:   endpoint.TruncateResultMessage(message),
			Origin:    endpoint.ResultOriginPush,
		}
		if ping != nil {
			result.Duration = time.Duration(*ping * float64(time.Millisecond))
		}
		if !result.Success && !result.Pending {
			result.Errors = []string{result.Message}
		}
		// The registry processes the pushes of the monitored endpoints and heartbeats, so that none is accepted once they
		// are stopped; external endpoints of the configuration file without heartbeat are not in the registry
		err = watchdog.SubmitEndpointResult(target.Key, result)
		notMonitored := errors.Is(err, watchdog.ErrEndpointNotMonitored) || errors.Is(err, watchdog.ErrMonitoringStopped)
		if notMonitored && target.External != nil && !target.Managed {
			err = watchdog.ProcessExternalEndpointResult(target.External, result, cfg, true)
			notMonitored = false
		}
		if notMonitored {
			return rejectPush(c, trustedProxies, pushNotFoundMessage)
		}
		if err != nil {
			logr.Errorf("[api.pushHandler] Failed to store the push of endpoint with key=%s: %s", target.Key, err.Error())
			return sendPushResponse(c, http.StatusNotFound, pushResponse{Message: pushStorageErrorMessage})
		}
		if len(globalKeyName) > 0 {
			logr.Infof("[api.pushHandler] Received push for endpoint with key=%s with the global key %s; success=%v; pending=%v", target.Key, globalKeyName, result.Success, result.Pending)
		} else {
			logr.Infof("[api.pushHandler] Received push for endpoint with key=%s; success=%v; pending=%v", target.Key, result.Success, result.Pending)
		}
		return sendPushResponse(c, http.StatusOK, pushResponse{OK: true})
	}
}

// parsePushPing reads the ping like the Uptime Kuma does with parseFloat(ping) || null: the numeric prefix is used,
// and an empty, non-numeric or zero value is ignored. A ping out of range is an error.
func parsePushPing(value string) (*float64, error) {
	match := javaScriptFloatPrefix.FindString(strings.TrimLeft(value, " \t\n\r\v\f"))
	if len(match) == 0 {
		return nil, nil
	}
	var ping float64
	if unsigned := strings.TrimLeft(match, "+-"); unsigned == "Infinity" {
		ping = math.Inf(1)
		if strings.HasPrefix(match, "-") {
			ping = math.Inf(-1)
		}
	} else {
		ping, _ = strconv.ParseFloat(match, 64)
	}
	if ping == 0 {
		return nil, nil
	}
	if ping < 0 || ping > maximumPushPing {
		return nil, errInvalidPushPing
	}
	return &ping, nil
}

// rejectPush answers 404 with the given message, or 429 once the client exceeded the rejections allowed per minute.
// Valid pushes are never limited.
func rejectPush(c *echo.Context, trustedProxies []netip.Prefix, message string) error {
	remoteIP := httpx.RemoteIP(c)
	clientIP := statuspage.ClientIP(remoteIP, httpx.HeaderValues(c, echo.HeaderXForwardedFor), trustedProxies)
	if allowed, retryAfter := pushLimiter.Hit(clientIP, time.Now()); !allowed {
		httpx.SetHeader(c, echo.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		return sendPushResponse(c, http.StatusTooManyRequests, pushResponse{Message: pushTooManyRequestsMessage})
	}
	return sendPushResponse(c, http.StatusNotFound, pushResponse{Message: message})
}

func sendPushResponse(c *echo.Context, status int, response pushResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return err
	}
	httpx.SetHeader(c, echo.HeaderCacheControl, "no-store")
	// Written out: the constant of echo says "charset=UTF-8", and the clients of the Uptime Kuma got it in lowercase
	httpx.SetHeader(c, echo.HeaderContentType, "application/json; charset=utf-8")
	return httpx.Send(c, status, body)
}
