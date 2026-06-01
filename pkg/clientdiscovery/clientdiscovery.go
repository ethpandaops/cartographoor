// Package clientdiscovery fetches metadata (latest versions, repositories, etc.)
// about known Ethereum clients from GitHub. It is intentionally kept separate
// from pkg/discovery so that consumers of the discovery data model do not have
// to compile the GitHub client and its dependencies.
package clientdiscovery

import (
	"context"
	"fmt"
	"slices"
	"strings"

	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

// Ensure Discoverer implements discovery.ClientDiscovererInterface.
var _ discovery.ClientDiscovererInterface = (*Discoverer)(nil)

// Discoverer handles fetching information about Ethereum clients from GitHub.
type Discoverer struct {
	log    *logrus.Logger
	client *gh.Client
}

// New creates a new client Discoverer. If token is empty, an unauthenticated
// GitHub client is used (subject to stricter rate limits).
func New(log *logrus.Logger, token string) *Discoverer {
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

	return &Discoverer{
		log:    log,
		client: client,
	}
}

// DiscoverClients fetches information about all known Ethereum clients.
func (d *Discoverer) DiscoverClients(ctx context.Context) (map[string]discovery.ClientInfo, error) {
	d.log.Info("Discovering client information")

	clients := make(map[string]discovery.ClientInfo)

	// Process all clients
	allClients := append(discovery.CLClients, discovery.ELClients...)

	for _, clientName := range allClients {
		repo, ok := discovery.DefaultRepositories[clientName]
		if !ok {
			continue
		}

		branch, ok := discovery.DefaultBranches[clientName]
		if !ok {
			branch = ""
		}

		websiteURL, ok := discovery.DefaultWebsiteURLs[clientName]
		if !ok {
			websiteURL = ""
		}

		docsURL, ok := discovery.DefaultDocsURLs[clientName]
		if !ok {
			docsURL = ""
		}

		displayName, ok := discovery.DefaultDisplayNames[clientName]
		if !ok {
			displayName = clientName
		}

		var clientType string

		if slices.Contains(discovery.ELClients, clientName) {
			clientType = discovery.ELClientType
		} else {
			clientType = discovery.CLClientType
		}

		// Create client info
		clientInfo := discovery.ClientInfo{
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
func (d *Discoverer) getLatestVersion(ctx context.Context, repo string) (string, error) {
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
