package github

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"gopkg.in/yaml.v3"
)

// parseConfigYAML extracts chainId, genesisTime, genesisDelay and fork epochs from config.yaml file.
func (p *Provider) parseConfigYAML(
	ctx context.Context,
	owner, repo, networkName string,
) (chainID uint64, genesisTime uint64, genesisDelay uint64, forks *discovery.ForksConfig, err error) {
	// Construct path to config.yaml
	configPath := path.Join(networkConfigDir, networkName, "metadata", "config.yaml")

	// Try to get file content
	fileContent, _, _, err := p.githubClient.Repositories.GetContents(ctx, owner, repo, configPath, nil)
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("failed to get config.yaml: %w", err)
	}

	// Decode content
	content, err := fileContent.GetContent()
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("failed to decode config.yaml content: %w", err)
	}

	// Parse YAML
	var configData map[string]interface{}
	if yamlErr := yaml.Unmarshal([]byte(content), &configData); yamlErr != nil {
		return 0, 0, 0, nil, fmt.Errorf("failed to parse config.yaml: %w", yamlErr)
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

	// Extract fork epochs
	forks = p.extractForkEpochs(configData, networkName)

	return chainID, genesisTime, genesisDelay, forks, nil
}

// extractForkEpochs extracts fork epochs from the config data.
func (p *Provider) extractForkEpochs(configData map[string]interface{}, networkName string) *discovery.ForksConfig {
	consensusForks := make(map[string]discovery.ForkConfig)

	for key, value := range configData {
		// Look for keys ending with _FORK_EPOCH (case-insensitive)
		upperKey := strings.ToUpper(key)
		if !strings.HasSuffix(upperKey, "_FORK_EPOCH") {
			continue
		}

		// Extract fork name (everything before _FORK_EPOCH)
		forkName := strings.TrimSuffix(upperKey, "_FORK_EPOCH")
		forkName = strings.ToLower(forkName)

		// Parse epoch value
		epoch, ok := p.parseEpochValue(value, networkName, forkName)
		if !ok {
			continue
		}

		// Add to consensus forks
		consensusForks[forkName] = discovery.ForkConfig{
			Epoch: epoch,
		}
	}

	// Only create ForksConfig if we found at least one fork
	if len(consensusForks) > 0 {
		return &discovery.ForksConfig{
			Consensus: consensusForks,
		}
	}

	return nil
}

// parseEpochValue parses an epoch value from various types.
func (p *Provider) parseEpochValue(value interface{}, networkName, forkName string) (uint64, bool) {
	switch v := value.(type) {
	case int:
		if v >= 0 {
			return uint64(v), true
		}

		p.log.WithField("network", networkName).WithField("fork", forkName).Debug("Fork epoch value is negative, skipping")

		return 0, false
	case int64:
		if v >= 0 {
			return uint64(v), true
		}

		p.log.WithField("network", networkName).WithField("fork", forkName).Debug("Fork epoch value is negative, skipping")

		return 0, false
	case uint64:
		return v, true
	case string:
		parsedEpoch, parseErr := strconv.ParseUint(v, 10, 64)
		if parseErr != nil {
			p.log.WithError(parseErr).WithField("network", networkName).WithField("fork", forkName).Debug("Failed to parse fork epoch as uint64")

			return 0, false
		}

		return parsedEpoch, true
	default:
		p.log.WithField("network", networkName).WithField("fork", forkName).Debug("Fork epoch has unexpected type")

		return 0, false
	}
}
