package github

import (
	"context"
	"strings"

	gh "github.com/google/go-github/v53/github"
)

const (
	networkConfigDir     = "network-configs"
	kubernetesDir        = "kubernetes"
	kubernetesArchiveDir = "kubernetes-archive"
	active               = "active"
	inactive             = "inactive"
	unknown              = "unknown"
)

// parseValuesYaml decodes the content of values.yaml and extracts config file paths and domain.
func (p *Provider) parseValuesYaml(
	ctx context.Context,
	fileContent *gh.RepositoryContent,
	networkName string,
) ([]string, string) {
	// Get content from GitHub response
	content, err := fileContent.GetContent()
	if err != nil {
		p.log.WithError(err).WithField("network", networkName).Error("Failed to decode values.yaml content")

		return nil, ""
	}

	// Extract domain
	domain := p.extractDomain(content)

	// Extract config file paths
	configPaths := p.extractConfigPaths(content)

	return configPaths, domain
}

// extractDomain extracts the domain from values.yaml content.
func (p *Provider) extractDomain(content string) string {
	var domain string

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "domain:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				domain = strings.TrimSpace(parts[1])

				break
			}
		}
	}

	return domain
}

// extractConfigPaths extracts the config file paths from values.yaml content.
func (p *Provider) extractConfigPaths(content string) []string {
	var (
		configPaths     []string
		inConfigSection = false
		inFilesSection  = false
	)

	for _, line := range strings.Split(content, "\n") {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the config section
		if strings.HasPrefix(trimmedLine, "config:") {
			inConfigSection = true

			continue
		}

		// Check if we're entering the files section inside config
		if inConfigSection && strings.HasPrefix(trimmedLine, "files:") {
			inFilesSection = true

			continue
		}

		// Process file paths inside the files section
		if inConfigSection && inFilesSection && strings.HasPrefix(trimmedLine, "- path:") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) > 1 {
				configPaths = append(configPaths, strings.TrimSpace(parts[1]))
			}
		}

		// Only exit the config section if we're at a new top-level section (indicated by a line with no indentation)
		if inConfigSection && !strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "#") &&
			trimmedLine != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
	}

	return configPaths
}
