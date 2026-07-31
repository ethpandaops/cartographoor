package validatorranges

import "testing"

func TestStripRepoPrefix(t *testing.T) {
	tests := []struct {
		name        string
		networkName string
		repo        string
		want        string
	}{
		{
			name:        "prefix matches the repo name and is stripped",
			networkName: "glamsterdam-devnet-7",
			repo:        "ethpandaops/glamsterdam-devnets",
			want:        "devnet-7",
		},
		{
			name:        "another repo not in the old hardcoded list still strips correctly",
			networkName: "peerdas-devnet-7",
			repo:        "ethpandaops/peerdas-devnets",
			want:        "devnet-7",
		},
		{
			name:        "repo previously covered by the old hardcoded list still works",
			networkName: "fusaka-devnet-5",
			repo:        "ethpandaops/fusaka-devnets",
			want:        "devnet-5",
		},
		{
			name:        "network name has no matching repo prefix, left unchanged",
			networkName: "mainnet",
			repo:        "ethpandaops/ansible",
			want:        "mainnet",
		},
		{
			name:        "malformed repo path, left unchanged",
			networkName: "glamsterdam-devnet-7",
			repo:        "not-a-valid-repo-path",
			want:        "glamsterdam-devnet-7",
		},
		{
			name:        "empty repo, left unchanged",
			networkName: "glamsterdam-devnet-7",
			repo:        "",
			want:        "glamsterdam-devnet-7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripRepoPrefix(tt.networkName, tt.repo)
			if got != tt.want {
				t.Errorf("stripRepoPrefix(%q, %q) = %q, want %q", tt.networkName, tt.repo, got, tt.want)
			}
		})
	}
}
