package github

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"gopkg.in/yaml.v3"
)

// parseConfigYAML extracts chainId, genesisTime and genesisDelay from config.yaml file.
func (p *Provider) parseConfigYAML(
	ctx context.Context,
	owner, repo, networkName string,
) (chainID uint64, genesisTime uint64, genesisDelay uint64, err error) {
	// Construct path to config.yaml
	configPath := path.Join(networkConfigDir, networkName, "metadata", "config.yaml")

	// Try to get file content
	fileContent, _, _, err := p.githubClient.Repositories.GetContents(ctx, owner, repo, configPath, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get config.yaml: %w", err)
	}

	// Decode content
	content, err := fileContent.GetContent()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to decode config.yaml content: %w", err)
	}

	// Parse YAML
	var configData map[string]interface{}
	if yamlErr := yaml.Unmarshal([]byte(content), &configData); yamlErr != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse config.yaml: %w", yamlErr)
	}

	// Extract DEPOSIT_CHAIN_ID
	if depositChainIDVal, ok := configData["DEPOSIT_CHAIN_ID"]; ok {
		switch v := depositChainIDVal.(type) {
		case int:
			if v >= 0 {
				chainID = uint64(v)
			}
		case int64:
			if v >= 0 {
				chainID = uint64(v)
			}
		case uint64:
			chainID = v
		case string:
			chainID, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				p.log.WithError(err).WithField("network", networkName).Debug("Failed to parse DEPOSIT_CHAIN_ID as uint64")
			}
		default:
			p.log.WithField("network", networkName).Debug("DEPOSIT_CHAIN_ID has unexpected type")
		}
	}

	// Extract MIN_GENESIS_TIME
	if minGenesisTimeVal, ok := configData["MIN_GENESIS_TIME"]; ok {
		switch v := minGenesisTimeVal.(type) {
		case int:
			if v >= 0 {
				genesisTime = uint64(v)
			}
		case int64:
			if v >= 0 {
				genesisTime = uint64(v)
			}
		case uint64:
			genesisTime = v
		case string:
			genesisTime, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				p.log.WithError(err).WithField("network", networkName).Debug("Failed to parse MIN_GENESIS_TIME as uint64")
			}
		default:
			p.log.WithField("network", networkName).Debug("MIN_GENESIS_TIME has unexpected type")
		}
	}

	// Extract GENESIS_DELAY
	if genesisDelayVal, ok := configData["GENESIS_DELAY"]; ok {
		switch v := genesisDelayVal.(type) {
		case int:
			if v >= 0 {
				genesisDelay = uint64(v)
			}
		case int64:
			if v >= 0 {
				genesisDelay = uint64(v)
			}
		case uint64:
			genesisDelay = v
		case string:
			genesisDelay, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				p.log.WithError(err).WithField("network", networkName).Debug("Failed to parse GENESIS_DELAY as uint64")
			}
		default:
			p.log.WithField("network", networkName).Debug("GENESIS_DELAY has unexpected type")
		}
	}

	return chainID, genesisTime, genesisDelay, nil
}
