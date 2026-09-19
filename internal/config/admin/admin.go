// Package admin contains the configuration of the web administration of endpoints
package admin

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	// ErrInvalidAllowedOrigin is returned when an entry of allowed-origins is not in the scheme://host[:port] format
	ErrInvalidAllowedOrigin = errors.New("admin.allowed-origins entries must be in the format scheme://host[:port]")
)

// Config is the configuration of the web administration of endpoints
type Config struct {
	// Enabled is whether endpoints can be managed through the administration API and web interface
	Enabled bool `yaml:"enabled,omitempty"`

	// AllowedSubjects is the list of OIDC subjects (sub claim) allowed to administer endpoints.
	// It is required when security.oidc is configured.
	AllowedSubjects []string `yaml:"allowed-subjects,omitempty"`

	// AllowedOrigins is the list of origins (scheme://host[:port]) accepted for requests that change endpoints.
	// When empty, the origin is derived from the Host header and the scheme of the request.
	AllowedOrigins []string `yaml:"allowed-origins,omitempty"`
}

// IsEnabled returns whether the administration is enabled. It is safe to call on a nil Config.
func (c *Config) IsEnabled() bool {
	return c != nil && c.Enabled
}

// ValidateAndSetDefaults validates the configuration and normalizes its values
func (c *Config) ValidateAndSetDefaults() error {
	subjects := make([]string, 0, len(c.AllowedSubjects))
	for _, subject := range c.AllowedSubjects {
		if subject = strings.TrimSpace(subject); len(subject) > 0 {
			subjects = append(subjects, subject)
		}
	}
	c.AllowedSubjects = subjects
	origins := make([]string, 0, len(c.AllowedOrigins))
	for _, origin := range c.AllowedOrigins {
		normalized, err := normalizeOrigin(origin)
		if err != nil {
			return err
		}
		origins = append(origins, normalized)
	}
	c.AllowedOrigins = origins
	return nil
}

// IsSubjectAllowed returns whether the OIDC subject is in AllowedSubjects, ignoring case
func (c *Config) IsSubjectAllowed(subject string) bool {
	for _, allowedSubject := range c.AllowedSubjects {
		if strings.EqualFold(allowedSubject, subject) {
			return true
		}
	}
	return false
}

func normalizeOrigin(origin string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || len(parsed.Host) == 0 ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidAllowedOrigin, origin)
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), nil
}
