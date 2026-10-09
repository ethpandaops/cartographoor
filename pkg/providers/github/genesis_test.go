package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractBlobSchedule(t *testing.T) {
	// Test timing parameters
	testTiming := chainTiming{
		genesisTime:         1000,
		slotsPerEpoch:       32,
		slotDurationSeconds: 12,
	}

	tests := []struct {
		name        string
		configData  map[string]any
		expected    []discovery.BlobSchedule
		expectEmpty bool
	}{
		{
			name: "valid blob schedule with two entries",
			configData: map[string]any{
				"BLOB_SCHEDULE": []any{
					map[string]any{
						"EPOCH":               412672,
						"MAX_BLOBS_PER_BLOCK": 15,
					},
					map[string]any{
						"EPOCH":               419072,
						"MAX_BLOBS_PER_BLOCK": 21,
					},
				},
			},
			expected: []discovery.BlobSchedule{
				{Epoch: 412672, Timestamp: testTiming.genesisTime + (412672 * testTiming.slotsPerEpoch * testTiming.slotDurationSeconds), MaxBlobsPerBlock: 15},
				{Epoch: 419072, Timestamp: testTiming.genesisTime + (419072 * testTiming.slotsPerEpoch * testTiming.slotDurationSeconds), MaxBlobsPerBlock: 21},
			},
			expectEmpty: false,
		},
		{
			name: "valid blob schedule with string values",
			configData: map[string]any{
				"BLOB_SCHEDULE": []any{
					map[string]any{
						"EPOCH":               "412672",
						"MAX_BLOBS_PER_BLOCK": "15",
					},
				},
			},
			expected: []discovery.BlobSchedule{
				{Epoch: 412672, Timestamp: testTiming.genesisTime + (412672 * testTiming.slotsPerEpoch * testTiming.slotDurationSeconds), MaxBlobsPerBlock: 15},
			},
			expectEmpty: false,
		},
		{
			name:        "no blob schedule",
			configData:  map[string]any{},
			expected:    nil,
			expectEmpty: true,
		},
		{
			name: "blob schedule with missing epoch",
			configData: map[string]any{
				"BLOB_SCHEDULE": []any{
					map[string]any{
						"MAX_BLOBS_PER_BLOCK": 15,
					},
				},
			},
			expected:    nil,
			expectEmpty: true,
		},
		{
			name: "blob schedule with missing max blobs",
			configData: map[string]any{
				"BLOB_SCHEDULE": []any{
					map[string]any{
						"EPOCH": 412672,
					},
				},
			},
			expected:    nil,
			expectEmpty: true,
		},
		{
			name: "blob schedule with invalid type",
			configData: map[string]any{
				"BLOB_SCHEDULE": "not an array",
			},
			expected:    nil,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a provider with a logger
			log := logrus.New()
			log.SetLevel(logrus.DebugLevel)

			p := &Provider{
				log: log,
			}

			result := p.extractBlobSchedule(tt.configData, "test-network", testTiming)

			if tt.expectEmpty {
				assert.Nil(t, result, "Expected nil blob schedule")
			} else {
				assert.NotNil(t, result, "Expected non-nil blob schedule")
				assert.Equal(t, len(tt.expected), len(result), "Unexpected number of blob schedule entries")

				for i, expected := range tt.expected {
					assert.Equal(t, expected.Epoch, result[i].Epoch, "Epoch mismatch at index %d", i)
					assert.Equal(t, expected.Timestamp, result[i].Timestamp, "Timestamp mismatch at index %d", i)
					assert.Equal(t, expected.MaxBlobsPerBlock, result[i].MaxBlobsPerBlock, "MaxBlobsPerBlock mismatch at index %d", i)
				}
			}
		})
	}
}

func TestParseConfigYAML_TimestampsIncludeGenesisDelay(t *testing.T) {
	const configYAML = `
MIN_GENESIS_TIME: 1000
GENESIS_DELAY: 60
TEST_FORK_EPOCH: 1
SLOTS_PER_EPOCH: 32
SLOT_DURATION_MS: 12000
`

	mux := http.NewServeMux()
	mux.HandleFunc("/ethpandaops/example-devnets/contents/network-configs/devnet-1/metadata/config.yaml",
		func(w http.ResponseWriter, r *http.Request) {
			content := base64.StdEncoding.EncodeToString([]byte(configYAML))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"type":"file","encoding":"base64","content":%q}`, content)
		})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	log := logrus.New()

	provider, err := NewProvider(log, nil)
	require.NoError(t, err)

	provider.githubClient = gh.NewClient(&http.Client{Transport: &mockTransport{URL: srv.URL}})

	_, genesisTime, genesisDelay, forks, _, err := provider.parseConfigYAML(
		context.Background(), "ethpandaops", "example-devnets", "devnet-1")
	require.NoError(t, err)

	// The raw values returned for GenesisConfig must stay separate and unchanged.
	assert.Equal(t, uint64(1000), genesisTime)
	assert.Equal(t, uint64(60), genesisDelay)

	require.NotNil(t, forks)

	fork, ok := forks.Consensus["test"]
	require.True(t, ok, "expected a 'test' fork entry")

	// The derived timestamp must anchor on MIN_GENESIS_TIME + GENESIS_DELAY,
	// not MIN_GENESIS_TIME alone.
	wantTimestamp := uint64(1000) + uint64(60) + uint64(1)*32*12
	assert.Equal(t, wantTimestamp, fork.Timestamp)
}
