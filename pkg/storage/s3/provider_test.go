package s3

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Skip all S3 tests for now as they require proper mocking of AWS SDK

func TestS3Provider_ConfigDefaults(t *testing.T) {
	log := logrus.New()

	// Test with minimal config
	cfg := Config{
		BucketName: "test-bucket",
	}

	provider, err := NewProvider(log, cfg)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "networks.json", provider.config.Key)
	assert.Equal(t, "application/json", provider.config.ContentType)
	assert.Equal(t, 5*time.Second, provider.config.RetryDuration)
	assert.Equal(t, 3, provider.config.MaxRetries)
	assert.Equal(t, 20, provider.config.BackoffJitterPercent)
}