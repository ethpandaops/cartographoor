package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractHostname tests hostname extraction from various URL formats.
func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH URL with user",
			url:      "devops@lighthouse-geth-1.fusaka.ethpandaops.io",
			expected: "lighthouse-geth-1.fusaka.ethpandaops.io",
		},
		{
			name:     "BeaconAPI URL",
			url:      "bn-lighthouse-geth-1.fusaka.ethpandaops.io",
			expected: "bn-lighthouse-geth-1.fusaka.ethpandaops.io",
		},
		{
			name:     "RPC URL",
			url:      "rpc-lighthouse-geth-1.fusaka.ethpandaops.io",
			expected: "rpc-lighthouse-geth-1.fusaka.ethpandaops.io",
		},
		{
			name:     "Simple hostname",
			url:      "google.com",
			expected: "google.com",
		},
		{
			name:     "SSH URL with different user",
			url:      "admin@server.example.com",
			expected: "server.example.com",
		},
		{
			name:     "URL with subdomain",
			url:      "api.service.example.com",
			expected: "api.service.example.com",
		},
	}

	log := logrus.New().WithField("test", "extract_hostname")
	validator := NewValidator(log, 3*time.Second, 10)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.extractHostname(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateClient_ValidHostnames tests successful validation with real resolvable hostnames.
func TestValidateClient_ValidHostnames(t *testing.T) {
	log := logrus.New().WithField("test", "valid_hostnames")
	validator := NewValidator(log, 5*time.Second, 15)
	ctx := context.Background()

	tests := []struct {
		name   string
		client ClientInfo
	}{
		{
			name: "Consensus client with valid hostnames",
			client: ClientInfo{
				ClientName: "test-lighthouse-1",
				ClientType: "lighthouse",
				SSH:        "devops@google.com",
				BeaconAPI:  "cloudflare.com",
			},
		},
		{
			name: "Execution client with valid hostnames",
			client: ClientInfo{
				ClientName: "test-geth-1",
				ClientType: "geth",
				SSH:        "devops@github.com",
				RPC:        "amazon.com",
			},
		},
		{
			name: "Client with only SSH",
			client: ClientInfo{
				ClientName: "test-client",
				ClientType: "test",
				SSH:        "devops@dns.google",
			},
		},
		{
			name: "Consensus client with all URLs",
			client: ClientInfo{
				ClientName: "test-prysm-1",
				ClientType: "prysm",
				SSH:        "devops@microsoft.com",
				BeaconAPI:  "apple.com",
			},
		},
		{
			name: "Execution client with all URLs",
			client: ClientInfo{
				ClientName: "test-nethermind-1",
				ClientType: "nethermind",
				SSH:        "devops@dns.google",
				RPC:        "example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateClient(ctx, tt.client)
			assert.NoError(t, err, "Expected validation to succeed for all valid URLs")
		})
	}
}

// TestValidateClient_InvalidHostnames tests that clients with any invalid URL fail validation.
func TestValidateClient_InvalidHostnames(t *testing.T) {
	log := logrus.New().WithField("test", "invalid_hostnames")
	validator := NewValidator(log, 3*time.Second, 15)
	ctx := context.Background()

	tests := []struct {
		name   string
		client ClientInfo
	}{
		{
			name: "Consensus client with invalid SSH hostname",
			client: ClientInfo{
				ClientName: "test-lighthouse-1",
				ClientType: "lighthouse",
				SSH:        "devops@invalid-hostname-that-does-not-exist-12345.com",
				BeaconAPI:  "cloudflare.com",
			},
		},
		{
			name: "Consensus client with invalid BeaconAPI hostname",
			client: ClientInfo{
				ClientName: "test-lighthouse-2",
				ClientType: "lighthouse",
				SSH:        "devops@google.com",
				BeaconAPI:  "invalid-beacon-api-hostname-xyz-999.com",
			},
		},
		{
			name: "Execution client with invalid SSH hostname",
			client: ClientInfo{
				ClientName: "test-geth-1",
				ClientType: "geth",
				SSH:        "devops@this-domain-definitely-does-not-exist-abc123.com",
				RPC:        "cloudflare.com",
			},
		},
		{
			name: "Execution client with invalid RPC hostname",
			client: ClientInfo{
				ClientName: "test-geth-2",
				ClientType: "geth",
				SSH:        "devops@google.com",
				RPC:        "invalid-rpc-endpoint-hostname-test-456.com",
			},
		},
		{
			name: "Client with all invalid hostnames",
			client: ClientInfo{
				ClientName: "test-bad-client",
				ClientType: "test",
				SSH:        "devops@bad-ssh-host-xyz-111.com",
				BeaconAPI:  "bad-beacon-host-xyz-222.com",
				RPC:        "bad-rpc-host-xyz-333.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateClient(ctx, tt.client)
			require.Error(t, err, "Expected validation to fail for client with invalid URL(s)")
		})
	}
}

// TestValidateClient_EmptyURLs tests that empty URLs are skipped during validation.
func TestValidateClient_EmptyURLs(t *testing.T) {
	log := logrus.New().WithField("test", "empty_urls")
	validator := NewValidator(log, 3*time.Second, 15)
	ctx := context.Background()

	tests := []struct {
		name   string
		client ClientInfo
	}{
		{
			name: "Client with empty SSH",
			client: ClientInfo{
				ClientName: "test-client-1",
				ClientType: "lighthouse",
				SSH:        "",
				BeaconAPI:  "cloudflare.com",
			},
		},
		{
			name: "Client with empty BeaconAPI",
			client: ClientInfo{
				ClientName: "test-client-2",
				ClientType: "lighthouse",
				SSH:        "devops@google.com",
				BeaconAPI:  "",
			},
		},
		{
			name: "Client with empty RPC",
			client: ClientInfo{
				ClientName: "test-client-3",
				ClientType: "geth",
				SSH:        "devops@google.com",
				RPC:        "",
			},
		},
		{
			name: "Client with all empty URLs",
			client: ClientInfo{
				ClientName: "test-client-4",
				ClientType: "test",
				SSH:        "",
				BeaconAPI:  "",
				RPC:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateClient(ctx, tt.client)
			assert.NoError(t, err, "Empty URLs should not cause validation to fail")
		})
	}
}

// TestNewValidator tests validator constructor with various parameters.
func TestNewValidator(t *testing.T) {
	log := logrus.New().WithField("test", "new_validator")

	tests := []struct {
		name          string
		dnsTimeout    time.Duration
		maxConcurrent int64
		expectNotNil  bool
	}{
		{
			name:          "Standard configuration",
			dnsTimeout:    3 * time.Second,
			maxConcurrent: 15,
			expectNotNil:  true,
		},
		{
			name:          "Low concurrency",
			dnsTimeout:    5 * time.Second,
			maxConcurrent: 1,
			expectNotNil:  true,
		},
		{
			name:          "High concurrency",
			dnsTimeout:    1 * time.Second,
			maxConcurrent: 100,
			expectNotNil:  true,
		},
		{
			name:          "Very short timeout",
			dnsTimeout:    100 * time.Millisecond,
			maxConcurrent: 10,
			expectNotNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(log, tt.dnsTimeout, tt.maxConcurrent)
			if tt.expectNotNil {
				assert.NotNil(t, validator, "Validator should not be nil")
				assert.NotNil(t, validator.log, "Logger should not be nil")
				assert.NotNil(t, validator.sem, "Semaphore should not be nil")
				assert.Equal(t, tt.dnsTimeout, validator.dnsTimeout, "DNS timeout should match")
			}
		})
	}
}

// TestValidateHostname_DirectCall tests the validateHostname method directly.
func TestValidateHostname_DirectCall(t *testing.T) {
	log := logrus.New().WithField("test", "validate_hostname_direct")
	validator := NewValidator(log, 3*time.Second, 15)
	ctx := context.Background()

	tests := []struct {
		name        string
		urlType     string
		url         string
		shouldError bool
	}{
		{
			name:        "Valid SSH URL",
			urlType:     "ssh",
			url:         "devops@google.com",
			shouldError: false,
		},
		{
			name:        "Valid BeaconAPI URL",
			urlType:     "beacon-api",
			url:         "cloudflare.com",
			shouldError: false,
		},
		{
			name:        "Valid RPC URL",
			urlType:     "rpc",
			url:         "github.com",
			shouldError: false,
		},
		{
			name:        "Invalid SSH URL",
			urlType:     "ssh",
			url:         "devops@this-host-does-not-exist-xyz-123.com",
			shouldError: true,
		},
		{
			name:        "Invalid BeaconAPI URL",
			urlType:     "beacon-api",
			url:         "invalid-beacon-host-xyz-456.com",
			shouldError: true,
		},
		{
			name:        "Invalid RPC URL",
			urlType:     "rpc",
			url:         "invalid-rpc-host-xyz-789.com",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateHostname(ctx, tt.urlType, tt.url)
			if tt.shouldError {
				require.Error(t, err, "Expected validation to fail")
				assert.Contains(t, err.Error(), tt.urlType, "Error should contain URL type")
			} else {
				assert.NoError(t, err, "Expected validation to succeed")
			}
		})
	}
}
