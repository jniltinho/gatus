package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

const (
	// LocalsClientIP is the key of the locals of a request holding the IP address of its client (netip.Addr), computed
	// by the api package with status-pages.trusted-proxies. Without it, the IP address of the connection is used.
	LocalsClientIP = "gatus.client-ip"

	// localsBasicAuthentication is the key of the locals of a request holding its basicAuthentication
	localsBasicAuthentication = "gatus.basic-authentication"

	// loginSessionTokenSize is the size of the random token of a login session, in bytes
	loginSessionTokenSize = 32
)

var (
	// ErrInvalidCredentials is returned by Config.Login when the username or the password is wrong
	ErrInvalidCredentials = errors.New("invalid username or password")

	// ErrLoginSessionsUnsupported is returned by Config.Login when the storage does not support login sessions
	ErrLoginSessionsUnsupported = errors.New("the storage does not support login sessions")

	// compareHashAndPassword checks a password against its bcrypt hash. Tests replace it to count the checks.
	compareHashAndPassword = bcrypt.CompareHashAndPassword
)

// TooManyFailuresError is returned by Config.Login when the client is blocked by the failure limiter
type TooManyFailuresError struct {
	// RetryAfter is how long the client should wait before retrying
	RetryAfter time.Duration
}

func (e *TooManyFailuresError) Error() string {
	return "too many failed authentication attempts"
}

// basicAuthentication is the result of the authentication of a request with security.basic, computed once per request
type basicAuthentication struct {
	username      string
	authenticated bool

	// retryAfter is positive when the request has Authorization: Basic but its client is blocked by the failure limiter
	retryAfter time.Duration
}

// UsesBasicLogin returns whether the login screen and the login sessions are used: security.basic without OIDC (fork)
func (c *Config) UsesBasicLogin() bool {
	return c != nil && c.Basic != nil && c.OIDC == nil
}

// LoginMethod returns the login method exposed to the frontend: "oidc", "basic" or empty without authentication
func (c *Config) LoginMethod() string {
	switch {
	case c == nil:
		return ""
	case c.OIDC != nil:
		return "oidc"
	case c.Basic != nil:
		return "basic"
	default:
		return ""
	}
}

// Login checks the credentials sent by the login screen and, when they are right, creates a new login session and sets
// its cookie. The session cookie of the request, if any, is ignored so that a session cannot be fixed. It returns
// ErrInvalidCredentials, a *TooManyFailuresError or an error of the storage.
func (c *Config) Login(ctx *fiber.Ctx, username, password string) error {
	now := time.Now()
	clientIP := requestClientIP(ctx)
	limiter := c.failureLimiter()
	if blocked, retryAfter := limiter.Blocked(clientIP, now); blocked {
		return &TooManyFailuresError{RetryAfter: retryAfter}
	}
	if !c.Basic.checkCredentials(username, password) {
		limiter.Failure(clientIP, now)
		return ErrInvalidCredentials
	}
	loginSessionStore, ok := store.GetLoginSessionStore()
	if !ok {
		return ErrLoginSessionsUnsupported
	}
	if deleted, err := loginSessionStore.DeleteExpiredLoginSessions(now); err != nil {
		logr.Warnf("[security.Login] Failed to delete the expired login sessions: %s", err.Error())
	} else if deleted > 0 {
		logr.Debugf("[security.Login] Deleted %d expired login sessions", deleted)
	}
	tokenBytes := make([]byte, loginSessionTokenSize)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	ttl := c.Basic.sessionTTL()
	err := loginSessionStore.CreateLoginSession(&common.LoginSession{
		TokenHash:             hashLoginSessionToken(token),
		Username:              c.Basic.Username,
		CredentialFingerprint: c.Basic.credentialFingerprint(),
		CreatedAt:             now,
		ExpiresAt:             now.Add(ttl),
	})
	if err != nil {
		return err
	}
	setLoginSessionCookie(ctx, token, int(ttl.Seconds()), time.Time{})
	return nil
}

// Logout deletes the login session of the request, if any, and expires its cookie
func (c *Config) Logout(ctx *fiber.Ctx) error {
	var err error
	if token := ctx.Cookies(cookieNameSession); isLoginSessionToken(token) {
		if loginSessionStore, ok := store.GetLoginSessionStore(); ok {
			err = loginSessionStore.DeleteLoginSession(hashLoginSessionToken(token))
		}
	}
	setLoginSessionCookie(ctx, "", 0, time.Unix(0, 0))
	return err
}

// basicMiddleware responds with 401 to the requests that have neither a valid login session nor the right
// Authorization: Basic, and with 429 to the requests with Authorization: Basic from a client blocked by the limiter
func (c *Config) basicMiddleware(ctx *fiber.Ctx) error {
	authentication := c.authenticateBasic(ctx)
	if authentication.authenticated {
		return ctx.Next()
	}
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	if authentication.retryAfter > 0 {
		ctx.Set(fiber.HeaderRetryAfter, strconv.Itoa(int(math.Ceil(authentication.retryAfter.Seconds()))))
		return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many failed authentication attempts"})
	}
	// Browsers open their native credentials dialog on this challenge, so it is only sent to other clients
	if !isBrowserRequest(ctx) {
		ctx.Set(fiber.HeaderWWWAuthenticate, "Basic")
	}
	return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
}

// authenticateBasic authenticates the request with its login session or, without a valid one, with its Authorization:
// Basic header, under the failure limiter. The result is computed once and kept in the locals of the request, so that
// a wrong password counts as a single failure and is checked once even when several handlers ask for it.
func (c *Config) authenticateBasic(ctx *fiber.Ctx) *basicAuthentication {
	if authentication, ok := ctx.Locals(localsBasicAuthentication).(*basicAuthentication); ok {
		return authentication
	}
	authentication := &basicAuthentication{}
	ctx.Locals(localsBasicAuthentication, authentication)
	now := time.Now()
	if username, ok := c.Basic.sessionUsername(ctx.Cookies(cookieNameSession), now); ok {
		authentication.username, authentication.authenticated = username, true
	} else if username, password, ok := basicCredentials(ctx); ok {
		clientIP := requestClientIP(ctx)
		limiter := c.failureLimiter()
		if blocked, retryAfter := limiter.Blocked(clientIP, now); blocked {
			authentication.retryAfter = retryAfter
		} else if c.Basic.checkCredentials(username, password) {
			authentication.username, authentication.authenticated = c.Basic.Username, true
		} else {
			limiter.Failure(clientIP, now)
			logr.Debugf("[security.authenticateBasic] Wrong credentials in Authorization header from %s", clientIP)
		}
	}
	if authentication.authenticated {
		ctx.Locals("username", authentication.username)
	}
	return authentication
}

// failureLimiter returns the limiter of the authentication failures of this configuration: a reload starts a new count
func (c *Config) failureLimiter() *failureLimiter {
	c.limiterOnce.Do(func() {
		c.limiter = newFailureLimiter(failureLimiterMaximumFailures, failureLimiterMaximumKeys)
	})
	return c.limiter
}

// sessionUsername returns the username of the login session of the token, if it exists, has not expired and was
// created with the current credential. A session that expired or was created with another credential is deleted.
func (c *BasicConfig) sessionUsername(token string, now time.Time) (string, bool) {
	if !isLoginSessionToken(token) {
		return "", false
	}
	loginSessionStore, ok := store.GetLoginSessionStore()
	if !ok {
		return "", false
	}
	tokenHash := hashLoginSessionToken(token)
	session, err := loginSessionStore.GetLoginSession(tokenHash)
	if err != nil {
		if !errors.Is(err, common.ErrLoginSessionNotFound) {
			logr.Errorf("[security.sessionUsername] Failed to get the login session: %s", err.Error())
		}
		return "", false
	}
	if !now.Before(session.ExpiresAt) || subtle.ConstantTimeCompare([]byte(session.CredentialFingerprint), []byte(c.credentialFingerprint())) != 1 {
		if err := loginSessionStore.DeleteLoginSession(tokenHash); err != nil {
			logr.Errorf("[security.sessionUsername] Failed to delete an invalid login session: %s", err.Error())
		}
		return "", false
	}
	return session.Username, true
}

// checkCredentials returns whether the username and the password are the configured ones. Both are always checked, the
// username in constant time and the password with bcrypt against the configured hash, so that the response time does
// not tell whether the username is right.
func (c *BasicConfig) checkCredentials(username, password string) bool {
	expectedUsername := sha256.Sum256([]byte(c.Username))
	receivedUsername := sha256.Sum256([]byte(username))
	usernameMatches := subtle.ConstantTimeCompare(expectedUsername[:], receivedUsername[:]) == 1
	passwordHash, _ := base64.URLEncoding.DecodeString(c.PasswordBcryptHashBase64Encoded)
	passwordMatches := compareHashAndPassword(passwordHash, []byte(password)) == nil
	return usernameMatches && passwordMatches
}

// credentialFingerprint returns the SHA-256 hash, in hexadecimal, of the configured username and password hash
func (c *BasicConfig) credentialFingerprint() string {
	sum := sha256.Sum256([]byte(strconv.Itoa(len(c.Username)) + ":" + c.Username + c.PasswordBcryptHashBase64Encoded))
	return hex.EncodeToString(sum[:])
}

// basicCredentials returns the username and the password of the Authorization: Basic header of the request
func basicCredentials(ctx *fiber.Ctx) (string, string, bool) {
	scheme, encoded, found := strings.Cut(strings.TrimSpace(ctx.Get(fiber.HeaderAuthorization)), " ")
	if !found || !strings.EqualFold(scheme, "basic") {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", "", false
	}
	return strings.Cut(string(decoded), ":")
}

// isBrowserRequest returns whether the request comes from a browser or from the frontend
func isBrowserRequest(ctx *fiber.Ctx) bool {
	// Fork: an EventSource cannot send X-Requested-With and, on an HTTP page outside of localhost, the browser does not
	// send Sec-Fetch-* either, but it always accepts text/event-stream
	return len(ctx.Get("Sec-Fetch-Site")) > 0 || len(ctx.Get("Sec-Fetch-Mode")) > 0 || len(ctx.Get(fiber.HeaderXRequestedWith)) > 0 ||
		strings.Contains(ctx.Get(fiber.HeaderAccept), "text/event-stream")
}

// requestClientIP returns the IP address of the client of the request
func requestClientIP(ctx *fiber.Ctx) netip.Addr {
	if clientIP, ok := ctx.Locals(LocalsClientIP).(netip.Addr); ok {
		return clientIP
	}
	remoteIP, _ := netip.AddrFromSlice(ctx.Context().RemoteIP())
	return remoteIP.Unmap()
}

// isLoginSessionToken returns whether the value has the format of a login session token, without querying the storage
func isLoginSessionToken(token string) bool {
	if len(token) != base64.RawURLEncoding.EncodedLen(loginSessionTokenSize) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil
}

// hashLoginSessionToken returns the SHA-256 hash of the token, in hexadecimal: only the hash is stored
func hashLoginSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// setLoginSessionCookie sets the session cookie: Secure when the connection is TLS or X-Forwarded-Proto is https
func setLoginSessionCookie(ctx *fiber.Ctx, token string, maxAge int, expires time.Time) {
	ctx.Cookie(&fiber.Cookie{
		Name:     cookieNameSession,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expires,
		Secure:   isSecureRequest(ctx),
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

// isSecureRequest returns whether the connection is TLS or the reverse proxy says, with X-Forwarded-Proto, that it is
func isSecureRequest(ctx *fiber.Ctx) bool {
	if ctx.Context().IsTLS() {
		return true
	}
	forwardedProto, _, _ := strings.Cut(ctx.Get(fiber.HeaderXForwardedProto), ",")
	return strings.EqualFold(strings.TrimSpace(forwardedProto), "https")
}
