package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// DefaultSourceURL is the production cartographoor endpoint.
	DefaultSourceURL = "https://ethpandaops-platform-production-cartographoor.ams3.cdn.digitaloceanspaces.com/networks.json"

	// DefaultRefreshInterval is how often to fetch new data.
	DefaultRefreshInterval = 5 * time.Minute

	// DefaultRequestTimeout is the HTTP request timeout.
	DefaultRequestTimeout = 30 * time.Second

	// MinRefreshInterval is the minimum allowed refresh interval.
	MinRefreshInterval = 1 * time.Minute

	// MaxRefreshInterval is the maximum allowed refresh interval.
	MaxRefreshInterval = 24 * time.Hour
)

// Config holds configuration for a cartographoor client provider.
type Config struct {
	// SourceURL is the URL to fetch network data from.
	// Defaults to DefaultSourceURL if empty.
	SourceURL string

	// RefreshInterval is how often to poll for new data.
	// Must be between MinRefreshInterval and MaxRefreshInterval.
	// Defaults to DefaultRefreshInterval if zero.
	RefreshInterval time.Duration

	// RequestTimeout is the HTTP request timeout.
	// Defaults to DefaultRequestTimeout if zero.
	RequestTimeout time.Duration

	// HTTPClient is an optional custom HTTP client.
	// If nil, a default client with proper timeout will be created.
	HTTPClient *http.Client
}

// Validate validates the configuration and applies defaults.
func (c *Config) Validate() error {
	// Apply defaults
	if c.SourceURL == "" {
		c.SourceURL = DefaultSourceURL
	}

	if c.RefreshInterval == 0 {
		c.RefreshInterval = DefaultRefreshInterval
	}

	if c.RequestTimeout == 0 {
		c.RequestTimeout = DefaultRequestTimeout
	}

	// Validate SourceURL
	parsedURL, err := url.Parse(c.SourceURL)
	if err != nil {
		return fmt.Errorf("invalid SourceURL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("SourceURL must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Validate RefreshInterval
	if c.RefreshInterval < MinRefreshInterval {
		return fmt.Errorf("RefreshInterval must be at least %v, got: %v", MinRefreshInterval, c.RefreshInterval)
	}

	if c.RefreshInterval > MaxRefreshInterval {
		return fmt.Errorf("RefreshInterval must be at most %v, got: %v", MaxRefreshInterval, c.RefreshInterval)
	}

	// Validate RequestTimeout
	if c.RequestTimeout < 1*time.Second {
		return fmt.Errorf("RequestTimeout must be at least 1 second, got: %v", c.RequestTimeout)
	}

	if c.RequestTimeout >= c.RefreshInterval {
		return fmt.Errorf("RequestTimeout must be less than RefreshInterval")
	}

	// Create default HTTP client if needed
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: c.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	return nil
}
