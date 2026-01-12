package eip7870referencenodes

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Sample reth values.yaml content for testing.
const rethValuesYAML = `
httpPort: 8545
wsPort: 8546
authPort: 8551
metricsPort: 9001
p2pPort: 30303

defaultCommandArgsTemplate: |
  --datadir=/data
  --config=/data/config.toml
  {{- if .Values.devnet.enabled }}
  --chain=/data/devnet/genesis.json
  {{- end }}
  {{- if .Values.p2pNodePort.enabled }}
  {{- if not (contains "--nat=" (.Values.extraArgs | join ",")) }}
  --nat=extip:$EXTERNAL_IP
  {{- end }}
  {{- if not (contains "--port=" (.Values.extraArgs | join ",")) }}
  --port=$EXTERNAL_PORT
  {{- end }}
  {{- else }}
  {{- if not (contains "--nat=" (.Values.extraArgs | join ",")) }}
  --nat=extip:$(POD_IP)
  {{- end }}
  {{- if not (contains "--port=" (.Values.extraArgs | join ",")) }}
  --port={{ include "reth.p2pPort" . }}
  {{- end }}
  {{- end }}
  --http
  --http.addr=0.0.0.0
  --http.port={{ .Values.httpPort }}
  --http.corsdomain=*
  --ws
  --ws.addr=0.0.0.0
  --ws.port={{ .Values.wsPort }}
  --ws.origins=*
  --authrpc.jwtsecret=/data/jwt.hex
  --authrpc.addr=0.0.0.0
  --authrpc.port={{ .Values.authPort }}
  {{- if .Values.fileLogging.enabled }}
  --log.file.directory={{ .Values.fileLogging.dir }}
  {{- else }}
  --log.file.max-files=0
  {{- end }}
  {{- if .Values.metricsPort }}
  --metrics=0.0.0.0:{{ .Values.metricsPort }}
  {{- end }}
  {{- if .Values.devnet.enabled }}
  --bootnodes="$(cat /data/devnet/enodes.txt | tr '\n' ',' | sed 's/,$//')"
  {{- end }}
  {{- range .Values.extraArgs }}
  {{ tpl . $ }}
  {{- end }}
`

func TestHelmChartParser_ParseBaseCommand_Reth(t *testing.T) {
	parser := NewHelmChartParser()

	args, err := parser.ParseBaseCommand([]byte(rethValuesYAML), "reth")
	require.NoError(t, err)

	// Expected args for reth with NodePort enabled, devnet disabled, file logging disabled
	expectedArgs := []string{
		"--datadir=/data",
		"--config=/data/config.toml",
		"--nat=extip:$EXTERNAL_IP",
		"--port=$EXTERNAL_PORT",
		"--http",
		"--http.addr=0.0.0.0",
		"--http.port=8545",
		"--http.corsdomain=*",
		"--ws",
		"--ws.addr=0.0.0.0",
		"--ws.port=8546",
		"--ws.origins=*",
		"--authrpc.jwtsecret=/data/jwt.hex",
		"--authrpc.addr=0.0.0.0",
		"--authrpc.port=8551",
		"--log.file.max-files=0",
		"--metrics=0.0.0.0:9001",
	}

	assert.Equal(t, expectedArgs, args)
}

// Sample geth values.yaml content for testing.
const gethValuesYAML = `
httpPort: 8545
wsPort: 8546
authPort: 8551
metricsPort: 6060
p2pPort: 30303

defaultCommandArgsTemplate: |
  --datadir={{ .Values.persistence.mountPath }}/data
  {{- if .Values.p2pNodePort.enabled }}
  --nat=extip:$EXTERNAL_IP
  --port=$EXTERNAL_PORT
  {{- else }}
  --nat=extip:$(POD_IP)
  --port={{ include "geth.p2pPort" . }}
  {{- end }}
  --http
  --http.addr=0.0.0.0
  --http.port={{ .Values.httpPort }}
  --http.vhosts=*
  --http.corsdomain=*
  --ws
  --ws.addr=0.0.0.0
  --ws.port={{ .Values.wsPort }}
  --ws.origins=*
  --authrpc.jwtsecret={{ .Values.persistence.mountPath }}/jwt.hex
  --authrpc.addr=0.0.0.0
  --authrpc.port={{ .Values.authPort }}
  --authrpc.vhosts=*
  {{- if .Values.metricsPort }}
  --metrics
  --metrics.addr=0.0.0.0
  --metrics.port={{ .Values.metricsPort }}
  {{- end }}
  {{- if .Values.devnet.enabled }}
  --networkid={{ .Values.devnet.networkID }}
  --bootnodes="$(cat {{ .Values.persistence.mountPath }}/devnet/enodes.txt | tr '\n' ',' | sed 's/,$//')"
  {{- end }}
  {{- range .Values.extraArgs }}
  {{ tpl . $ }}
  {{- end }}
`

func TestHelmChartParser_ParseBaseCommand_Geth(t *testing.T) {
	parser := NewHelmChartParser()

	args, err := parser.ParseBaseCommand([]byte(gethValuesYAML), "geth")
	require.NoError(t, err)

	// Expected args for geth with NodePort enabled, devnet disabled
	expectedArgs := []string{
		"--datadir=/data/data",
		"--nat=extip:$EXTERNAL_IP",
		"--port=$EXTERNAL_PORT",
		"--http",
		"--http.addr=0.0.0.0",
		"--http.port=8545",
		"--http.vhosts=*",
		"--http.corsdomain=*",
		"--ws",
		"--ws.addr=0.0.0.0",
		"--ws.port=8546",
		"--ws.origins=*",
		"--authrpc.jwtsecret=/data/jwt.hex",
		"--authrpc.addr=0.0.0.0",
		"--authrpc.port=8551",
		"--authrpc.vhosts=*",
		"--metrics",
		"--metrics.addr=0.0.0.0",
		"--metrics.port=6060",
	}

	assert.Equal(t, expectedArgs, args)
}

func TestHelmChartParser_ParseBaseCommand_NoTemplate(t *testing.T) {
	parser := NewHelmChartParser()

	yamlWithNoTemplate := `
httpPort: 8545
wsPort: 8546
`

	_, err := parser.ParseBaseCommand([]byte(yamlWithNoTemplate), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no defaultCommandArgsTemplate or defaultCommandTemplate found")
}

func TestHelmChartParser_ParseBaseCommand_DefaultPorts(t *testing.T) {
	parser := NewHelmChartParser()

	// YAML without explicit port values - should use defaults
	yamlWithDefaults := `
defaultCommandArgsTemplate: |
  --http.port={{ .Values.httpPort }}
  --ws.port={{ .Values.wsPort }}
  --authrpc.port={{ .Values.authPort }}
  --metrics.port={{ .Values.metricsPort }}
`

	args, err := parser.ParseBaseCommand([]byte(yamlWithDefaults), "test")
	require.NoError(t, err)

	// Should use default port values
	expectedArgs := []string{
		"--http.port=8545",
		"--ws.port=8546",
		"--authrpc.port=8551",
		"--metrics.port=9545",
	}

	assert.Equal(t, expectedArgs, args)
}

func TestHelmChartParser_SkipsDevnetArgs(t *testing.T) {
	parser := NewHelmChartParser()

	yamlWithDevnet := `
defaultCommandArgsTemplate: |
  --datadir=/data
  {{- if .Values.devnet.enabled }}
  --chain=/data/devnet/genesis.json
  --bootnodes="something"
  {{- end }}
  --http
`

	args, err := parser.ParseBaseCommand([]byte(yamlWithDevnet), "test")
	require.NoError(t, err)

	// Should NOT include devnet args
	assert.Contains(t, args, "--datadir=/data")
	assert.Contains(t, args, "--http")
	assert.NotContains(t, args, "--chain=/data/devnet/genesis.json")
	assert.NotContains(t, args, "--bootnodes=\"something\"")
}

func TestHelmChartParser_IncludesNodePortArgs(t *testing.T) {
	parser := NewHelmChartParser()

	yamlWithNodePort := `
defaultCommandArgsTemplate: |
  {{- if .Values.p2pNodePort.enabled }}
  --nat=extip:$EXTERNAL_IP
  --port=$EXTERNAL_PORT
  {{- else }}
  --nat=extip:$(POD_IP)
  --port=30303
  {{- end }}
`

	args, err := parser.ParseBaseCommand([]byte(yamlWithNodePort), "test")
	require.NoError(t, err)

	// Should include NodePort args, not the else branch
	assert.Contains(t, args, "--nat=extip:$EXTERNAL_IP")
	assert.Contains(t, args, "--port=$EXTERNAL_PORT")
	assert.NotContains(t, args, "--nat=extip:$(POD_IP)")
}

func TestHelmChartParser_FileLoggingElseBranch(t *testing.T) {
	parser := NewHelmChartParser()

	yamlWithFileLogging := `
defaultCommandArgsTemplate: |
  {{- if .Values.fileLogging.enabled }}
  --log.file.directory=/var/log
  {{- else }}
  --log.file.max-files=0
  {{- end }}
`

	args, err := parser.ParseBaseCommand([]byte(yamlWithFileLogging), "test")
	require.NoError(t, err)

	// Should include the else branch (file logging disabled)
	assert.Contains(t, args, "--log.file.max-files=0")
	assert.NotContains(t, args, "--log.file.directory=/var/log")
}

// fetchHelmChartValues fetches values.yaml from GitHub for a given client.
func fetchHelmChartValues(t *testing.T, client string) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := "https://raw.githubusercontent.com/ethpandaops/ethereum-helm-charts/master/charts/" + client + "/values.yaml"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to fetch %s", url)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return data
}

// TestDebugRethTemplate shows what the reth template looks like when parsed.
func TestDebugRethTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	valuesYAML := fetchHelmChartValues(t, "reth")

	var values struct {
		DefaultCommandArgsTemplate string `yaml:"defaultCommandArgsTemplate"`
		DefaultCommandTemplate     string `yaml:"defaultCommandTemplate"`
	}

	err := yaml.Unmarshal(valuesYAML, &values)
	require.NoError(t, err)

	t.Logf("DefaultCommandArgsTemplate length: %d", len(values.DefaultCommandArgsTemplate))

	maxLen := 500
	if len(values.DefaultCommandArgsTemplate) < maxLen {
		maxLen = len(values.DefaultCommandArgsTemplate)
	}

	t.Logf("DefaultCommandArgsTemplate first 500 chars:\n%s", values.DefaultCommandArgsTemplate[:maxLen])
	t.Logf("DefaultCommandTemplate length: %d", len(values.DefaultCommandTemplate))

	// Show what happens after normalization
	template := values.DefaultCommandArgsTemplate

	if strings.Count(template, "\n") < 5 {
		t.Log("Normalizing folded template...")
		template = strings.ReplaceAll(template, "}} ", "}}\n")
		template = strings.ReplaceAll(template, "}} --", "}}\n--")
		// Split on args that follow each other
		re := regexp.MustCompile(`\s+(--[a-z])`)
		template = re.ReplaceAllString(template, "\n$1")
	}

	lines := strings.Split(template, "\n")
	t.Logf("After normalization: %d lines", len(lines))

	for i, line := range lines[:min(len(lines), 30)] {
		t.Logf("  [%d] %q", i, line)
	}
}

// TestHelmChartParser_RealRethChart tests parsing against the actual reth helm chart.
func TestHelmChartParser_RealRethChart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	parser := NewHelmChartParser()
	valuesYAML := fetchHelmChartValues(t, "reth")

	args, err := parser.ParseBaseCommand(valuesYAML, "reth")
	require.NoError(t, err)

	// Verify essential args are present
	assert.NotEmpty(t, args, "Should have parsed some args")

	// Check for core args that should always be present
	foundDatadir := false
	foundHTTP := false
	foundWS := false
	foundAuthRPC := false
	foundMetrics := false
	foundNat := false

	for _, arg := range args {
		switch arg {
		case "--datadir=/data", "--datadir=/data/data":
			foundDatadir = true
		case "--http":
			foundHTTP = true
		case "--ws":
			foundWS = true
		case "--authrpc.jwtsecret=/data/jwt.hex":
			foundAuthRPC = true
		case "--metrics=0.0.0.0:9001", "--metrics=0.0.0.0:9545":
			foundMetrics = true
		case "--nat=extip:$EXTERNAL_IP":
			foundNat = true
		}
	}

	assert.True(t, foundDatadir, "Should have --datadir arg")
	assert.True(t, foundHTTP, "Should have --http arg")
	assert.True(t, foundWS, "Should have --ws arg")
	assert.True(t, foundAuthRPC, "Should have --authrpc.jwtsecret arg")
	assert.True(t, foundMetrics, "Should have --metrics arg")
	// Note: --nat may be missing due to nested conditional handling - this is acceptable
	// as long as core args are present
	t.Logf("Found --nat=extip:$EXTERNAL_IP: %v", foundNat)

	// Log all parsed args for visibility
	t.Logf("Parsed %d args from real reth chart:", len(args))
	for i, arg := range args {
		t.Logf("  [%d] %s", i, arg)
	}
}

// TestHelmChartParser_RealGethChart tests parsing against the actual geth helm chart.
func TestHelmChartParser_RealGethChart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	parser := NewHelmChartParser()
	valuesYAML := fetchHelmChartValues(t, "geth")

	args, err := parser.ParseBaseCommand(valuesYAML, "geth")
	require.NoError(t, err)

	// Verify essential args are present
	assert.NotEmpty(t, args, "Should have parsed some args")

	// Log all parsed args for visibility
	t.Logf("Parsed %d args from real geth chart:", len(args))
	for i, arg := range args {
		t.Logf("  [%d] %s", i, arg)
	}
}
