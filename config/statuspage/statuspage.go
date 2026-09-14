// Package statuspage contains the configuration of the public status pages
package statuspage

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultRateLimit is the default number of costly requests per minute accepted from each client IP
	DefaultRateLimit = 120

	// MaximumSlugLength is the maximum length of a slug
	MaximumSlugLength = 64

	// MaximumTitleLength is the maximum number of characters of a title
	MaximumTitleLength = 100

	// MaximumDescriptionLength is the maximum number of characters of a description
	MaximumDescriptionLength = 1000

	// MaximumGroups is the maximum number of groups selected by a page
	MaximumGroups = 50

	// MaximumGroupLength is the maximum number of characters of a group name
	MaximumGroupLength = 200

	// MaximumEndpoints is the maximum number of endpoint keys selected by a page, and the maximum number of endpoints
	// shown on a page
	MaximumEndpoints = 200

	// MaximumEndpointKeyLength is the maximum length of an endpoint key
	MaximumEndpointKeyLength = 400
)

var (
	// ErrInvalidSlug is returned when a slug does not match the allowed format
	ErrInvalidSlug = errors.New("slug must have 1 to 64 lowercase letters, digits or hyphens, and must not start or end with a hyphen")

	// ErrReservedSlug is returned when a slug is reserved by the administration routes
	ErrReservedSlug = errors.New("slug is reserved")

	// ErrDuplicateSlug is returned when two pages of the configuration file have the same slug
	ErrDuplicateSlug = errors.New("slug is used by more than one status page")

	// ErrInvalidTitle is returned when a title is empty or too long
	ErrInvalidTitle = fmt.Errorf("title must have 1 to %d characters", MaximumTitleLength)

	// ErrDescriptionTooLong is returned when a description is too long
	ErrDescriptionTooLong = fmt.Errorf("description must have at most %d characters", MaximumDescriptionLength)

	// ErrEmptySelection is returned when a page selects no group and no endpoint
	ErrEmptySelection = errors.New("status page must select at least one group or endpoint")

	// ErrInvalidGroups is returned when the groups of a page are invalid
	ErrInvalidGroups = errors.New("invalid groups")

	// ErrInvalidEndpoints is returned when the endpoint keys of a page are invalid
	ErrInvalidEndpoints = errors.New("invalid endpoints")

	// ErrInvalidTrustedProxy is returned when an entry of trusted-proxies is neither an IP address nor a CIDR
	ErrInvalidTrustedProxy = errors.New("status-pages.trusted-proxies entries must be IP addresses or CIDRs")

	// ErrInvalidRateLimit is returned when rate-limit is negative
	ErrInvalidRateLimit = errors.New("status-pages.rate-limit must not be negative")

	slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

	// reservedSlugs collide with the static segments of the administration routes (/api/v1/admin/status-pages/<segment>)
	reservedSlugs = map[string]struct{}{"new": {}, "options": {}, "preview": {}, "validate": {}}
)

// Config is the configuration of the public status pages
type Config struct {
	// Enabled is whether the public status pages are served. Defaults to true.
	Enabled *bool `yaml:"enabled,omitempty"`

	// TrustedProxies is the list of IP addresses or CIDRs of the reverse proxies whose X-Forwarded-For header is used
	// to identify the client IP
	TrustedProxies []string `yaml:"trusted-proxies,omitempty"`

	// RateLimit is the number of costly requests per minute accepted from each client IP. 0 disables the limit.
	// Defaults to DefaultRateLimit.
	RateLimit *int `yaml:"rate-limit,omitempty"`

	// Pages is the list of status pages defined in the configuration file
	Pages []*Page `yaml:"pages,omitempty"`

	trustedProxyPrefixes []netip.Prefix
}

// IsEnabled returns whether the public status pages are served. It is safe to call on a nil Config.
func (c *Config) IsEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// GetRateLimit returns the number of costly requests per minute accepted from each client IP, 0 meaning unlimited.
// It is safe to call on a nil Config.
func (c *Config) GetRateLimit() int {
	if c == nil || c.RateLimit == nil {
		return DefaultRateLimit
	}
	return *c.RateLimit
}

// TrustedProxyPrefixes returns the normalized trusted-proxies. It is safe to call on a nil Config.
func (c *Config) TrustedProxyPrefixes() []netip.Prefix {
	if c == nil {
		return nil
	}
	return c.trustedProxyPrefixes
}

// ValidateAndSetDefaults validates the configuration and normalizes its values
func (c *Config) ValidateAndSetDefaults() error {
	if c.RateLimit != nil && *c.RateLimit < 0 {
		return ErrInvalidRateLimit
	}
	prefixes := make([]netip.Prefix, 0, len(c.TrustedProxies))
	for _, trustedProxy := range c.TrustedProxies {
		prefix, err := parseTrustedProxy(trustedProxy)
		if err != nil {
			return err
		}
		prefixes = append(prefixes, prefix)
	}
	c.trustedProxyPrefixes = prefixes
	slugs := make(map[string]struct{}, len(c.Pages))
	for _, page := range c.Pages {
		if page == nil {
			return fmt.Errorf("%w: empty status page", ErrEmptySelection)
		}
		if err := page.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid status page %q: %w", page.Slug, err)
		}
		if _, exists := slugs[page.Slug]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateSlug, page.Slug)
		}
		slugs[page.Slug] = struct{}{}
	}
	return nil
}

// Page is the definition of a public status page
type Page struct {
	// Slug identifies the page in its public path (/status/<slug>). It cannot be changed.
	Slug string `yaml:"slug" json:"slug"`

	// Title is shown at the top of the page
	Title string `yaml:"title" json:"title"`

	// Description is shown below the title, as plain text
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Groups selects every enabled endpoint whose group is in the list, including the ones created later
	Groups []string `yaml:"groups,omitempty" json:"groups,omitempty"`

	// Endpoints selects endpoints by key
	Endpoints []string `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	// Enabled is whether the page is published. Pages of the configuration file default to true.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// IsEnabled returns whether the page is published, defaulting to true
func (p *Page) IsEnabled() bool {
	return p.Enabled == nil || *p.Enabled
}

// ValidateAndSetDefaults validates the page and normalizes its values: the title, the description and the group names
// are trimmed, and the endpoint keys are trimmed and converted to lowercase
func (p *Page) ValidateAndSetDefaults() error {
	p.Slug = strings.TrimSpace(p.Slug)
	if err := ValidateSlug(p.Slug); err != nil {
		return err
	}
	p.Title = strings.TrimSpace(p.Title)
	if length := utf8.RuneCountInString(p.Title); length == 0 || length > MaximumTitleLength {
		return ErrInvalidTitle
	}
	p.Description = strings.TrimSpace(p.Description)
	if utf8.RuneCountInString(p.Description) > MaximumDescriptionLength {
		return ErrDescriptionTooLong
	}
	groups, err := normalizeList(p.Groups, MaximumGroups, MaximumGroupLength, strings.TrimSpace)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidGroups, err)
	}
	endpoints, err := normalizeList(p.Endpoints, MaximumEndpoints, MaximumEndpointKeyLength, func(key string) string {
		return strings.ToLower(strings.TrimSpace(key))
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidEndpoints, err)
	}
	if len(groups) == 0 && len(endpoints) == 0 {
		return ErrEmptySelection
	}
	p.Groups, p.Endpoints = groups, endpoints
	return nil
}

// ValidateSlug returns an error if the slug does not match the allowed format or is reserved
func ValidateSlug(slug string) error {
	if len(slug) > MaximumSlugLength || !slugPattern.MatchString(slug) {
		return ErrInvalidSlug
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		return fmt.Errorf("%w: %s", ErrReservedSlug, slug)
	}
	return nil
}

// NormalizeGroup returns the group name as compared with the groups of a page
func NormalizeGroup(group string) string {
	return strings.TrimSpace(group)
}

func normalizeList(values []string, maximumItems, maximumLength int, normalize func(string) string) ([]string, error) {
	if len(values) > maximumItems {
		return nil, fmt.Errorf("at most %d entries are allowed", maximumItems)
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalize(value)
		if length := utf8.RuneCountInString(value); length == 0 || length > maximumLength {
			return nil, fmt.Errorf("entries must have 1 to %d characters", maximumLength)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf("duplicate entry %q", value)
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func parseTrustedProxy(value string) (netip.Prefix, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("%w: %q", ErrInvalidTrustedProxy, value)
		}
		if prefix.Addr().Is4In6() {
			bits := prefix.Bits() - 96
			if bits < 0 {
				return netip.Prefix{}, fmt.Errorf("%w: %q", ErrInvalidTrustedProxy, value)
			}
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), bits)
		}
		return prefix.Masked(), nil
	}
	addr, err := netip.ParseAddr(value)
	if err != nil || addr.Zone() != "" {
		return netip.Prefix{}, fmt.Errorf("%w: %q", ErrInvalidTrustedProxy, value)
	}
	addr = addr.Unmap()
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}
