package inventory

import (
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

// InventoryData represents the complete inventory data for a network.
type InventoryData struct {
	Network          string       `json:"network"`
	Repository       string       `json:"repository"`
	LastUpdated      time.Time    `json:"lastUpdated"`
	ConsensusClients []ClientInfo `json:"consensusClients"`
	ExecutionClients []ClientInfo `json:"executionClients"`
}

// ClientInfo represents information about a single client.
type ClientInfo struct {
	ClientName    string            `json:"clientName"`
	ClientType    string            `json:"clientType"`
	Version       string            `json:"version"`
	DockerImage   string            `json:"dockerImage,omitempty"`
	PeerID        string            `json:"peerId,omitempty"`
	NodeID        string            `json:"nodeId,omitempty"`
	ENR           string            `json:"enr,omitempty"`
	Enode         string            `json:"enode,omitempty"`
	Status        string            `json:"status"`
	PeerCount     int               `json:"peerCount,omitempty"`
	PeersInbound  int               `json:"peersInbound,omitempty"`
	PeersOutbound int               `json:"peersOutbound,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// DoraConsensusResponse represents the response from Dora consensus clients API.
type DoraConsensusResponse struct {
	Clients []DoraConsensusClient `json:"clients"`
}

// DoraExecutionResponse represents the response from Dora execution clients API.
type DoraExecutionResponse struct {
	Clients []DoraExecutionClient `json:"clients"`
}

// DoraConsensusClient represents a consensus client from Dora API.
// The JSON tags use snake_case to match the Dora API response format.
//
//nolint:tagliatelle // Dora API uses snake_case
type DoraConsensusClient struct {
	ClientName    string                 `json:"client_name"`
	ClientType    string                 `json:"client_type"`
	Version       string                 `json:"version"`
	PeerID        string                 `json:"peer_id"`
	NodeID        string                 `json:"node_id"`
	ENR           string                 `json:"enr,omitempty"`
	Status        string                 `json:"status"`
	PeerCount     int                    `json:"peer_count"`
	PeersInbound  int                    `json:"peers_inbound"`
	PeersOutbound int                    `json:"peers_outbound"`
	Metadata      map[string]interface{} `json:"metadata"`
	// Blockchain fields are intentionally omitted (head_slot, head_root)
}

// DoraExecutionClient represents an execution client from Dora API.
// The JSON tags use snake_case to match the Dora API response format.
//
//nolint:tagliatelle // Dora API uses snake_case
type DoraExecutionClient struct {
	ClientName    string `json:"client_name"`
	ClientType    string `json:"client_type"`
	Version       string `json:"version"`
	PeerID        string `json:"peer_id"`
	NodeID        string `json:"node_id"`
	Enode         string `json:"enode,omitempty"`
	Status        string `json:"status"`
	PeerCount     int    `json:"peer_count"`
	PeersInbound  int    `json:"peers_inbound"`
	PeersOutbound int    `json:"peers_outbound"`
	// Blockchain fields are intentionally omitted (block_number, block_hash)
}

// NetworksResult represents a simplified version of discovery.Result for inventory.
// It only includes the fields needed for inventory generation, avoiding the Provider interface.
type NetworksResult struct {
	NetworkMetadata map[string]discovery.RepositoryMetadata `json:"networkMetadata"`
	Networks        map[string]discovery.Network            `json:"networks"`
	Clients         map[string]discovery.ClientInfo         `json:"clients"`
	LastUpdate      time.Time                               `json:"lastUpdate"`
	Duration        float64                                 `json:"duration"`
	// Providers field is intentionally omitted to avoid unmarshal issues
}
