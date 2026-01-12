package eip7870referencenodes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandBuilder_DeduplicateArgs(t *testing.T) {
	builder := NewCommandBuilder()

	tests := []struct {
		name           string
		base           []string
		clientSpecific []string
		feature        []string
		expectedBase   []string
		expectedClient []string
		expectedFeat   []string
	}{
		{
			name: "clientSpecific overrides base flag with different value",
			base: []string{
				"--http",
				"--http.addr=0.0.0.0",
				"--http.port=8545",
			},
			clientSpecific: []string{
				"--http=false",
			},
			feature:        []string{},
			expectedBase:   []string{"--http.addr=0.0.0.0", "--http.port=8545"},
			expectedClient: []string{"--http=false"},
			expectedFeat:   []string{},
		},
		{
			name: "removes duplicates within clientSpecific",
			base: []string{
				"--datadir=/data",
			},
			clientSpecific: []string{
				"--externalcl",
				"--http.vhosts=*",
				"--externalcl", // duplicate
			},
			feature:        []string{},
			expectedBase:   []string{"--datadir=/data"},
			expectedClient: []string{"--externalcl", "--http.vhosts=*"},
			expectedFeat:   []string{},
		},
		{
			name: "clientSpecific overrides base with same flag name",
			base: []string{
				"--http.api=eth,web3",
				"--http.vhosts=localhost",
			},
			clientSpecific: []string{
				"--http.api=admin,debug,eth",
				"--http.vhosts=*",
			},
			feature:        []string{},
			expectedBase:   []string{},
			expectedClient: []string{"--http.api=admin,debug,eth", "--http.vhosts=*"},
			expectedFeat:   []string{},
		},
		{
			name: "feature overrides both base and clientSpecific",
			base: []string{
				"--metrics.port=9545",
			},
			clientSpecific: []string{
				"--metrics.port=6060",
			},
			feature: []string{
				"--metrics.port=9001",
			},
			expectedBase:   []string{},
			expectedClient: []string{},
			expectedFeat:   []string{"--metrics.port=9001"},
		},
		{
			name: "erigon-like scenario",
			base: []string{
				"--datadir=/data",
				"--http",
				"--http.addr=0.0.0.0",
				"--http.port=8545",
				"--http.vhosts=*",
				"--http.api=eth,erigon,web3",
				"--externalcl",
			},
			clientSpecific: []string{
				"--externalcl",
				"--http.vhosts=*",
				"--http.api=admin,debug,eth,net",
			},
			feature:        []string{},
			expectedBase:   []string{"--datadir=/data", "--http", "--http.addr=0.0.0.0", "--http.port=8545"},
			expectedClient: []string{"--externalcl", "--http.vhosts=*", "--http.api=admin,debug,eth,net"},
			expectedFeat:   []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.deduplicateArgs(tc.base, tc.clientSpecific, tc.feature)
			assert.Equal(t, tc.expectedBase, result.base, "base args mismatch")
			assert.Equal(t, tc.expectedClient, result.clientSpecific, "clientSpecific args mismatch")
			assert.Equal(t, tc.expectedFeat, result.feature, "feature args mismatch")
		})
	}
}

func TestExtractFlagName(t *testing.T) {
	tests := []struct {
		arg      string
		expected string
	}{
		{"--http", "--http"},
		{"--http=false", "--http"},
		{"--http.port=8545", "--http.port"},
		{"--http.api=eth,web3,debug", "--http.api"},
		{"--datadir=/data", "--datadir"},
	}

	for _, tc := range tests {
		t.Run(tc.arg, func(t *testing.T) {
			result := extractFlagName(tc.arg)
			assert.Equal(t, tc.expected, result)
		})
	}
}
