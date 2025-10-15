package inventory

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Validator validates client URLs using DNS resolution and HTTP checks.
type Validator struct {
	log         *logrus.Entry
	dnsTimeout  time.Duration
	httpTimeout time.Duration
	httpClient  *http.Client
	sem         *semaphore.Weighted
}

// NewValidator creates a new Validator instance.
//
// Parameters:
//   - log: logger with component field already set
//   - dnsTimeout: timeout for each DNS lookup (e.g., 3*time.Second)
//   - maxConcurrent: max concurrent validation operations (DNS + HTTPS HEAD requests)
func NewValidator(
	log *logrus.Entry,
	dnsTimeout time.Duration,
	maxConcurrent int64,
) *Validator {
	httpTimeout := dnsTimeout * 2 // HTTP timeout is 2x DNS timeout (e.g., 6s)

	return &Validator{
		log:         log.WithField("component", "inventory_validator"),
		dnsTimeout:  dnsTimeout,
		httpTimeout: httpTimeout,
		httpClient: &http.Client{
			Timeout: httpTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
		sem: semaphore.NewWeighted(maxConcurrent),
	}
}

// ValidateClient validates all URLs for a client.
// If ANY URL validation fails, the entire client is considered invalid.
// SSH URLs are validated via DNS only.
// BeaconAPI and RPC URLs are validated via HTTPS HEAD request.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - client: the client with URLs to validate
//
// Returns:
//   - error: non-nil if any URL validation fails
func (v *Validator) ValidateClient(
	ctx context.Context,
	client ClientInfo,
) error {
	// Validate SSH URL (DNS only - can't HTTP check SSH protocol)
	if client.SSH != "" {
		if err := v.validateHostname(ctx, "ssh", client.SSH); err != nil {
			return fmt.Errorf("SSH URL validation failed: %w", err)
		}
	}

	// Validate BeaconAPI URL (HTTPS HEAD request)
	if client.BeaconAPI != "" {
		if err := v.validateHTTPEndpoint(ctx, "beacon-api", client.BeaconAPI); err != nil {
			return fmt.Errorf("BeaconAPI URL validation failed: %w", err)
		}
	}

	// Validate RPC URL (HTTPS HEAD request)
	if client.RPC != "" {
		if err := v.validateHTTPEndpoint(ctx, "rpc", client.RPC); err != nil {
			return fmt.Errorf("RPC URL validation failed: %w", err)
		}
	}

	return nil
}

// validateHostname performs DNS lookup for a hostname.
//
// Parameters:
//   - ctx: context for cancellation
//   - urlType: "ssh", "beacon-api", or "rpc" for logging
//   - url: the full URL string
func (v *Validator) validateHostname(
	ctx context.Context,
	urlType,
	url string,
) error {
	hostname := v.extractHostname(url)

	// Acquire semaphore to limit concurrent DNS lookups
	if err := v.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}
	defer v.sem.Release(1)

	// Create context with timeout for DNS lookup
	lookupCtx, cancel := context.WithTimeout(ctx, v.dnsTimeout)
	defer cancel()

	// Perform DNS lookup
	addrs, err := net.DefaultResolver.LookupHost(lookupCtx, hostname)
	if err != nil {
		v.log.WithFields(logrus.Fields{
			"url_type": urlType,
			"url":      url,
			"hostname": hostname,
			"error":    err,
		}).Warn("DNS validation failed")

		return fmt.Errorf(
			"DNS lookup failed for %s URL %s (hostname: %s): %w",
			urlType,
			url,
			hostname,
			err,
		)
	}

	// Verify we got at least one address
	if len(addrs) == 0 {
		v.log.WithFields(logrus.Fields{
			"url_type": urlType,
			"url":      url,
			"hostname": hostname,
		}).Warn("DNS validation returned no addresses")

		return fmt.Errorf(
			"DNS lookup returned no addresses for %s URL %s (hostname: %s)",
			urlType,
			url,
			hostname,
		)
	}

	return nil
}

// validateHTTPEndpoint validates an HTTPS endpoint via HTTP HEAD request.
// Only HTTPS is attempted - HTTP fallback is not used.
// DNS resolution is performed implicitly by the HTTP client.
func (v *Validator) validateHTTPEndpoint(ctx context.Context, urlType, hostname string) error {
	// Acquire semaphore for HTTP request
	if err := v.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire semaphore for %s URL validation: %w", urlType, err)
	}
	defer v.sem.Release(1)

	// Only try HTTPS - do not fall back to HTTP
	httpsURL := "https://" + hostname
	if err := v.doHTTPHead(ctx, httpsURL); err != nil {
		v.log.WithFields(logrus.Fields{
			"url_type": urlType,
			"hostname": hostname,
			"error":    err,
		}).Warn("Client URL validation failed")

		return fmt.Errorf("failed to validate %s URL %s via HTTPS: %w", urlType, hostname, err)
	}

	return nil
}

// doHTTPHead performs an HTTP HEAD request to check if an endpoint is accessible.
func (v *Validator) doHTTPHead(ctx context.Context, url string) error {
	reqCtx, cancel := context.WithTimeout(ctx, v.httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP HEAD request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP HEAD request failed: %w", err)
	}
	defer resp.Body.Close()

	// Accept 2xx (success), 3xx (redirects), or 401/404/405 (service exists but requires auth/endpoint missing/method not allowed)
	// Since we only use HTTPS, redirects are acceptable
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	// 401/404/405 means service is running but requires auth, endpoint doesn't exist, or method not allowed
	// This is acceptable - the service is accessible
	if resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusNotFound ||
		resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}

	return fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
}

// extractHostname extracts hostname from URL.
//
// Handles different URL formats:
//   - SSH URL: "devops@hostname" → "hostname"
//   - BeaconAPI/RPC: "hostname" → "hostname" (no change)
func (v *Validator) extractHostname(url string) string {
	// Handle SSH URLs with user@host format
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// For BeaconAPI/RPC URLs, return as-is
	return url
}
