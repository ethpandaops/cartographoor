package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvSubst(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test-value")
	os.Setenv("ANOTHER_VAR", "another-value")
	os.Setenv("NUMERIC_VAR", "12345")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("ANOTHER_VAR")
		os.Unsetenv("NUMERIC_VAR")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No substitution",
			input:    "This is a test string with no variables",
			expected: "This is a test string with no variables",
		},
		{
			name:     "Curly braces syntax",
			input:    "This is a ${TEST_VAR}",
			expected: "This is a test-value",
		},
		{
			name:     "Simple variable syntax",
			input:    "This is a $TEST_VAR",
			expected: "This is a test-value",
		},
		{
			name:     "Multiple variables",
			input:    "First: ${TEST_VAR}, Second: $ANOTHER_VAR",
			expected: "First: test-value, Second: another-value",
		},
		{
			name:     "Variable in the middle",
			input:    "The value is ${TEST_VAR} in the middle",
			expected: "The value is test-value in the middle",
		},
		{
			name:     "Numeric variable",
			input:    "Number: $NUMERIC_VAR",
			expected: "Number: 12345",
		},
		{
			name:     "Non-existent variable",
			input:    "This uses ${NON_EXISTENT_VAR}",
			expected: "This uses ${NON_EXISTENT_VAR}",
		},
		{
			name:     "Mixed existing and non-existing variables",
			input:    "${TEST_VAR} and ${NON_EXISTENT_VAR}",
			expected: "test-value and ${NON_EXISTENT_VAR}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnvSubst(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvSubstBytes(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	input := []byte("This is ${TEST_VAR}")
	expected := []byte("This is test-value")

	result := EnvSubstBytes(input)
	assert.Equal(t, expected, result)
}
