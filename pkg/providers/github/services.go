package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

// Common service patterns for Ethereum networks.
var servicePatterns = map[string]string{
	"faucet":          "https://faucet.%s",
	"json_rpc":        "https://rpc.%s",
	"beacon_rpc":      "https://beacon.%s",
	"explorer":        "https://explorer.%s",
	"forkmon":         "https://forkmon.%s",
	"assertoor":       "https://assertoor.%s",
	"dora":            "https://dora.%s",
	"checkpoint_sync": "https://checkpoint-sync.%s",
	"ethstats":        "https://ethstats.%s",
	"tracoor":         "https://tracoor.%s",
}

// Special case services with custom patterns.
var specialServicePatterns = map[string]func(string) string{
	"beacon_explorer": func(domain string) string {
		parts := strings.Split(domain, ".")
		if len(parts) > 0 {
			return fmt.Sprintf("https://%s.beaconcha.in", parts[0])
		}

		return ""
	},
	"blobscan": func(domain string) string {
		// Global blobscan service.
		return "https://blobscan.com"
	},
	"devnet_spec": func(domain string) string {
		parts := strings.Split(domain, ".")
		if len(parts) == 0 {
			return ""
		}

		domainPrefix := parts[0]

		// Extract prefix (e.g., "pectra", "dencun") and network name.
		prefixParts := strings.SplitN(domainPrefix, "-", 2)
		if len(prefixParts) != 2 {
			return ""
		}

		prefix := prefixParts[0]      // e.g., "pectra", "dencun".
		networkName := prefixParts[1] // e.g., "msf-1", "devnet-6".

		// Construct the URL using the extracted prefix and network name.
		return fmt.Sprintf("https://github.com/ethpandaops/%s-devnets/tree/master/network-configs/%s/metadata",
			prefix, networkName)
	},
}

// getServiceURLs constructs and validates service URLs for a network.
func (p *Provider) getServiceURLs(ctx context.Context, domain string) *discovery.ServiceURLs {
	services := &discovery.ServiceURLs{}
	client := &http.Client{
		Timeout: 2 * time.Second, // Short timeout for quick checks.
	}

	p.log.WithField("domain", domain).Debug("Checking service URLs")

	// Use channels to collect results from goroutines.
	type urlCheckResult struct {
		serviceKey string
		url        string
		valid      bool
	}

	var (
		resultCh  = make(chan urlCheckResult)
		numChecks int
	)

	// Start goroutines for common services
	for serviceKey, pattern := range servicePatterns {
		numChecks++

		go func(key, pattern string) {
			url := fmt.Sprintf(pattern, domain)
			valid := p.isURLValid(ctx, client, url)

			resultCh <- urlCheckResult{
				serviceKey: key,
				url:        url,
				valid:      valid,
			}
		}(serviceKey, pattern)
	}

	// Start goroutines for special services that need validation
	for serviceKey, patternFunc := range specialServicePatterns {
		// Skip static URLs that don't need validation.
		//nolint:goconst // No need.
		if serviceKey == "devnet_spec" || serviceKey == "blobscan" {
			continue
		}

		url := patternFunc(domain)
		if url != "" {
			numChecks++

			go func(key, url string) {
				valid := p.isURLValid(ctx, client, url)

				resultCh <- urlCheckResult{
					serviceKey: key,
					url:        url,
					valid:      valid,
				}
			}(serviceKey, url)
		}
	}

	// Collect results
	for i := 0; i < numChecks; i++ {
		result := <-resultCh

		p.log.WithFields(map[string]interface{}{
			"service": result.serviceKey,
			"url":     result.url,
			"valid":   result.valid,
		}).Debug("Checked service URL")

		if result.valid {
			switch result.serviceKey {
			case "faucet":
				services.Faucet = result.url
			case "json_rpc":
				services.JSONRPC = result.url
			case "beacon_rpc":
				services.BeaconRPC = result.url
			case "explorer":
				services.Explorer = result.url
			case "forkmon":
				services.Forkmon = result.url
			case "assertoor":
				services.Assertoor = result.url
			case "dora":
				services.Dora = result.url
			case "checkpoint_sync":
				services.CheckpointSync = result.url
			case "ethstats":
				services.Ethstats = result.url
			case "beacon_explorer":
				services.BeaconExplorer = result.url
			case "tracoor":
				services.Tracoor = result.url
			}
		}
	}

	// Add static services that don't need validation
	for serviceKey, patternFunc := range specialServicePatterns {
		if serviceKey == "devnet_spec" || serviceKey == "blobscan" {
			url := patternFunc(domain)
			if url != "" {
				switch serviceKey {
				case "devnet_spec":
					services.DevnetSpec = url
				case "blobscan":
					// Always use the global blobscan.com
					services.Blobscan = url
				}

				p.log.WithFields(map[string]interface{}{
					"service": serviceKey,
					"url":     url,
					"valid":   true, // Assumed valid for static URLs
				}).Debug("Added static service URL without validation")
			}
		}
	}

	return services
}

// isURLValid checks if a URL is reachable.
func (p *Provider) isURLValid(ctx context.Context, client *http.Client, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx, 3xx, and some 4xx status codes as valid
	// 404 might be valid for an API that exists but the endpoint is not found
	return resp.StatusCode < 500
}
