package discovery

import (
	"context"
	"fmt"
	"slices"
	"strings"

	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Define a list of known clients.
const (
	CLLighthouse = "lighthouse"
	CLPrysm      = "prysm"
	CLLodestar   = "lodestar"
	CLNimbus     = "nimbus"
	CLTeku       = "teku"
	CLGrandine   = "grandine"
	ELNethermind = "nethermind"
	ELNimbusel   = "nimbusel"
	ELBesu       = "besu"
	ELGeth       = "geth"
	ELReth       = "reth"
	ELErigon     = "erigon"
	ELEthereumJS = "ethereumjs"
)

const (
	CLClientType = "consensus"
	ELClientType = "execution"
)

var (
	// Buckets of known clients.
	CLClients = []string{CLLighthouse, CLPrysm, CLLodestar, CLNimbus, CLTeku, CLGrandine}
	ELClients = []string{ELNethermind, ELNimbusel, ELBesu, ELGeth, ELReth, ELErigon, ELEthereumJS}

	// DefaultDisplayNames maps clients to their default display names.
	DefaultDisplayNames = map[string]string{
		CLLighthouse: "Lighthouse",
		CLPrysm:      "Prysm",
		CLLodestar:   "Lodestar",
		CLNimbus:     "Nimbus",
		CLTeku:       "Teku",
		CLGrandine:   "Grandine",
		ELNethermind: "Nethermind",
		ELNimbusel:   "NimbusEL",
		ELBesu:       "Besu",
		ELGeth:       "Geth",
		ELReth:       "Reth",
		ELErigon:     "Erigon",
		ELEthereumJS: "EthereumJS",
	}

	// DefaultRepositories maps clients to their default source repositories.
	DefaultRepositories = map[string]string{
		CLLighthouse: "sigp/lighthouse",
		CLPrysm:      "OffchainLabs/prysm",
		CLLodestar:   "chainsafe/lodestar",
		CLNimbus:     "status-im/nimbus-eth2",
		CLTeku:       "ConsenSys/teku",
		CLGrandine:   "grandinetech/grandine",
		ELNethermind: "NethermindEth/nethermind",
		ELNimbusel:   "status-im/nimbus-eth1",
		ELBesu:       "hyperledger/besu",
		ELGeth:       "ethereum/go-ethereum",
		ELReth:       "paradigmxyz/reth",
		ELErigon:     "erigontech/erigon",
		ELEthereumJS: "ethereumjs/ethereumjs-monorepo",
	}

	// DefaultBranches maps clients to their default branches.
	DefaultBranches = map[string]string{
		CLLighthouse: "stable",
		CLPrysm:      "develop",
		CLLodestar:   "unstable",
		CLNimbus:     "stable",
		CLTeku:       "master",
		CLGrandine:   "develop",
		ELNethermind: "master",
		ELNimbusel:   "master",
		ELBesu:       "main",
		ELGeth:       "master",
		ELReth:       "main",
		ELErigon:     "main",
		ELEthereumJS: "master",
	}

	DefaultWebsiteURLs = map[string]string{
		CLLighthouse: "https://lighthouse.sigmaprime.io/",
		CLTeku:       "https://consensys.io/teku",
		CLPrysm:      "https://www.offchainlabs.com/prysm",
		CLLodestar:   "https://lodestar.chainsafe.io/",
		CLNimbus:     "https://nimbus.team/",
		CLGrandine:   "https://grandine.io/",
		ELNethermind: "https://nethermind.io/",
		ELNimbusel:   "https://nimbus.team/",
		ELBesu:       "https://hyperledger.org/",
		ELGeth:       "https://geth.ethereum.org/",
		ELReth:       "https://www.paradigm.xyz/",
		ELErigon:     "https://erigon.tech/",
		ELEthereumJS: "https://ethereumjs.github.io/",
	}

	DefaultDocsURLs = map[string]string{
		CLLighthouse: "https://lighthouse-book.sigmaprime.io/",
		CLTeku:       "https://docs.teku.consensys.io/",
		CLPrysm:      "https://www.offchainlabs.com/prysm/docs",
		CLLodestar:   "https://chainsafe.github.io/lodestar/",
		CLNimbus:     "https://nimbus.guide/index.html",
		CLGrandine:   "https://docs.grandine.io/",
		ELNethermind: "https://docs.nethermind.io/",
		ELNimbusel:   "https://nimbus.guide/index.html",
		ELBesu:       "https://besu.hyperledger.org/",
		ELGeth:       "https://geth.ethereum.org/docs",
		ELReth:       "https://reth.rs/",
		ELErigon:     "https://docs.erigon.tech/",
		ELEthereumJS: "https://ethereumjs.readthedocs.io/",
	}
)

// Ensure ClientDiscoverer implements ClientDiscovererInterface.
var _ ClientDiscovererInterface = (*ClientDiscoverer)(nil)

// ClientDiscoverer handles fetching information about Ethereum clients.
type ClientDiscoverer struct {
	log    *logrus.Logger
	client *gh.Client
}

// NewClientDiscoverer creates a new ClientDiscoverer.
func NewClientDiscoverer(log *logrus.Logger, token string) *ClientDiscoverer {
	log = log.WithField("module", "client_discoverer").Logger

	var client *gh.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(context.Background(), ts)
		client = gh.NewClient(tc)
	} else {
		client = gh.NewClient(nil)
	}

	return &ClientDiscoverer{
		log:    log,
		client: client,
	}
}

// DiscoverClients fetches information about all known Ethereum clients.
func (d *ClientDiscoverer) DiscoverClients(ctx context.Context) (map[string]ClientInfo, error) {
	d.log.Info("Discovering client information")

	clients := make(map[string]ClientInfo)

	// Process all clients
	allClients := append(CLClients, ELClients...)

	for _, clientName := range allClients {
		repo, ok := DefaultRepositories[clientName]
		if !ok {
			continue
		}

		branch, ok := DefaultBranches[clientName]
		if !ok {
			branch = ""
		}

		websiteURL, ok := DefaultWebsiteURLs[clientName]
		if !ok {
			websiteURL = ""
		}

		docsURL, ok := DefaultDocsURLs[clientName]
		if !ok {
			docsURL = ""
		}

		displayName, ok := DefaultDisplayNames[clientName]
		if !ok {
			displayName = clientName
		}

		var clientType string

		if slices.Contains(ELClients, clientName) {
			clientType = ELClientType
		} else {
			clientType = CLClientType
		}

		// Create client info
		clientInfo := ClientInfo{
			Name:        clientName,
			DisplayName: displayName,
			Repository:  repo,
			Type:        clientType,
			Branch:      branch,
			Logo:        fmt.Sprintf("https://ethpandaops.io/img/clients/%s.jpg", clientName),
			WebsiteURL:  websiteURL,
			DocsURL:     docsURL,
		}

		// Try to fetch the latest version
		version, err := d.getLatestVersion(ctx, repo)
		if err != nil {
			d.log.WithError(err).WithField("client", clientName).Warn("Failed to get latest version")
		} else {
			clientInfo.LatestVersion = version
		}

		clients[clientName] = clientInfo
	}

	d.log.WithField("count", len(clients)).Info("Client discovery complete")

	return clients, nil
}

// getLatestVersion fetches the latest released version of a client.
func (d *ClientDiscoverer) getLatestVersion(ctx context.Context, repo string) (string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repository format: %s", repo)
	}

	owner, repoName := parts[0], parts[1]

	// First try to get the latest release
	release, _, err := d.client.Repositories.GetLatestRelease(ctx, owner, repoName)
	if err == nil && release != nil && release.TagName != nil {
		return *release.TagName, nil
	}

	// If no release found, try to get tags
	opts := &gh.ListOptions{
		PerPage: 1, // We only need the latest tag
	}

	tags, _, err := d.client.Repositories.ListTags(ctx, owner, repoName, opts)
	if err == nil && len(tags) > 0 && tags[0].Name != nil {
		return *tags[0].Name, nil
	}

	// If all else fails, return an empty string
	return "", fmt.Errorf("no version information found")
}
