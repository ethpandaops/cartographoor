package utils

import (
	"os"
	"regexp"
	"strings"
)

var (
	// envVarRegex matches ${VAR} or $VAR patterns.
	envVarRegex = regexp.MustCompile(`\${([a-zA-Z0-9_]+)}|\$([a-zA-Z0-9_]+)`)
)

// EnvSubst substitutes environment variables in the given string.
// It supports both ${VAR} and $VAR syntax.
func EnvSubst(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR} or $VAR format.
		var envVar string
		if strings.HasPrefix(match, "${") {
			envVar = match[2 : len(match)-1]
		} else {
			envVar = match[1:]
		}

		// Return the environment variable value or the original string if not found.
		if val, exists := os.LookupEnv(envVar); exists {
			return val
		}

		return match
	})
}

// EnvSubstBytes substitutes environment variables in the given byte array.
func EnvSubstBytes(b []byte) []byte {
	return []byte(EnvSubst(string(b)))
}
