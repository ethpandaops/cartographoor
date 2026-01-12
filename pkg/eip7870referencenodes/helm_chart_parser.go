package eip7870referencenodes

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// HelmChartParser parses Helm chart values.yaml files to extract base commands.
type HelmChartParser struct{}

// NewHelmChartParser creates a new HelmChartParser.
func NewHelmChartParser() *HelmChartParser {
	return &HelmChartParser{}
}

// ParseBaseCommand parses a Helm chart values.yaml and extracts the base command arguments.
// It renders the defaultCommandTemplate with default values and returns individual arguments.
func (p *HelmChartParser) ParseBaseCommand(valuesYAML []byte, client string) ([]string, error) {
	var values HelmChartValues
	if err := yaml.Unmarshal(valuesYAML, &values); err != nil {
		return nil, fmt.Errorf("failed to parse helm chart values: %w", err)
	}

	if values.DefaultCommandTemplate == "" {
		return nil, fmt.Errorf("no defaultCommandTemplate found in values.yaml")
	}

	// Set defaults if not specified
	if values.HTTPPort == 0 {
		values.HTTPPort = 8545
	}

	if values.WSPort == 0 {
		values.WSPort = 8546
	}

	if values.AuthPort == 0 {
		values.AuthPort = 8551
	}

	if values.MetricsPort == 0 {
		values.MetricsPort = 9545
	}

	if values.P2PPort == 0 {
		values.P2PPort = 30303
	}

	// Render the template to extract arguments
	args := p.renderTemplate(values.DefaultCommandTemplate, values, client)

	return args, nil
}

// renderTemplate renders the defaultCommandTemplate and extracts arguments.
// This is a simplified renderer that handles the common patterns in helm charts.
func (p *HelmChartParser) renderTemplate(template string, values HelmChartValues, client string) []string {
	// Remove shell wrapper (- sh, -ac, etc.) and extract the actual command
	lines := strings.Split(template, "\n")

	var args []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip shell wrapper lines
		if line == "- sh" || line == "- -ac" || line == ">-" || line == ">" || line == "-ac" {
			continue
		}

		// Skip template control structures for devnet mode
		if strings.Contains(line, "devnet.enabled") {
			continue
		}

		if strings.Contains(line, "genesis-file") || strings.Contains(line, "bootnodes=") {
			// Skip devnet-specific args
			continue
		}

		// Skip lines that are just template logic
		if strings.HasPrefix(line, "{{-") && strings.HasSuffix(line, "}}") {
			continue
		}

		// Skip extraArgs range
		if strings.Contains(line, "range .Values.extraArgs") || strings.Contains(line, "tpl . $") {
			continue
		}

		// Skip else and end blocks
		if strings.Contains(line, "{{- else }}") || strings.Contains(line, "{{- end }}") {
			continue
		}

		// Skip the POD_IP path (we want EXTERNAL_IP for NodePort)
		if strings.Contains(line, "$(POD_IP)") {
			continue
		}

		// Remove exec prefix if present (start of actual command)
		line = strings.TrimPrefix(line, "exec ")

		// Check if this is just the client binary name
		if line == client || line == "exec "+client {
			continue
		}

		// Skip if line is just the include function for port
		if strings.Contains(line, "include \"") {
			continue
		}

		// Process the line - substitute template values
		arg := p.substituteValues(line, values, client)
		if arg != "" && strings.HasPrefix(arg, "--") {
			args = append(args, arg)
		}
	}

	return args
}

// substituteValues replaces Helm template expressions with actual values.
func (p *HelmChartParser) substituteValues(line string, values HelmChartValues, client string) string {
	// Remove any remaining template control flow markers
	line = strings.TrimPrefix(line, "{{- ")
	line = strings.TrimSuffix(line, " }}")

	// Skip lines that contain template conditionals
	if strings.Contains(line, "if ") || strings.Contains(line, "end") ||
		strings.Contains(line, "else") || strings.Contains(line, "range") {
		return ""
	}

	// Replace port values
	line = replaceTemplateVar(line, ".Values.httpPort", fmt.Sprintf("%d", values.HTTPPort))
	line = replaceTemplateVar(line, ".Values.wsPort", fmt.Sprintf("%d", values.WSPort))
	line = replaceTemplateVar(line, ".Values.authPort", fmt.Sprintf("%d", values.AuthPort))
	line = replaceTemplateVar(line, ".Values.metricsPort", fmt.Sprintf("%d", values.MetricsPort))

	// Replace p2p port - use include function pattern
	p2pPortPattern := regexp.MustCompile(`\{\{\s*include\s+"` + client + `\.p2pPort"\s+\.\s*\}\}`)
	line = p2pPortPattern.ReplaceAllString(line, fmt.Sprintf("%d", values.P2PPort))

	// Replace NodePort specific values with placeholder env vars
	line = replaceTemplateVar(line, ".Values.p2pNodePort.externalIP", "$EXTERNAL_IP")
	line = replaceTemplateVar(line, ".Values.p2pNodePort.port", "$EXTERNAL_PORT")

	// Replace persistence mount path
	line = replaceTemplateVar(line, ".Values.persistence.mountPath", "/data")

	// Clean up any remaining template syntax
	line = cleanTemplateResidue(line)

	return strings.TrimSpace(line)
}

// replaceTemplateVar replaces a Helm template variable pattern with a value.
func replaceTemplateVar(line, varPattern, value string) string {
	// Pattern: {{ .Values.something }} or {{.Values.something}}
	patterns := []string{
		`\{\{\s*` + regexp.QuoteMeta(varPattern) + `\s*\}\}`,
		`\{\{-?\s*` + regexp.QuoteMeta(varPattern) + `\s*-?\}\}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		line = re.ReplaceAllString(line, value)
	}

	return line
}

// cleanTemplateResidue removes any remaining Helm template syntax.
func cleanTemplateResidue(line string) string {
	// Remove {{ }} and {{- -}} patterns
	re := regexp.MustCompile(`\{\{-?\s*[^}]*\s*-?\}\}`)
	line = re.ReplaceAllString(line, "")

	// Clean up multiple spaces
	line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")

	return strings.TrimSpace(line)
}
