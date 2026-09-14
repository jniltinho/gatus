package config

// ValidateStatusPagesConfig validates the structure of the public status pages configuration. Whether the groups and
// endpoints selected by the pages exist is only checked when the pages are loaded, after the managed endpoints.
func ValidateStatusPagesConfig(config *Config) error {
	if config.StatusPages == nil {
		return nil
	}
	return config.StatusPages.ValidateAndSetDefaults()
}
