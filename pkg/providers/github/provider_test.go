package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gh "github.com/google/go-github/v53/github"
	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	provider, err := NewProvider(log)
	require.NoError(t, err)

	assert.Equal(t, "github", provider.Name())
}

func TestProvider_Discover(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Mock GitHub API server
	ts := mockGitHubAPI(t)
	defer ts.Close()

	provider, err := NewProvider(log)
	require.NoError(t, err)

	// Create a custom GitHub client that uses our test server
	httpClient := &http.Client{
		Transport: &mockTransport{URL: ts.URL},
	}
	provider.client = gh.NewClient(httpClient)

	// Configure discovery
	config := discovery.Config{
		GitHub: struct {
			Repositories []string `mapstructure:"repositories"`
			Token        string   `mapstructure:"token"`
		}{
			Repositories: []string{"ethpandaops/dencun-devnets"},
			Token:        "",
		},
	}

	// Test discovery
	networks, err := provider.Discover(context.Background(), config)
	require.NoError(t, err)

	// Validate the discovered networks
	require.Len(t, networks, 9)

	// Check specific networks
	expectedNetworks := map[string]bool{
		"devnet-10":   true,
		"devnet-11":   true,
		"devnet-12":   true,
		"devnet-4":    true,
		"devnet-5":    true,
		"gsf-1":       true,
		"gsf-2":       true,
		"msf-1":       true,
		"sepolia-sf1": true,
	}

	for _, network := range networks {
		assert.True(t, expectedNetworks[network.Name], "Unexpected network: %s", network.Name)
		assert.Equal(t, "ethpandaops/dencun-devnets", network.Repository)
		assert.Equal(t, "network-configs/"+network.Name, network.Path)
		assert.Contains(t, network.URL, "github.com/ethpandaops/dencun-devnets/tree/main/network-configs/")
		assert.Equal(t, "active", network.Status)
		assert.WithinDuration(t, time.Now(), network.LastUpdated, 10*time.Second)
	}
}

func TestProvider_Discover_NoRepositories(t *testing.T) {
	log := logrus.New()
	provider, err := NewProvider(log)
	require.NoError(t, err)

	config := discovery.Config{
		GitHub: struct {
			Repositories []string `mapstructure:"repositories"`
			Token        string   `mapstructure:"token"`
		}{
			Repositories: []string{},
			Token:        "",
		},
	}

	_, err = provider.Discover(context.Background(), config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no repositories configured")
}

func TestProvider_Discover_InvalidRepository(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	provider, err := NewProvider(log)
	require.NoError(t, err)

	config := discovery.Config{
		GitHub: struct {
			Repositories []string `mapstructure:"repositories"`
			Token        string   `mapstructure:"token"`
		}{
			Repositories: []string{"invalid-repo-format"},
			Token:        "",
		},
	}

	_, err = provider.Discover(context.Background(), config)
	require.NoError(t, err)
	// Should not error, just log a warning and return empty networks
}

// Mock HTTP transport to redirect GitHub API requests to our test server
type mockTransport struct {
	URL string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request URL and override the host with our test server
	req2 := new(http.Request)
	*req2 = *req
	req2.URL.Scheme = "http"
	req2.URL.Host = req.Host
	
	// Remove api.github.com from the path and update it
	path := req.URL.Path
	path = path[len("/repos"):]
	req2.URL.Path = path
	
	// Set the full URL to our test server
	req2.URL.Host = t.URL[7:] // Remove "http://"
	req2.Host = t.URL[7:]     // Remove "http://"

	// Send the request to our mock server
	return http.DefaultTransport.RoundTrip(req2)
}

// Mock GitHub API
func mockGitHubAPI(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Mock repository contents
	mux.HandleFunc("/ethpandaops/dencun-devnets/contents", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name": ".editorconfig", "type": "file", "html_url": "https://github.com/ethpandaops/dencun-devnets/blob/main/.editorconfig"},
			{"name": ".gitattributes", "type": "file", "html_url": "https://github.com/ethpandaops/dencun-devnets/blob/main/.gitattributes"},
			{"name": ".github", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/.github"},
			{"name": ".gitignore", "type": "file", "html_url": "https://github.com/ethpandaops/dencun-devnets/blob/main/.gitignore"},
			{"name": "README.md", "type": "file", "html_url": "https://github.com/ethpandaops/dencun-devnets/blob/main/README.md"},
			{"name": "network-configs", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs"}
		]`))
	})

	// Mock network-configs contents
	mux.HandleFunc("/ethpandaops/dencun-devnets/contents/network-configs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name": "devnet-10", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-10"},
			{"name": "devnet-11", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-11"},
			{"name": "devnet-12", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-12"},
			{"name": "devnet-4", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-4"},
			{"name": "devnet-5", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-5"},
			{"name": "gsf-1", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/gsf-1"},
			{"name": "gsf-2", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/gsf-2"},
			{"name": "msf-1", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/msf-1"},
			{"name": "sepolia-sf1", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/sepolia-sf1"}
		]`))
	})

	// Return test server
	server := httptest.NewServer(mux)
	return server
}