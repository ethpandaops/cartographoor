package eip7870referencenodes

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// HelmChartParser parses Helm chart values.yaml files to extract base commands.
type HelmChartParser struct {
	portOverrides PortOverrides
}

// NewHelmChartParser creates a new HelmChartParser with optional port overrides.
func NewHelmChartParser(portOverrides PortOverrides) *HelmChartParser {
	return &HelmChartParser{
		portOverrides: portOverrides,
	}
}

// resolvePort returns the first non-zero value from: override, helmValue, defaultValue.
func (p *HelmChartParser) resolvePort(override, helmValue, defaultValue int) int {
	if override != 0 {
		return override
	}

	if helmValue != 0 {
		return helmValue
	}

	return defaultValue
}

// ParseBaseCommand parses a Helm chart values.yaml and extracts the base command arguments.
// It handles two formats:
// 1. defaultCommandArgsTemplate - separate args template (reth, besu, nethermind).
// 2. defaultCommandTemplate - inline args in command template (geth, erigon).
func (p *HelmChartParser) ParseBaseCommand(valuesYAML []byte, client string) ([]string, error) {
	var values HelmChartValues
	if err := yaml.Unmarshal(valuesYAML, &values); err != nil {
		return nil, fmt.Errorf("failed to parse helm chart values: %w", err)
	}

	// Determine which template to use
	template := values.DefaultCommandArgsTemplate
	if template == "" {
		// Fall back to extracting args from DefaultCommandTemplate
		template = values.DefaultCommandTemplate
		if template == "" {
			return nil, fmt.Errorf("no defaultCommandArgsTemplate or defaultCommandTemplate found in values.yaml")
		}
	}

	// Apply port overrides from config, falling back to helm chart values, then defaults
	values.HTTPPort = p.resolvePort(p.portOverrides.HTTPPort, values.HTTPPort, 8545)
	values.WSPort = p.resolvePort(p.portOverrides.WSPort, values.WSPort, 8546)
	values.AuthPort = p.resolvePort(p.portOverrides.AuthPort, values.AuthPort, 8551)
	values.MetricsPort = p.resolvePort(p.portOverrides.MetricsPort, values.MetricsPort, 9545)
	values.P2PPort = p.resolvePort(p.portOverrides.P2PPort, values.P2PPort, 30303)

	// Render the template to extract arguments
	args := p.renderTemplate(template, values, client)

	return args, nil
}

// blockType represents the type of conditional block we're tracking.
type blockType int

const (
	blockNone blockType = iota
	blockDevnet
	blockNodePort
	blockFileLogging
	blockMetrics
	blockExtraArgs
	blockOther
)

// classifyBlock determines the block type and whether to include it.
func classifyBlock(line string) (blockType, bool) {
	switch {
	case strings.Contains(line, "devnet.enabled"):
		return blockDevnet, false // We don't want devnet args
	case strings.Contains(line, "p2pNodePort.enabled"):
		return blockNodePort, true // We want NodePort args
	case strings.Contains(line, "fileLogging.enabled"):
		return blockFileLogging, false // We don't want file logging, we want the else branch
	case strings.Contains(line, "metricsPort"):
		return blockMetrics, true // We want metrics
	case strings.Contains(line, "if not (contains"):
		// This is a guard condition like "if extraArgs doesn't already contain X"
		// We want to include these args since we're generating a base command
		return blockOther, true
	default:
		return blockOther, true // For other conditionals, default to include
	}
}

// blockStack manages nested conditional block state during template parsing.
type blockStack struct {
	stack []struct {
		blockType     blockType
		shouldInclude bool
	}
}

func newBlockStack() *blockStack {
	return &blockStack{
		stack: make([]struct {
			blockType     blockType
			shouldInclude bool
		}, 0, 8),
	}
}

func (bs *blockStack) push(bt blockType, include bool) {
	bs.stack = append(bs.stack, struct {
		blockType     blockType
		shouldInclude bool
	}{bt, include})
}

func (bs *blockStack) pop() {
	if len(bs.stack) > 0 {
		bs.stack = bs.stack[:len(bs.stack)-1]
	}
}

func (bs *blockStack) popN(n int) {
	for i := 0; i < n && len(bs.stack) > 0; i++ {
		bs.stack = bs.stack[:len(bs.stack)-1]
	}
}

func (bs *blockStack) shouldInclude() bool {
	for _, state := range bs.stack {
		if !state.shouldInclude {
			return false
		}
	}

	return true
}

func (bs *blockStack) handleElse() {
	if len(bs.stack) == 0 {
		return
	}

	lastIdx := len(bs.stack) - 1

	switch bs.stack[lastIdx].blockType {
	case blockFileLogging:
		bs.stack[lastIdx].shouldInclude = true
	case blockNodePort:
		bs.stack[lastIdx].shouldInclude = false
	default:
		bs.stack[lastIdx].shouldInclude = !bs.stack[lastIdx].shouldInclude
	}
}

// normalizeTemplate converts folded block scalar format to newline-separated lines.
func normalizeTemplate(template string) string {
	if strings.Contains(template, "\n") && strings.Count(template, "\n") >= 5 {
		return template
	}

	// This is likely a folded template - split on }} markers to get segments
	template = strings.ReplaceAll(template, "}} ", "}}\n")
	template = strings.ReplaceAll(template, "}} --", "}}\n--")
	// Also handle args that follow each other without template markers
	template = regexp.MustCompile(`\s+(--[a-z])`).ReplaceAllString(template, "\n$1")

	return template
}

// countEndMarkers counts {{- end }} markers in a line.
func countEndMarkers(line string) int {
	return strings.Count(line, "end }}") + strings.Count(line, "end}}")
}

// renderTemplate renders the defaultCommandArgsTemplate and extracts arguments.
// It handles conditional blocks with these assumptions for reference nodes:
// - NodePort is enabled (use $EXTERNAL_IP and $EXTERNAL_PORT)
// - Devnet is disabled (skip devnet-specific args)
// - File logging is disabled (use --log.file.max-files=0)
// - Metrics are enabled (include metrics args).
func (p *HelmChartParser) renderTemplate(template string, values HelmChartValues, client string) []string {
	template = normalizeTemplate(template)
	lines := strings.Split(template, "\n")
	bs := newBlockStack()
	args := make([]string, 0, 32)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		isPureTemplate := strings.HasPrefix(line, "{{") && strings.HasSuffix(line, "}}")

		if isPureTemplate {
			p.processPureTemplateLine(line, bs)

			continue
		}

		if !bs.shouldInclude() {
			bs.popN(countEndMarkers(line))

			continue
		}

		if p.shouldSkipLine(line) {
			continue
		}

		// Check for embedded {{- if ... }} markers
		if strings.Contains(line, "{{") && strings.Contains(line, "if ") {
			bt, include := classifyBlock(line)
			bs.push(bt, include)
		}

		// Process the arg
		arg := p.substituteValues(line, values, client)
		if arg != "" && strings.HasPrefix(arg, "--") {
			args = append(args, arg)
		}

		// Process any end markers
		bs.popN(countEndMarkers(line))
	}

	return args
}

// processPureTemplateLine handles lines that are only template control structures.
func (p *HelmChartParser) processPureTemplateLine(line string, bs *blockStack) {
	switch {
	case strings.Contains(line, "if "):
		bt, include := classifyBlock(line)
		bs.push(bt, include)
	case strings.Contains(line, "else"):
		bs.handleElse()
	case strings.Contains(line, "end"):
		bs.pop()
	case strings.Contains(line, "range"):
		bs.push(blockExtraArgs, false)
	}
}

// shouldSkipLine returns true if the line should be skipped.
func (p *HelmChartParser) shouldSkipLine(line string) bool {
	// Skip lines where $(POD_IP) is used as a substitute for EXTERNAL_IP
	// (e.g., --nat=extip:$(POD_IP) in non-NodePort mode)
	// But keep lines where $(POD_IP) is used for a different purpose
	// (e.g., --Network.LocalIp=$(POD_IP) which is the local bind address)
	if strings.Contains(line, "$(POD_IP)") {
		// Skip if POD_IP is used in nat/extip context (alternative to EXTERNAL_IP)
		if strings.Contains(line, "nat=") || strings.Contains(line, "extip") {
			return true
		}
		// Keep other uses of POD_IP (like LocalIp, which is a different setting)
	}

	// Skip lines with include functions for P2P port (we use EXTERNAL_PORT)
	if strings.Contains(line, "include \"") && strings.Contains(line, "p2pPort") {
		return true
	}

	return false
}

// substituteValues replaces Helm template expressions with actual values.
func (p *HelmChartParser) substituteValues(line string, values HelmChartValues, client string) string {
	// First, strip trailing template markers like {{- end }} or {{- if ... }}
	// These can be attached to actual args
	line = regexp.MustCompile(`\s*\{\{-?\s*(end|if\s+[^}]+|else)\s*-?\}\}\s*$`).ReplaceAllString(line, "")
	line = regexp.MustCompile(`^\s*\{\{-?\s*(end|else)\s*-?\}\}\s*`).ReplaceAllString(line, "")

	// Skip lines that are ONLY template control structures
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
		// This is a pure template line
		return ""
	}

	// Skip comment lines
	if strings.Contains(trimmed, "{{/*") {
		return ""
	}

	// Direct string replacements for common patterns
	replacements := map[string]string{
		"{{ .Values.httpPort }}":               fmt.Sprintf("%d", values.HTTPPort),
		"{{ .Values.wsPort }}":                 fmt.Sprintf("%d", values.WSPort),
		"{{ .Values.authPort }}":               fmt.Sprintf("%d", values.AuthPort),
		"{{ .Values.metricsPort }}":            fmt.Sprintf("%d", values.MetricsPort),
		"{{ .Values.persistence.mountPath }}":  "/data",
		"{{ .Values.p2pNodePort.externalIP }}": "$EXTERNAL_IP",
		"{{ .Values.p2pNodePort.port }}":       "$EXTERNAL_PORT",
	}

	for pattern, value := range replacements {
		line = strings.ReplaceAll(line, pattern, value)
	}

	// Handle include function for p2p port
	p2pPortPattern := regexp.MustCompile(`\{\{\s*include\s+"[^"]+\.p2pPort"\s+\.\s*\}\}`)
	line = p2pPortPattern.ReplaceAllString(line, fmt.Sprintf("%d", values.P2PPort))

	// Clean up any remaining template syntax
	line = cleanTemplateResidue(line)

	return strings.TrimSpace(line)
}

// cleanTemplateResidue removes any remaining Helm template syntax.
func cleanTemplateResidue(line string) string {
	// Remove {{ }} and {{- -}} patterns that weren't substituted
	re := regexp.MustCompile(`\{\{-?\s*[^}]*\s*-?\}\}`)
	line = re.ReplaceAllString(line, "")

	// Clean up multiple spaces
	line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")

	return strings.TrimSpace(line)
}
