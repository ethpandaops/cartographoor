package validatorranges

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetcher handles downloading inventory files from GitHub.
type Fetcher struct {
	httpClient *http.Client
}

// NewFetcher creates a new Fetcher instance.
func NewFetcher() *Fetcher {
	return &Fetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchInventoryFile downloads a single inventory file from a URL.
func (f *Fetcher) FetchInventoryFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch inventory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// BuildInventoryURLs constructs URLs for standard inventory files for a network.
func (f *Fetcher) BuildInventoryURLs(repo, network string) []string {
	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/master/ansible/inventories/%s", repo, network)

	return []string{
		fmt.Sprintf("%s/inventory.ini", baseURL),
		fmt.Sprintf("%s/hetzner_inventory.ini", baseURL),
	}
}

// FetchMultiple fetches inventory files from multiple URLs, returning the content and successful URLs.
func (f *Fetcher) FetchMultiple(ctx context.Context, urls []string) ([][]byte, []string, error) {
	contents := make([][]byte, 0, len(urls))
	successfulURLs := make([]string, 0, len(urls))

	for _, url := range urls {
		data, err := f.FetchInventoryFile(ctx, url)
		if err != nil {
			// Skip failed fetches (e.g., file doesn't exist)
			continue
		}

		contents = append(contents, data)
		successfulURLs = append(successfulURLs, url)
	}

	if len(contents) == 0 {
		return nil, nil, fmt.Errorf("no inventory files could be fetched")
	}

	return contents, successfulURLs, nil
}
