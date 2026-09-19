// Package httpx holds what the handlers need from the HTTP framework, in one place. Gatus moved from Fiber (fasthttp)
// to Echo v5 (net/http), and several methods kept their name while changing their meaning: echo.Context.Get reads the
// store of the request and not a header, Path returns the registered route and not the path of the request, and the
// body can only be read once. Every handler goes through these functions, so that those rules live here and not in
// the head of whoever writes the next handler.
package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/labstack/echo/v5"
)

const (
	// MaximumBodySize is the largest body the server accepts, on every route: the 4 MiB Fiber used to enforce by default,
	// which the limits of the administration (256 KB), of the restore (3.5 MiB) and of the login (4 KB) stay below
	MaximumBodySize = 4 << 20

	// bodyKey is the key of the store holding the body read ahead by BufferBody
	bodyKey = "gatus.httpx.body"

	mimeTextPlain = "text/plain; charset=utf-8"
	mimeJSON      = "application/json"
)

// ErrBodyTooLarge is returned by Body when the body is larger than the limit of the route
var ErrBodyTooLarge = errors.New("request body is too large")

// Header returns a header of the request. Never use echo.Context.Get for this: it reads the store of the request, so
// `c.Get("Origin")` is always nil and a check built on it lets everything through.
func Header(c *echo.Context, name string) string {
	return c.Request().Header.Get(name)
}

// HeaderValues returns every line of a header of the request. X-Forwarded-For may come in several lines, and
// http.Header.Get only returns the first one.
func HeaderValues(c *echo.Context, name string) []string {
	return c.Request().Header.Values(name)
}

// SetHeader sets a header of the response. It must be called before the body is written: net/http sends the headers
// with the first byte, and what is set afterwards never reaches the client.
func SetHeader(c *echo.Context, name, value string) {
	c.Response().Header().Set(name, value)
}

// Vary adds a field to the Vary header of the response, once
func Vary(c *echo.Context, field string) {
	header := c.Response().Header()
	for _, line := range header.Values(echo.HeaderVary) {
		for _, existing := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(existing), field) {
				return
			}
		}
	}
	if current := header.Get(echo.HeaderVary); len(current) > 0 {
		header.Set(echo.HeaderVary, current+", "+field)
		return
	}
	header.Set(echo.HeaderVary, field)
}

// Path returns the path of the request. Never use echo.Context.Path for this: it returns the registered route, such as
// /status/:slug, whatever the request was.
func Path(c *echo.Context) string {
	return c.Request().URL.Path
}

// IsTLS returns whether the connection itself uses TLS. Never use echo.Context.Scheme for this: it trusts
// X-Forwarded-Proto and X-Forwarded-Ssl from anyone.
func IsTLS(c *echo.Context) bool {
	return c.Request().TLS != nil
}

// RemoteIP returns the IP address of the connection, never of a header. Never use echo.Context.RealIP for this: what it
// returns depends on Echo.IPExtractor, which this project never sets, so that the rule does not rest on a default.
func RemoteIP(c *echo.Context) netip.Addr {
	return RemoteIPOf(c.Request())
}

// RemoteIPOf returns the IP address of the connection of a request
func RemoteIPOf(request *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return address.Unmap()
}

// Send writes the status and the body. The Content-Type must already be set, otherwise it is text/plain.
func Send(c *echo.Context, status int, body []byte) error {
	header := c.Response().Header()
	if len(header.Get(echo.HeaderContentType)) == 0 {
		header.Set(echo.HeaderContentType, mimeTextPlain)
	}
	c.Response().WriteHeader(status)
	_, err := c.Response().Write(body)
	return err
}

// SendString writes the status and a text
func SendString(c *echo.Context, status int, body string) error {
	return Send(c, status, []byte(body))
}

// SendStatus writes the status with its standard text as the body, as Fiber did
func SendStatus(c *echo.Context, status int) error {
	return SendString(c, status, http.StatusText(status))
}

// NoContent writes the status without a body
func NoContent(c *echo.Context, status int) error {
	c.Response().WriteHeader(status)
	return nil
}

// JSON writes the status and the value as JSON. It does not use echo.Context.JSON, whose encoder adds a line break at
// the end of every body.
func JSON(c *echo.Context, status int, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return JSONBlob(c, status, body)
}

// JSONBlob writes the status and a body that is already JSON
func JSONBlob(c *echo.Context, status int, body []byte) error {
	c.Response().Header().Set(echo.HeaderContentType, mimeJSON)
	c.Response().WriteHeader(status)
	_, err := c.Response().Write(body)
	return err
}

// BufferBody reads the body of every request ahead, up to MaximumBodySize, and answers 413 above it BEFORE the handler
// runs. echo's BodyLimit only notices the excess of a body without Content-Length while it is being read, so a handler
// that never reads the body — the push — would record its result with a chunked body of any size. The body read ahead
// is what Body returns, as many times as it is asked: a middleware that inspects the body does not leave the handler
// with an empty one.
func BufferBody(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		request := c.Request()
		if request.Body == nil || request.Body == http.NoBody {
			return next(c)
		}
		if request.ContentLength > MaximumBodySize {
			return tooLarge(c)
		}
		body, err := io.ReadAll(io.LimitReader(request.Body, MaximumBodySize+1))
		if err != nil {
			return SendString(c, http.StatusBadRequest, "invalid request body")
		}
		if len(body) > MaximumBodySize {
			return tooLarge(c)
		}
		_ = request.Body.Close()
		request.Body = io.NopCloser(bytes.NewReader(body))
		c.Set(bodyKey, body)
		return next(c)
	}
}

func tooLarge(c *echo.Context) error {
	// The rest of the body is not read: the connection is closed, as Fiber did
	c.Response().Header().Set("Connection", "close")
	return SendString(c, http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge))
}

// Body returns the body of the request, read ahead by BufferBody
func Body(c *echo.Context) []byte {
	body, _ := c.Get(bodyKey).([]byte)
	return body
}

// BodyWithin returns the body of the request, or ErrBodyTooLarge when it is larger than the limit of the route
func BodyWithin(c *echo.Context, limit int) ([]byte, error) {
	body := Body(c)
	if len(body) > limit {
		return nil, ErrBodyTooLarge
	}
	return body, nil
}

// Router is what *echo.Echo and *echo.Group have in common, for the functions that register routes on either
type Router interface {
	Use(middleware ...echo.MiddlewareFunc)
	Add(method, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) echo.RouteInfo
	Any(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) echo.RouteInfo
	Group(prefix string, middleware ...echo.MiddlewareFunc) *echo.Group
}

// GetAndHead registers a handler for GET and for HEAD. Fiber registered HEAD with every GET; Echo does not, and its
// RouterConfig.AutoHandleHEAD must stay off: it would run the handler of an event stream, which would reserve a slot and
// open a stream, for a HEAD. net/http drops the body of the answer to a HEAD by itself.
func GetAndHead(router Router, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) {
	router.Add(http.MethodGet, path, handler, middleware...)
	router.Add(http.MethodHead, path, handler, middleware...)
}
