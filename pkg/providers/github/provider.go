package github

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

const (
	networkConfigDir = "network-configs"
)

// Provider implements the discovery.Provider interface for GitHub.
type Provider struct {
	log    *logrus.Logger
	client *gh.Client
}

// NewProvider creates a new GitHub provider.
func NewProvider(log *logrus.Logger) (*Provider, error) {
	log = log.WithField("provider", "github").Logger

	return &Provider{
		log: log,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "github"
}

// Discover discovers networks using GitHub.
func (p *Provider) Discover(ctx context.Context, config discovery.Config) ([]discovery.Network, error) {
	if len(config.GitHub.Repositories) == 0 {
		return nil, fmt.Errorf("no repositories configured")
	}

	// Create GitHub client
	client := p.getClient(ctx, config.GitHub.Token)

	var networks []discovery.Network

	// Discover networks for each repository
	for _, repoPath := range config.GitHub.Repositories {
		parts := strings.Split(repoPath, "/")
		if len(parts) != 2 {
			p.log.WithField("repository", repoPath).Warn("Invalid repository path")
			continue
		}

		owner, repo := parts[0], parts[1]
		p.log.WithFields(logrus.Fields{
			"owner": owner,
			"repo":  repo,
		}).Info("Discovering networks in repository")

		// Check if network-configs directory exists
		netConfigPath := networkConfigDir
		_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, netConfigPath, nil)
		if err != nil {
			p.log.WithError(err).WithFields(logrus.Fields{
				"owner": owner,
				"repo":  repo,
				"path":  netConfigPath,
			}).Error("Failed to get contents of network-configs directory")
			continue
		}

		// Process directories in network-configs
		for _, content := range dirContent {
			if *content.Type != "dir" {
				continue
			}

			networkName := *content.Name
			p.log.WithFields(logrus.Fields{
				"owner":   owner,
				"repo":    repo,
				"network": networkName,
			}).Debug("Found network")

			// Create network
			network := discovery.Network{
				Name:        networkName,
				Repository:  repoPath,
				Path:        path.Join(netConfigPath, networkName),
				URL:         *content.HTMLURL,
				Status:      "active",
				LastUpdated: time.Now(), // ideally would use last commit time
			}

			networks = append(networks, network)
		}
	}

	return networks, nil
}

// getClient returns a GitHub client.
func (p *Provider) getClient(ctx context.Context, token string) *gh.Client {
	if p.client != nil {
		return p.client
	}

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		p.client = gh.NewClient(tc)
	} else {
		p.client = gh.NewClient(nil)
	}

	return p.client
}