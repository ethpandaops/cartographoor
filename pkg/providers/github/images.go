package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	gh "github.com/google/go-github/v53/github"
)

const (
	// Path to the images.yaml file in the repository.
	imagesYamlPath = "ansible/inventories/%s/group_vars/all/images.yaml"
)

// getImages fetches and parses the images.yaml file for a network.
func (p *Provider) getImages(
	ctx context.Context,
	client *gh.Client,
	owner, repo, networkName string,
) (*discovery.Images, error) {
	// The images.yaml file is typically found in the ansible/inventories/{networkName}/group_vars/all/ directory.
	imagePath := fmt.Sprintf(imagesYamlPath, networkName)

	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, imagePath, nil)
	if err != nil || fileContent == nil {
		p.log.WithError(err).WithFields(map[string]interface{}{
			"network": networkName,
			"path":    imagePath,
		}).Debug("Failed to get images.yaml file")

		return nil, err
	}

	// Get content from GitHub response
	content, err := fileContent.GetContent()
	if err != nil {
		p.log.WithError(err).WithField("network", networkName).Debug("Failed to decode images.yaml content")

		return nil, err
	}

	// Construct the GitHub URL to the file
	fileURL := fmt.Sprintf("https://github.com/%s/%s/blob/master/%s", owner, repo, imagePath)

	// Parse the YAML content to extract client and tool images
	clients, tools := p.parseImagesYaml(content, networkName)

	return &discovery.Images{
		URL:     fileURL,
		Clients: clients,
		Tools:   tools,
	}, nil
}

// parseImagesYaml parses the images.yaml content to extract client and tool images.
func (p *Provider) parseImagesYaml(content, networkName string) ([]discovery.ClientImage, []discovery.ToolImage) {
	var (
		clients []discovery.ClientImage
		tools   []discovery.ToolImage
	)

	lines := strings.Split(content, "\n")

	var (
		inClientSection bool
		inToolSection   bool
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(trimmedLine, "default_ethereum_client_images:") {
			inClientSection = true
			inToolSection = false

			continue
		} else if strings.HasPrefix(trimmedLine, "default_tooling_images:") {
			inClientSection = false
			inToolSection = true

			continue
		}

		// If we're in a section and have an indented line with a colon, it's a key-value pair
		if (inClientSection || inToolSection) && strings.Contains(trimmedLine, ":") && strings.HasPrefix(line, " ") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				var (
					name      = strings.TrimSpace(parts[0])
					fullImage = strings.TrimSpace(parts[1])
				)

				// Handle empty values
				if fullImage == "" {
					if inClientSection {
						clients = append(clients, discovery.ClientImage{
							Name:    name,
							Version: "",
						})
					} else if inToolSection {
						tools = append(tools, discovery.ToolImage{
							Name:    name,
							Version: "",
						})
					}

					continue
				}

				// Extract version from the full image tag
				version := ""

				if strings.Contains(fullImage, ":") {
					// Normal case - extract version from after colon
					imageParts := strings.Split(fullImage, ":")
					if len(imageParts) > 1 {
						version = imageParts[len(imageParts)-1]
					}
				} else {
					// If the value doesn't contain a colon:
					if strings.HasPrefix(fullImage, "docker.") {
						// For docker URLs without a tag, use the last path segment as version
						urlParts := strings.Split(fullImage, "/")
						if len(urlParts) > 0 {
							version = urlParts[len(urlParts)-1]
						}
					} else {
						// For simple values like "electra", use as version
						version = fullImage
					}
				}

				if inClientSection {
					clients = append(clients, discovery.ClientImage{
						Name:    name,
						Version: version,
					})
				} else if inToolSection {
					tools = append(tools, discovery.ToolImage{
						Name:    name,
						Version: version,
					})
				}
			}
		}
	}

	return clients, tools
}
