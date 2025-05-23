package github

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
)

// parseGenesisJSON extracts chainId and genesisTime from genesis.json file.
func (p *Provider) parseGenesisJSON(
	ctx context.Context,
	owner, repo, networkName string,
) (chainID uint64, genesisTime uint64, err error) {
	// Construct path to genesis.json
	genesisPath := path.Join(networkConfigDir, networkName, "metadata", "genesis.json")

	// Try to get file content
	fileContent, _, _, err := p.githubClient.Repositories.GetContents(ctx, owner, repo, genesisPath, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get genesis.json: %w", err)
	}

	// Decode content
	content, err := fileContent.GetContent()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode genesis.json content: %w", err)
	}

	// Parse JSON
	var genesisData map[string]interface{}
	if gdErr := json.Unmarshal([]byte(content), &genesisData); gdErr != nil {
		return 0, 0, fmt.Errorf("failed to parse genesis.json: %w", gdErr)
	}

	// Extract chainId
	config, ok := genesisData["config"].(map[string]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("config not found in genesis.json")
	}

	// Get chainId (could be numeric or string)
	chainIDVal := config["chainId"]

	switch v := chainIDVal.(type) {
	case float64:
		chainID = uint64(v)
	case json.Number:
		chainID, err = strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse chainId as uint64: %w", err)
		}
	case string:
		chainID, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse chainId as uint64: %w", err)
		}
	default:
		return 0, 0, fmt.Errorf("chainId has unexpected type")
	}

	// Extract timestamp from genesis.json
	timestampVal, ok := genesisData["timestamp"]
	if ok {
		// Parse timestamp based on its type
		switch v := timestampVal.(type) {
		case float64:
			genesisTime = uint64(v)
		case json.Number:
			genesisTime, err = strconv.ParseUint(string(v), 10, 64)
			if err != nil {
				p.log.WithError(err).WithField("network", networkName).Debug("Failed to parse timestamp as uint64")
			}
		case string:
			genesisTime, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				p.log.WithError(err).WithField("network", networkName).Debug("Failed to parse timestamp as uint64")
			}
		default:
			p.log.WithField("network", networkName).Debug("Timestamp has unexpected type")
		}
	}

	return chainID, genesisTime, nil
}
