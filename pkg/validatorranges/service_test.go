package validatorranges

import "testing"

func TestInventoryDirName(t *testing.T) {
	tests := []struct {
		network string
		repo    string
		want    string
	}{
		{network: "fusaka-devnet-5", repo: "ethpandaops/fusaka-devnets", want: "devnet-5"},
		{network: "glamsterdam-devnet-8", repo: "ethpandaops/glamsterdam-devnets", want: "devnet-8"},
		{network: "glamsterdam-msf-1", repo: "ethpandaops/glamsterdam-devnets", want: "msf-1"},
		{network: "heze-devnet-1", repo: "ethpandaops/heze-devnets", want: "devnet-1"},
		{network: "mainnet", repo: "ethpandaops/ansible", want: "mainnet"},
		{network: "hoodi", repo: "", want: "hoodi"},
		{network: "other-devnet-0", repo: "ethpandaops/glamsterdam-devnets", want: "other-devnet-0"},
		{network: "devnet-3", repo: "ethpandaops/devnets", want: "devnet-3"},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			if got := inventoryDirName(tt.network, tt.repo); got != tt.want {
				t.Errorf("inventoryDirName(%q, %q) = %q, want %q", tt.network, tt.repo, got, tt.want)
			}
		})
	}
}
