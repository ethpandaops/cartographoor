package inventory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
)

// Generator is responsible for generating inventory data.
type Generator struct {
	log     *logrus.Entry
	fetcher *Fetcher
}

// NewGenerator creates a new Generator instance.
func NewGenerator(log *logrus.Entry) *Generator {
	return &Generator{
		log:     log.WithField("component", "inventory_generator"),
		fetcher: NewFetcher(log),
	}
}

// GenerateForNetwork generates inventory data for a specific network.
func (g *Generator) GenerateForNetwork(ctx context.Context, fullNetworkName string, network discovery.Network) (*InventoryData, error) {
	// Check if the network has a Dora URL
	if network.ServiceURLs == nil || network.ServiceURLs.Dora == "" {
		g.log.WithField("network", network.Name).Debug("No Dora URL found for network, skipping inventory generation")

		//nolint:nilnil // Returning nil is intentional - no Dora URL means no inventory to generate
		return nil, nil
	}

	doraURL := network.ServiceURLs.Dora

	// Perform health check before attempting to fetch data
	if err := g.fetcher.CheckHealth(ctx, doraURL); err != nil {
		g.log.WithFields(logrus.Fields{
			"network": network.Name,
			"doraURL": doraURL,
			"error":   err,
		}).Warn("Dora health check failed, skipping inventory generation")

		//nolint:nilnil,nilerr // Returning nil is intentional - unhealthy Dora means we skip, not fail
		return nil, nil
	}

	g.log.WithFields(logrus.Fields{
		"network": network.Name,
		"doraURL": doraURL,
	}).Info("Generating inventory for network")

	// Fetch consensus and execution clients concurrently
	var (
		consensusClients []DoraConsensusClient
		executionClients []DoraExecutionClient
		consensusErr     error
		executionErr     error
		wg               sync.WaitGroup
	)

	wg.Add(2)

	// Fetch consensus clients
	go func() {
		defer wg.Done()

		consensusClients, consensusErr = g.fetcher.FetchConsensusClients(ctx, doraURL)
	}()

	// Fetch execution clients
	go func() {
		defer wg.Done()

		executionClients, executionErr = g.fetcher.FetchExecutionClients(ctx, doraURL)
	}()

	wg.Wait()

	// Check for errors
	if consensusErr != nil {
		g.log.WithError(consensusErr).WithField("network", network.Name).Error("Failed to fetch consensus clients")

		return nil, fmt.Errorf("failed to fetch consensus clients: %w", consensusErr)
	}

	if executionErr != nil {
		g.log.WithError(executionErr).WithField("network", network.Name).Error("Failed to fetch execution clients")

		return nil, fmt.Errorf("failed to fetch execution clients: %w", executionErr)
	}

	// Process clients and match with docker images
	inventory := &InventoryData{
		Network:          fullNetworkName,
		Repository:       network.Repository,
		LastUpdated:      time.Now().UTC(),
		ConsensusClients: make([]ClientInfo, 0, len(consensusClients)),
		ExecutionClients: make([]ClientInfo, 0, len(executionClients)),
	}

	// Process consensus clients
	for _, client := range consensusClients {
		clientInfo := g.processConsensusClient(client, fullNetworkName, network)
		inventory.ConsensusClients = append(inventory.ConsensusClients, clientInfo)
	}

	// Process execution clients
	for _, client := range executionClients {
		clientInfo := g.processExecutionClient(client, fullNetworkName, network)
		inventory.ExecutionClients = append(inventory.ExecutionClients, clientInfo)
	}

	g.log.WithFields(logrus.Fields{
		"network":           network.Name,
		"consensus_clients": len(inventory.ConsensusClients),
		"execution_clients": len(inventory.ExecutionClients),
	}).Info("Generated inventory for network")

	return inventory, nil
}

// processConsensusClient processes a consensus client from Dora API.
func (g *Generator) processConsensusClient(client DoraConsensusClient, fullNetworkName string, network discovery.Network) ClientInfo {
	// Validate and normalize the client type
	clientType := client.ClientType
	if clientType != "" {
		// Normalize to match our constants
		normalizedType := strings.ToLower(strings.TrimSpace(clientType))
		if !discovery.IsKnownClient(normalizedType) {
			g.log.WithFields(logrus.Fields{
				"clientName": client.ClientName,
				"clientType": clientType,
				"version":    client.Version,
			}).Debug("Unknown consensus client type detected")
		}

		clientType = normalizedType
	}

	// Construct SSH and BeaconAPI URLs based on DNS type
	var ssh, beaconAPI string
	if network.SelfHostedDNS {
		// Self-hosted DNS pattern
		ssh = fmt.Sprintf("devops@%s.srv.%s.ethpandaops.io", client.ClientName, fullNetworkName)
		beaconAPI = fmt.Sprintf("bn-%s.srv.%s.ethpandaops.io", client.ClientName, fullNetworkName)
	} else {
		// Cloudflare DNS pattern
		ssh = fmt.Sprintf("devops@%s.%s.ethpandaops.io", client.ClientName, fullNetworkName)
		beaconAPI = fmt.Sprintf("bn.%s.%s.ethpandaops.io", client.ClientName, fullNetworkName)
	}

	return ClientInfo{
		ClientName:  client.ClientName,
		ClientType:  clientType,
		Version:     client.Version,
		DockerImage: g.matchDockerImage(clientType, network.Images),
		PeerID:      client.PeerID,
		// NodeID omitted for consensus clients as it duplicates PeerID
		ENR:           client.ENR,
		Status:        client.Status,
		PeerCount:     client.PeerCount,
		PeersInbound:  client.PeersInbound,
		PeersOutbound: client.PeersOutbound,
		SSH:           ssh,
		BeaconAPI:     beaconAPI,
		Metadata:      g.convertMetadata(client.Metadata),
	}
}

// processExecutionClient processes an execution client from Dora API.
func (g *Generator) processExecutionClient(client DoraExecutionClient, fullNetworkName string, network discovery.Network) ClientInfo {
	// Extract client type from version string if not provided
	clientType := client.ClientType
	if clientType == "" && client.Version != "" {
		// Use centralized extraction function
		extractedType, isKnown := discovery.ExtractClientTypeFromVersion(client.Version)
		if extractedType != "" {
			clientType = extractedType
			if !isKnown {
				g.log.WithFields(logrus.Fields{
					"clientName": client.ClientName,
					"clientType": clientType,
					"version":    client.Version,
				}).Debug("Unknown execution client type extracted from version")
			}
		}
	} else if clientType != "" {
		// Normalize existing client type
		clientType = strings.ToLower(strings.TrimSpace(clientType))
	}

	// Debug log for docker image matching
	dockerImage := g.matchDockerImage(clientType, network.Images)
	if dockerImage == "" && clientType != "" {
		g.log.WithFields(logrus.Fields{
			"clientName": client.ClientName,
			"clientType": clientType,
			"version":    client.Version,
		}).Debug("No docker image found for execution client")
	}

	// Construct SSH and RPC URLs based on DNS type
	var ssh, rpc string
	if network.SelfHostedDNS {
		// Self-hosted DNS pattern
		ssh = fmt.Sprintf("devops@%s.srv.%s.ethpandaops.io", client.ClientName, fullNetworkName)
		rpc = fmt.Sprintf("rpc-%s.srv.%s.ethpandaops.io", client.ClientName, fullNetworkName)
	} else {
		// Cloudflare DNS pattern
		ssh = fmt.Sprintf("devops@%s.%s.ethpandaops.io", client.ClientName, fullNetworkName)
		rpc = fmt.Sprintf("rpc.%s.%s.ethpandaops.io", client.ClientName, fullNetworkName)
	}

	info := ClientInfo{
		ClientName:  client.ClientName,
		ClientType:  clientType,
		Version:     client.Version,
		DockerImage: dockerImage,
		PeerID:      client.PeerID,
		NodeID:      client.NodeID,
		Enode:       client.Enode,
		Status:      client.Status,
		SSH:         ssh,
		RPC:         rpc,
		Metadata:    make(map[string]string),
	}

	// Only include peer counts if they're non-zero
	if client.PeerCount > 0 || client.PeersInbound > 0 || client.PeersOutbound > 0 {
		info.PeerCount = client.PeerCount
		info.PeersInbound = client.PeersInbound
		info.PeersOutbound = client.PeersOutbound
	}

	return info
}

// matchDockerImage matches a client type with a docker image from the network configuration.
func (g *Generator) matchDockerImage(clientType string, images *discovery.Images) string {
	if images == nil || len(images.Clients) == 0 || clientType == "" {
		return ""
	}

	// Normalize the client type for comparison
	normalizedClientType := strings.ToLower(strings.TrimSpace(clientType))

	// Try to find a matching image from the discovered images
	for _, img := range images.Clients {
		// Normalize the image name for comparison
		imgName := strings.ToLower(img.Name)

		// Check for exact match
		if imgName == normalizedClientType {
			// Version field contains the docker image version from images.yaml
			return img.Version
		}

		// Check for special cases based on known client mappings
		// These handle cases where the image name differs from the client type
		if normalizedClientType == discovery.CLPrysm && imgName == "prysm_validator" {
			return img.Version
		}

		if normalizedClientType == discovery.CLNimbus && imgName == "nimbusel" {
			return img.Version
		}

		// Check if image name matches execution layer Nimbus variant
		if normalizedClientType == discovery.ELNimbusel && imgName == "nimbusel" {
			return img.Version
		}
	}

	return ""
}

// convertMetadata converts metadata from interface{} to string map.
func (g *Generator) convertMetadata(input map[string]interface{}) map[string]string {
	result := make(map[string]string)

	for key, value := range input {
		if value != nil {
			// Convert value to string
			switch v := value.(type) {
			case string:
				result[key] = v
			case bool:
				if v {
					result[key] = "true"
				} else {
					result[key] = "false"
				}
			case int, int32, int64, uint, uint32, uint64:
				result[key] = fmt.Sprintf("%d", v)
			case float32, float64:
				result[key] = fmt.Sprintf("%v", v)
			default:
				result[key] = fmt.Sprintf("%v", v)
			}
		}
	}

	return result
}
