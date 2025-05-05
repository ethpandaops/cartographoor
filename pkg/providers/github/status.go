package github

import (
	"context"
	"path"

	gh "github.com/google/go-github/v53/github"
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
