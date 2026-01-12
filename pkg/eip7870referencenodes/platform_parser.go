package eip7870referencenodes

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrImageNotFound is returned when image configuration is not found for a client.
var ErrImageNotFound = errors.New("image configuration not found")

// PlatformParser parses platform repository YAML files for client and feature configurations.
type PlatformParser struct {
	secretPatterns []string
}

// NewPlatformParser creates a new PlatformParser with the given secret patterns to redact.
func NewPlatformParser(secretPatterns []string) *PlatformParser {
	if len(secretPatterns) == 0 {
		secretPatterns = DefaultSecretPatterns
	}

	return &PlatformParser{
		secretPatterns: secretPatterns,
	}
}

// ClientOverlay contains the parsed client-specific configuration.
type ClientOverlay struct {
	Args  []string
	Image ImageInfo
}

// FeatureOverlay contains the parsed feature-specific configuration.
type FeatureOverlay struct {
	Args    []string
	EnvVars map[string]string
	Image   *ImageInfo // Optional override from feature
}

// ParseClientConfig parses a client configuration YAML file from the platform repo.
// It extracts client-specific arguments and image information.
func (p *PlatformParser) ParseClientConfig(yamlData []byte, client string) (*ClientOverlay, error) {
	// First, parse for client args using the nested structure
	var clientConfig PlatformClientConfig
	if err := yaml.Unmarshal(yamlData, &clientConfig); err != nil {
		return nil, fmt.Errorf("failed to parse client config YAML: %w", err)
	}

	overlay := &ClientOverlay{
		Args: make([]string, 0),
	}

	// Extract client-specific args
	if args, ok := clientConfig.EthereumNode.Global.ClientArgs.Clients.Execution[client]; ok {
		overlay.Args = args
	}

	// Parse image configuration - need a separate struct for this
	// because it's at a different path: ethereum-node.<client>.image
	imageConfig, err := p.parseClientImage(yamlData, client)
	if err == nil {
		overlay.Image = *imageConfig
	} else if !errors.Is(err, ErrImageNotFound) {
		return nil, fmt.Errorf("failed to parse client image: %w", err)
	}

	return overlay, nil
}

// parseClientImage extracts the image configuration for a specific client.
func (p *PlatformParser) parseClientImage(yamlData []byte, client string) (*ImageInfo, error) {
	// Parse into a generic map to access the nested client image
	var raw map[string]any
	if err := yaml.Unmarshal(yamlData, &raw); err != nil {
		return nil, err
	}

	// Navigate to ethereum-node.<client>.image
	ethereumNode, ok := raw["ethereum-node"].(map[string]any)
	if !ok {
		return nil, ErrImageNotFound
	}

	clientData, ok := ethereumNode[client].(map[string]any)
	if !ok {
		return nil, ErrImageNotFound
	}

	imageData, ok := clientData["image"].(map[string]any)
	if !ok {
		return nil, ErrImageNotFound
	}

	image := &ImageInfo{}

	if repo, ok := imageData["repository"].(string); ok {
		image.Repository = repo
	}

	if tag, ok := imageData["tag"].(string); ok {
		image.Tag = tag
	}

	return image, nil
}

// ParseFeatureConfig parses the 7870-reference feature YAML file.
// It extracts feature-specific arguments, environment variables, and image overrides.
func (p *PlatformParser) ParseFeatureConfig(yamlData []byte, client string) (*FeatureOverlay, error) {
	var featureConfig PlatformFeatureConfig
	if err := yaml.Unmarshal(yamlData, &featureConfig); err != nil {
		return nil, fmt.Errorf("failed to parse feature config YAML: %w", err)
	}

	overlay := &FeatureOverlay{
		Args:    make([]string, 0),
		EnvVars: make(map[string]string),
	}

	// Extract feature-specific args from global.clientArgs.features.7870-reference.execution.<client>
	if feature, ok := featureConfig.Global.ClientArgs.Features["7870-reference"]; ok {
		if args, ok := feature.Execution[client]; ok {
			overlay.Args = args
		}
	}

	// Extract environment variables and image overrides from ethereum-node.<client>
	if clientConfig, ok := featureConfig.EthereumNode[client]; ok {
		// Process extra env vars
		for _, env := range clientConfig.ExtraEnv {
			// Redact secrets
			value := p.redactSecrets(env.Value)
			overlay.EnvVars[env.Name] = value
		}

		// Check for image override
		if clientConfig.Image.Repository != "" || clientConfig.Image.Tag != "" {
			overlay.Image = &ImageInfo{
				Repository: clientConfig.Image.Repository,
				Tag:        clientConfig.Image.Tag,
			}
		}
	}

	return overlay, nil
}

// redactSecrets replaces secret patterns with a placeholder.
func (p *PlatformParser) redactSecrets(value string) string {
	for _, pattern := range p.secretPatterns {
		if strings.Contains(value, pattern) {
			return "$AUTH_HEADER (provide your own authentication)"
		}
	}

	return value
}

// HasKeelAnnotations checks if the feature config has Keel auto-update annotations.
func (p *PlatformParser) HasKeelAnnotations(yamlData []byte, client string) bool {
	var featureConfig PlatformFeatureConfig
	if err := yaml.Unmarshal(yamlData, &featureConfig); err != nil {
		return false
	}

	if clientConfig, ok := featureConfig.EthereumNode[client]; ok {
		if _, hasKeel := clientConfig.Annotations["keel.sh/policy"]; hasKeel {
			return true
		}
	}

	return false
}
