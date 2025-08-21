package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// Fetcher is responsible for fetching data from Dora APIs.
type Fetcher struct {
	log     *logrus.Entry
	client  *http.Client
	timeout time.Duration
}

// NewFetcher creates a new Fetcher instance.
func NewFetcher(log *logrus.Entry) *Fetcher {
	return &Fetcher{
		log: log.WithField("component", "inventory_fetcher"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// CheckHealth verifies if the Dora API is accessible and responding.
func (f *Fetcher) CheckHealth(ctx context.Context, doraURL string) error {
	// Check if the consensus clients endpoint is accessible
	consensusEndpoint, err := url.JoinPath(doraURL, "api/v1/clients/consensus")
	if err != nil {
		return fmt.Errorf("failed to construct consensus endpoint URL: %w", err)
	}

	// Use a shorter timeout for health checks
	healthClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Test consensus endpoint with HEAD request (lighter than GET)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, consensusEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := healthClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// If HEAD fails or returns non-OK, try a GET request (some servers don't support HEAD)
		if resp != nil {
			resp.Body.Close()
		}

		getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, consensusEndpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create GET health check request: %w", err)
		}

		resp, err = healthClient.Do(getReq)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
	}
	defer resp.Body.Close()

	// Accept only 200 OK status for API endpoints
	if resp.StatusCode == http.StatusOK {
		f.log.WithFields(logrus.Fields{
			"url":    doraURL,
			"status": resp.StatusCode,
		}).Debug("Dora health check passed")

		return nil
	}

	return fmt.Errorf("dora API not available (status: %d)", resp.StatusCode)
}

// FetchConsensusClients fetches all consensus clients from the Dora API.
func (f *Fetcher) FetchConsensusClients(ctx context.Context, doraURL string) ([]DoraConsensusClient, error) {
	endpoint, err := url.JoinPath(doraURL, "api/v1/clients/consensus")
	if err != nil {
		return nil, fmt.Errorf("failed to construct consensus clients URL: %w", err)
	}

	data, err := f.fetchWithRetry(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch consensus clients: %w", err)
	}

	var response DoraConsensusResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse consensus clients response: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"url":     endpoint,
		"clients": len(response.Clients),
	}).Debug("Fetched consensus clients from Dora API")

	return response.Clients, nil
}

// FetchExecutionClients fetches all execution clients from the Dora API.
func (f *Fetcher) FetchExecutionClients(ctx context.Context, doraURL string) ([]DoraExecutionClient, error) {
	endpoint, err := url.JoinPath(doraURL, "api/v1/clients/execution")
	if err != nil {
		return nil, fmt.Errorf("failed to construct execution clients URL: %w", err)
	}

	data, err := f.fetchWithRetry(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch execution clients: %w", err)
	}

	var response DoraExecutionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse execution clients response: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"url":     endpoint,
		"clients": len(response.Clients),
	}).Debug("Fetched execution clients from Dora API")

	return response.Clients, nil
}

// fetchWithRetry performs an HTTP GET request with retry logic.
func (f *Fetcher) fetchWithRetry(ctx context.Context, url string) ([]byte, error) {
	const maxRetries = 3

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		data, err := f.doFetch(ctx, url)
		if err == nil {
			return data, nil
		}

		lastErr = err

		if attempt < maxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			f.log.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"error":   err,
				"backoff": backoff,
			}).Warn("Request failed, retrying")

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// doFetch performs a single HTTP GET request.
func (f *Fetcher) doFetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
