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
	provider, err := NewProvider(log, nil)
	require.NoError(t, err)

	assert.Equal(t, "github", provider.Name())
}

func TestProvider_Discover(t *testing.T) {
	// Create test cases table to cover different scenarios
	testCases := []struct {
		name           string
		config         discovery.Config
		mockSetup      func(t *testing.T) *httptest.Server
		expectedError  string
		expectedCount  int
		expectedStatus map[string]string
		clientSetup    bool
	}{
		{
			name: "successful discovery with standard networks",
			config: discovery.Config{
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
					Token: "dummy-token",
				},
			},
			mockSetup:     mockGitHubAPI,
			expectedCount: 9,
			expectedStatus: map[string]string{
				"devnet-10":   "active",
				"devnet-11":   "active",
				"devnet-12":   "active",
				"devnet-4":    "active",
				"devnet-5":    "active",
				"gsf-1":       "active",
				"gsf-2":       "active",
				"msf-1":       "active",
				"sepolia-sf1": "active",
			},
			clientSetup: true,
		},
		{
			name: "successful discovery with name prefix",
			config: discovery.Config{
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
					Token: "dummy-token",
				},
			},
			mockSetup:     mockGitHubAPI,
			expectedCount: 9,
			expectedStatus: map[string]string{
				"dencun-devnet-10":   "active",
				"dencun-devnet-11":   "active",
				"dencun-devnet-12":   "active",
				"dencun-devnet-4":    "active",
				"dencun-devnet-5":    "active",
				"dencun-gsf-1":       "active",
				"dencun-gsf-2":       "active",
				"dencun-msf-1":       "active",
				"dencun-sepolia-sf1": "active",
			},
			clientSetup: true,
		},
		{
			name: "different network statuses",
			config: discovery.Config{
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
					Token: "dummy-token",
				},
			},
			mockSetup:     mockGitHubAPIWithStatus,
			expectedCount: 3,
			expectedStatus: map[string]string{
				"devnet-active":   "active",
				"devnet-inactive": "inactive",
				"devnet-unknown":  "unknown",
			},
			clientSetup: true,
		},
		{
			name: "invalid repository format",
			config: discovery.Config{
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
					Token: "dummy-token",
				},
			},
			expectedCount: 0,
			clientSetup:   true,
		},
		{
			name: "no repositories",
			config: discovery.Config{
				GitHub: struct {
					Repositories []discovery.GitHubRepositoryConfig `mapstructure:"repositories"`
					Token        string                             `mapstructure:"token"`
				}{
					Repositories: []discovery.GitHubRepositoryConfig{},
					Token:        "dummy-token",
				},
			},
			expectedError: "no repositories configured",
			clientSetup:   true,
		},
		{
			name: "no token",
			config: discovery.Config{
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
			},
			expectedError: "no GitHub token configured",
			clientSetup:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log := logrus.New()
			log.SetLevel(logrus.DebugLevel)

			provider, err := NewProvider(log, nil)
			require.NoError(t, err)

			// Set up mock API if needed
			var ts *httptest.Server
			if tc.mockSetup != nil {
				ts = tc.mockSetup(t)
				defer ts.Close()
			}

			// Configure client if needed
			if tc.clientSetup {
				if ts != nil {
					httpClient := &http.Client{
						Transport: &mockTransport{URL: ts.URL},
					}
					provider.githubClient = gh.NewClient(httpClient)
				} else {
					provider.githubClient = gh.NewClient(nil)
				}
			}

			// Test discovery
			networks, err := provider.Discover(context.Background(), tc.config)

			// Verify error cases
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)

				return
			}

			// For invalid repo format, we shouldn't error but return empty networks
			if tc.name == "invalid repository format" {
				require.NoError(t, err)
				assert.Empty(t, networks)

				return
			}

			// Verify successful cases
			require.NoError(t, err)
			assert.Len(t, networks, tc.expectedCount)

			// Verify network properties
			for key, network := range networks {
				// Check if the status matches expected
				if expected, ok := tc.expectedStatus[key]; ok {
					assert.Equal(t, expected, network.Status, "Network %s status should be %s", key, expected)
				}

				// Additional checks for standard networks
				if tc.name == "successful discovery with standard networks" {
					assert.Equal(t, key, network.Name)
					assert.Equal(t, "ethpandaops/dencun-devnets", network.Repository)
					assert.Equal(t, "network-configs/"+network.Name, network.Path)
					assert.Contains(t, network.URL, "github.com/ethpandaops/dencun-devnets/tree/main/network-configs/")
					assert.WithinDuration(t, time.Now(), network.LastUpdated, 10*time.Second)
				}

				// Additional checks for prefixed networks
				if tc.name == "successful discovery with name prefix" {
					// The names in the returned map have the prefix
					// but the original name in the network object does not
					originalName := network.Name
					prefixedName := key
					assert.Equal(t, "dencun-"+originalName, prefixedName)
					assert.Equal(t, "ethpandaops/dencun-devnets", network.Repository)
					assert.Equal(t, "network-configs/"+originalName, network.Path)
					assert.Contains(t, network.URL, "github.com/ethpandaops/dencun-devnets/tree/main/network-configs/")
					assert.WithinDuration(t, time.Now(), network.LastUpdated, 10*time.Second)
				}
			}
		})
	}
}

func TestServiceURLs(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	provider, err := NewProvider(log, nil)
	require.NoError(t, err)

	// Create a test server that simulates valid/invalid service endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Successful endpoints
		validEndpoints := map[string]bool{
			"/faucet.test-domain.com":          true,
			"/rpc.test-domain.com":             true,
			"/beacon.test-domain.com":          true,
			"/test-domain.beaconcha.in":        true,
			"/checkpoint-sync.test-domain.com": true,
		}

		if validEndpoints[r.URL.Path] {
			w.WriteHeader(http.StatusOK)

			return
		}

		// Server error responses for invalid services
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Test directly the isURLValid function with our test server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}

	// Test valid and invalid URLs
	t.Run("URL validation", func(t *testing.T) {
		// Valid URL
		assert.True(t, provider.isURLValid(ctx, client, server.URL+"/rpc.test-domain.com"))
		// Invalid URL
		assert.False(t, provider.isURLValid(ctx, client, server.URL+"/explorer.test-domain.com"))
	})

	// Verify service patterns and URL generation
	t.Run("Service patterns", func(t *testing.T) {
		// Standard patterns
		assert.Equal(t, "https://faucet.%s", servicePatterns["faucet"])
		assert.Equal(t, "https://rpc.%s", servicePatterns["json_rpc"])
		assert.Equal(t, "https://beacon.%s", servicePatterns["beacon_rpc"])

		// Special patterns
		t.Run("Beaconcha.in explorer", func(t *testing.T) {
			beaconchainURL := specialServicePatterns["beacon_explorer"]("test-domain.com")
			assert.Equal(t, "https://test-domain.beaconcha.in", beaconchainURL)
		})

		t.Run("Blobscan", func(t *testing.T) {
			blobscanURL := specialServicePatterns["blobscan"]("test-domain.com")
			assert.Equal(t, "https://blobscan.com", blobscanURL)
		})

		t.Run("Devnet spec with prefix", func(t *testing.T) {
			devnetDomain := "pectra-devnet-1.test-domain.com"
			devnetSpecURL := specialServicePatterns["devnet_spec"](devnetDomain)
			assert.Equal(t, "https://github.com/ethpandaops/pectra-devnets/tree/master/network-configs/devnet-1/metadata", devnetSpecURL)
		})

		t.Run("Devnet spec with invalid domain", func(t *testing.T) {
			emptySpecURL := specialServicePatterns["devnet_spec"]("")
			assert.Equal(t, "", emptySpecURL)
		})

		t.Run("Devnet spec with non-prefixed domain", func(t *testing.T) {
			nonPrefixDomain := "invalid-domain"
			nonPrefixedSpecURL := specialServicePatterns["devnet_spec"](nonPrefixDomain)
			expectedURL := "https://github.com/ethpandaops/invalid-devnets/tree/master/network-configs/domain/metadata"
			assert.Equal(t, expectedURL, nonPrefixedSpecURL)
		})
	})
}

// Mock HTTP transport to redirect GitHub API requests to our test server.
type mockTransport struct {
	URL string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request URL and override the host with our test server.
	req2 := new(http.Request)
	*req2 = *req
	req2.URL.Scheme = "http"
	req2.URL.Host = req.Host

	// Remove api.github.com from the path and update it.
	path := req.URL.Path
	path = path[len("/repos"):]
	req2.URL.Path = path

	// Set the full URL to our test server.
	req2.URL.Host = t.URL[7:] // Remove "http://"
	req2.Host = t.URL[7:]     // Remove "http://"

	// Send the request to our mock server
	return http.DefaultTransport.RoundTrip(req2)
}

func mockGitHubAPI(t *testing.T) *httptest.Server {
	t.Helper()

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
