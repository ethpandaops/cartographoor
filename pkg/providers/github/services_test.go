package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetServiceURLs tests the service URL discovery functionality.
func TestGetServiceURLs(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	provider, err := NewProvider(log)
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

	// Create a test context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test directly the isURLValid function with our test server
	client := &http.Client{Timeout: 2 * time.Second}

	// Test valid URL
	validReq, err := http.NewRequest("HEAD", server.URL+"/rpc.test-domain.com", nil)
	require.NoError(t, err)
	validResp, err := client.Do(validReq)
	require.NoError(t, err)
	defer validResp.Body.Close()
	assert.Less(t, validResp.StatusCode, 500)
	assert.True(t, provider.isURLValid(ctx, client, server.URL+"/rpc.test-domain.com"))

	// Test invalid URL
	invalidReq, err := http.NewRequest("HEAD", server.URL+"/explorer.test-domain.com", nil)
	require.NoError(t, err)
	invalidResp, err := client.Do(invalidReq)
	require.NoError(t, err)
	defer invalidResp.Body.Close()
	assert.GreaterOrEqual(t, invalidResp.StatusCode, 500)
	assert.False(t, provider.isURLValid(ctx, client, server.URL+"/explorer.test-domain.com"))

	// Verify the service patterns
	assert.Equal(t, "https://faucet.%s", servicePatterns["faucet"])
	assert.Equal(t, "https://rpc.%s", servicePatterns["json_rpc"])
	assert.Equal(t, "https://beacon.%s", servicePatterns["beacon_rpc"])

	// Test the beaconcha.in special pattern
	beaconchainURL := specialServicePatterns["beacon_explorer"]("test-domain.com")
	assert.Equal(t, "https://test-domain.beaconcha.in", beaconchainURL)

	// Test the blobscan special pattern
	blobscanURL := specialServicePatterns["blobscan"]("test-domain.com")
	assert.Equal(t, "https://blobscan.com", blobscanURL)

	// Test the devnet spec special pattern
	devnetDomain := "pectra-devnet-1.test-domain.com"
	devnetSpecURL := specialServicePatterns["devnet_spec"](devnetDomain)
	assert.Equal(t, "https://github.com/ethpandaops/pectra-devnets/tree/master/network-configs/devnet-1/metadata", devnetSpecURL)

	// Test the invalid domain for devnet spec
	emptySpecURL := specialServicePatterns["devnet_spec"]("")
	assert.Equal(t, "", emptySpecURL)

	// Test the non-prefixed domain for devnet spec
	nonPrefixDomain := "invalid-domain"
	nonPrefixedSpecURL := specialServicePatterns["devnet_spec"](nonPrefixDomain)
	expectedURL := "https://github.com/ethpandaops/invalid-devnets/tree/master/network-configs/domain/metadata"
	assert.Equal(t, expectedURL, nonPrefixedSpecURL)
}
