package inventory

import "time"

// Config holds configuration for inventory generation.
type Config struct {
	Validation ValidationConfig
}

// ValidationConfig holds configuration for URL validation.
type ValidationConfig struct {
	// Enabled controls whether DNS validation is performed
	Enabled bool

	// DNSTimeout is the timeout for each DNS hostname lookup
	DNSTimeout time.Duration

	// MaxConcurrentValidations is the maximum number of concurrent DNS validations
	MaxConcurrentValidations int64
}
