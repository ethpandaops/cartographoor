package github

import (
	"context"
	"path"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
)

// determineNetworkStatus determines if a network is active, inactive, or unknown.
func (p *Provider) determineNetworkStatus(
	ctx context.Context,
	client *gh.Client,
	owner, repo, networkName string,
) (string, []string, string) {
	var (
		status      = unknown
		configFiles []string
		domain      string
	)

	// Check if network exists in kubernetes directory (active).
	kubePath := path.Join(kubernetesDir, networkName)

	_, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, kubePath, nil)
	if err == nil || (resp != nil && resp.StatusCode != 404) {
		status = active

		// For active networks, try to get config values
		configFiles, domain = p.getNetworkConfigs(ctx, client, owner, repo, kubePath, networkName)
	} else {
		// Check if network exists in kubernetes-archive directory (inactive).
		archivePath := path.Join(kubernetesArchiveDir, networkName)

		_, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, archivePath, nil)
		if err == nil || (resp != nil && resp.StatusCode != 404) {
			status = inactive
		}
	}

	return status, configFiles, domain
}

// getNetworkDetails fetches configuration details and images for a network.
func (p *Provider) getNetworkDetails(
	ctx context.Context,
	client *gh.Client,
	owner, repo, networkName string,
) (status string, configFiles []string, domain string, images *discovery.Images, hiveURL string) {
	// Get basic network status, configs, and domain
	status, configFiles, domain = p.determineNetworkStatus(ctx, client, owner, repo, networkName)

	// For active networks, try to get images + hive information.
	if status == active {
		images, _ = p.getImages(ctx, client, owner, repo, networkName)
	}

	var err error

	hiveURL, err = p.checkHiveAvailability(ctx, owner, repo, networkName)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"repo":    repo,
			"network": networkName,
		}).Debug("hive is not available for network")
	}

	return status, configFiles, domain, images, hiveURL
}
