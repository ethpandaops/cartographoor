package github

import (
	"testing"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestExtractBlobSchedule(t *testing.T) {
	tests := []struct {
		name        string
		configData  map[string]interface{}
		expected    []discovery.BlobSchedule
		expectEmpty bool
	}{
		{
			name: "valid blob schedule with two entries",
			configData: map[string]interface{}{
				"BLOB_SCHEDULE": []interface{}{
					map[string]interface{}{
						"EPOCH":               412672,
						"MAX_BLOBS_PER_BLOCK": 15,
					},
					map[string]interface{}{
						"EPOCH":               419072,
						"MAX_BLOBS_PER_BLOCK": 21,
					},
				},
			},
			expected: []discovery.BlobSchedule{
				{Epoch: 412672, MaxBlobsPerBlock: 15},
				{Epoch: 419072, MaxBlobsPerBlock: 21},
			},
			expectEmpty: false,
		},
		{
			name: "valid blob schedule with string values",
			configData: map[string]interface{}{
				"BLOB_SCHEDULE": []interface{}{
					map[string]interface{}{
						"EPOCH":               "412672",
						"MAX_BLOBS_PER_BLOCK": "15",
					},
				},
			},
			expected: []discovery.BlobSchedule{
				{Epoch: 412672, MaxBlobsPerBlock: 15},
			},
			expectEmpty: false,
		},
		{
			name:        "no blob schedule",
			configData:  map[string]interface{}{},
			expected:    nil,
			expectEmpty: true,
		},
		{
			name: "blob schedule with missing epoch",
			configData: map[string]interface{}{
				"BLOB_SCHEDULE": []interface{}{
					map[string]interface{}{
						"MAX_BLOBS_PER_BLOCK": 15,
					},
				},
			},
			expected:    nil,
			expectEmpty: true,
		},
		{
			name: "blob schedule with missing max blobs",
			configData: map[string]interface{}{
				"BLOB_SCHEDULE": []interface{}{
					map[string]interface{}{
						"EPOCH": 412672,
					},
				},
			},
			expected:    nil,
			expectEmpty: true,
		},
		{
			name: "blob schedule with invalid type",
			configData: map[string]interface{}{
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

			result := p.extractBlobSchedule(tt.configData, "test-network")

			if tt.expectEmpty {
				assert.Nil(t, result, "Expected nil blob schedule")
			} else {
				assert.NotNil(t, result, "Expected non-nil blob schedule")
				assert.Equal(t, len(tt.expected), len(result), "Unexpected number of blob schedule entries")

				for i, expected := range tt.expected {
					assert.Equal(t, expected.Epoch, result[i].Epoch, "Epoch mismatch at index %d", i)
					assert.Equal(t, expected.MaxBlobsPerBlock, result[i].MaxBlobsPerBlock, "MaxBlobsPerBlock mismatch at index %d", i)
				}
			}
		})
	}
}
