package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	gh "github.com/google/go-github/v53/github"
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
			Repositories []discovery.GitHubRepositoryConfig `mapstructure:"repositories"`
			Token        string                             `mapstructure:"token"`
		}{
			Repositories: []discovery.GitHubRepositoryConfig{
				{
					Name:       "ethpandaops/dencun-devnets",
					NamePrefix: "",
				},
			},
			Token: "",
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

	for key, network := range networks {
		// Key should be the same as the original network name since we didn't use a prefix
		assert.Equal(t, key, network.Name)
		assert.True(t, expectedNetworks[key], "Unexpected network: %s", key)
		assert.Equal(t, "ethpandaops/dencun-devnets", network.Repository)
		assert.Equal(t, "network-configs/"+network.Name, network.Path)
		assert.Contains(t, network.URL, "github.com/ethpandaops/dencun-devnets/tree/main/network-configs/")
		assert.Equal(t, "active", network.Status)
		assert.WithinDuration(t, time.Now(), network.LastUpdated, 10*time.Second)
	}
}

func TestProvider_DiscoverWithPrefix(t *testing.T) {
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

	// Configure discovery with a prefix
	config := discovery.Config{
		GitHub: struct {
			Repositories []discovery.GitHubRepositoryConfig `mapstructure:"repositories"`
			Token        string                             `mapstructure:"token"`
		}{
			Repositories: []discovery.GitHubRepositoryConfig{
				{
					Name:       "ethpandaops/dencun-devnets",
					NamePrefix: "dencun-",
				},
			},
			Token: "",
		},
	}

	// Test discovery
	networks, err := provider.Discover(context.Background(), config)
	require.NoError(t, err)

	// Validate the discovered networks
	require.Len(t, networks, 9)

	// Check specific networks with prefixes
	expectedNetworks := map[string]string{
		"dencun-devnet-10":   "devnet-10",
		"dencun-devnet-11":   "devnet-11",
		"dencun-devnet-12":   "devnet-12",
		"dencun-devnet-4":    "devnet-4",
		"dencun-devnet-5":    "devnet-5",
		"dencun-gsf-1":       "gsf-1",
		"dencun-gsf-2":       "gsf-2",
		"dencun-msf-1":       "msf-1",
		"dencun-sepolia-sf1": "sepolia-sf1",
	}

	for key, network := range networks {
		originalName, exists := expectedNetworks[key]
		assert.True(t, exists, "Unexpected network key: %s", key)
		assert.Equal(t, originalName, network.Name, "Network name should be the original name without prefix")
		assert.Equal(t, "ethpandaops/dencun-devnets", network.Repository)
		assert.Equal(t, "network-configs/"+originalName, network.Path)
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
			Repositories []discovery.GitHubRepositoryConfig `mapstructure:"repositories"`
			Token        string                             `mapstructure:"token"`
		}{
			Repositories: []discovery.GitHubRepositoryConfig{},
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
			Repositories []discovery.GitHubRepositoryConfig `mapstructure:"repositories"`
			Token        string                             `mapstructure:"token"`
		}{
			Repositories: []discovery.GitHubRepositoryConfig{
				{
					Name:       "invalid-repo-format",
					NamePrefix: "",
				},
			},
			Token: "",
		},
	}

	_, err = provider.Discover(context.Background(), config)
	require.NoError(t, err)
	// Should not error, just log a warning and return empty networks
}

func TestProvider_DiscoverNetworkStatus(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Mock GitHub API server
	ts := mockGitHubAPIWithStatus(t)
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
			Repositories []discovery.GitHubRepositoryConfig `mapstructure:"repositories"`
			Token        string                             `mapstructure:"token"`
		}{
			Repositories: []discovery.GitHubRepositoryConfig{
				{
					Name:       "ethpandaops/pectra-devnets",
					NamePrefix: "",
				},
			},
			Token: "",
		},
	}

	// Test discovery
	networks, err := provider.Discover(context.Background(), config)
	require.NoError(t, err)

	// Validate the discovered networks
	require.Len(t, networks, 3)

	// Check status for each network
	expected := map[string]string{
		"devnet-active":   "active",   // exists in kubernetes/
		"devnet-inactive": "inactive", // exists in kubernetes-archive/
		"devnet-unknown":  "unknown",  // doesn't exist in either
	}

	for networkName, expectedStatus := range expected {
		network, exists := networks[networkName]
		assert.True(t, exists, "Network %s should exist", networkName)
		if exists {
			assert.Equal(t, expectedStatus, network.Status, "Network %s should have status %s", networkName, expectedStatus)
		}
	}
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
			{"name": "network-configs", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs"},
			{"name": "kubernetes", "type": "dir", "html_url": "https://github.com/ethpandaops/dencun-devnets/tree/main/kubernetes"}
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

	// Mock kubernetes directory paths for all the networks to mark them as active
	networks := []string{"devnet-10", "devnet-11", "devnet-12", "devnet-4", "devnet-5", "gsf-1", "gsf-2", "msf-1", "sepolia-sf1"}
	for _, network := range networks {
		path := fmt.Sprintf("/ethpandaops/dencun-devnets/contents/kubernetes/%s", network)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "some-file.yaml", "type": "file", "html_url": "https://github.com/ethpandaops/dencun-devnets/blob/main/kubernetes/` + network + `/some-file.yaml"}
			]`))
		})
	}

	// Return test server
	server := httptest.NewServer(mux)
	return server
}

// Mock GitHub API with status checking.
func mockGitHubAPIWithStatus(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Mock repository contents
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name": ".github", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/.github"},
			{"name": "network-configs", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/network-configs"},
			{"name": "kubernetes", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/kubernetes"},
			{"name": "kubernetes-archive", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/kubernetes-archive"}
		]`))
	})

	// Mock network-configs contents
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents/network-configs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name": "devnet-active", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/network-configs/devnet-active"},
			{"name": "devnet-inactive", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/network-configs/devnet-inactive"},
			{"name": "devnet-unknown", "type": "dir", "html_url": "https://github.com/ethpandaops/pectra-devnets/tree/main/network-configs/devnet-unknown"}
		]`))
	})

	// Mock kubernetes directory - devnet-active exists here
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents/kubernetes/devnet-active", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name": "some-file.yaml", "type": "file", "html_url": "https://github.com/ethpandaops/pectra-devnets/blob/main/kubernetes/devnet-active/some-file.yaml"}
		]`))
	})

	// Mock kubernetes directory - devnet-inactive does not exist here
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents/kubernetes/devnet-inactive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Not Found", "documentation_url": "https://docs.github.com"}`))
	})

	// Mock kubernetes directory - devnet-unknown does not exist here
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents/kubernetes/devnet-unknown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Not Found", "documentation_url": "https://docs.github.com"}`))
	})

	// Mock kubernetes-archive directory - devnet-inactive exists here
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents/kubernetes-archive/devnet-inactive", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name": "some-file.yaml", "type": "file", "html_url": "https://github.com/ethpandaops/pectra-devnets/blob/main/kubernetes-archive/devnet-inactive/some-file.yaml"}
		]`))
	})

	// Mock kubernetes-archive directory - devnet-unknown does not exist here
	mux.HandleFunc("/ethpandaops/pectra-devnets/contents/kubernetes-archive/devnet-unknown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Not Found", "documentation_url": "https://docs.github.com"}`))
	})

	// Return test server
	server := httptest.NewServer(mux)

	return server
}
