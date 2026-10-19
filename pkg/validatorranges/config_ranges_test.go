package validatorranges

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

func TestConfigRangesURL(t *testing.T) {
	tests := []struct {
		name    string
		network discovery.Network
		want    string
	}{
		{
			name: "published endpoint",
			network: discovery.Network{GenesisConfig: &discovery.GenesisConfig{API: []discovery.ConfigFile{
				{Path: "/api/v1/nodes/inventory", URL: "https://config.heze-devnet-1.ethpandaops.io/api/v1/nodes/inventory"},
				{Path: "/api/v1/nodes/validator-ranges", URL: "https://config.heze-devnet-1.ethpandaops.io/api/v1/nodes/validator-ranges"},
			}}},
			want: "https://config.heze-devnet-1.ethpandaops.io/api/v1/nodes/validator-ranges",
		},
		{
			name:    "no genesis config",
			network: discovery.Network{},
			want:    "",
		},
		{
			name: "no validator ranges endpoint",
			network: discovery.Network{GenesisConfig: &discovery.GenesisConfig{API: []discovery.ConfigFile{
				{Path: "/api/v1/nodes/inventory", URL: "https://config.example.io/api/v1/nodes/inventory"},
			}}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configRangesURL(tt.network); got != tt.want {
				t.Errorf("configRangesURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseConfigRanges(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string][]*ValidatorRange
		wantErr bool
	}{
		{
			name: "inclusive keys become exclusive ends, sorted per name",
			content: `{"history":[],"ranges":{
				"0-999":"prysm-ethrex-1",
				"85262-85266":"nflaig",
				"84268-84268":"nflaig",
				"84019-84118":"nflaig"}}`,
			want: map[string][]*ValidatorRange{
				"prysm-ethrex-1": {{Start: 0, End: 1000}},
				"nflaig":         {{Start: 84019, End: 84119}, {Start: 84268, End: 84269}, {Start: 85262, End: 85267}},
			},
		},
		{name: "empty ranges", content: `{"ranges":{}}`, wantErr: true},
		{name: "malformed key", content: `{"ranges":{"12":"a"}}`, wantErr: true},
		{name: "end before start", content: `{"ranges":{"10-5":"a"}}`, wantErr: true},
		{name: "empty name", content: `{"ranges":{"0-5":""}}`, wantErr: true},
		{name: "not json", content: `<html>`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfigRanges([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseConfigRanges() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseConfigRanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyConfigRanges(t *testing.T) {
	inventory := func() *ValidatorRanges {
		return &ValidatorRanges{
			Nodes: map[string]*Node{
				"teku-reth-1": {
					Groups:          []string{"teku_reth"},
					Tags:            []string{"cl:teku", "el:reth"},
					Attributes:      map[string]any{"cloud": "hetzner"},
					ValidatorRanges: []*ValidatorRange{{Start: 0, End: 100}},
					Source:          "ethpandaops",
				},
				"prysm-geth-1": {
					Groups:          []string{"prysm_geth"},
					Tags:            []string{"cl:prysm", "el:geth"},
					Attributes:      map[string]any{},
					ValidatorRanges: []*ValidatorRange{{Start: 100, End: 200}},
					Source:          "ethpandaops",
				},
				"prysm-besu-1": {
					Groups:          []string{"prysm_besu"},
					Tags:            []string{"cl:prysm", "el:besu"},
					Attributes:      map[string]any{},
					ValidatorRanges: []*ValidatorRange{{Start: 200, End: 300}},
					Source:          "ethpandaops",
				},
			},
			Validators: &ValidatorSummary{TotalCount: 300},
			Metadata:   &Metadata{Sources: []string{"https://example.com/inventory.ini"}},
		}
	}

	tests := []struct {
		name       string
		config     map[string][]*ValidatorRange
		wantRanges map[string][]*ValidatorRange
		wantSource map[string]string
		wantTotal  int
	}{
		{
			name: "config-only names are added untagged",
			config: map[string][]*ValidatorRange{
				"teku-reth-1":  {{Start: 0, End: 100}},
				"prysm-geth-1": {{Start: 100, End: 200}},
				"prysm-besu-1": {{Start: 200, End: 300}},
				"potuz":        {{Start: 300, End: 350}},
			},
			wantRanges: map[string][]*ValidatorRange{
				"teku-reth-1":  {{Start: 0, End: 100}},
				"prysm-geth-1": {{Start: 100, End: 200}},
				"prysm-besu-1": {{Start: 200, End: 300}},
				"potuz":        {{Start: 300, End: 350}},
			},
			wantSource: map[string]string{"potuz": configSourceName, "teku-reth-1": "ethpandaops"},
			wantTotal:  350,
		},
		{
			name: "config ranges replace inventory ranges for the same node",
			config: map[string][]*ValidatorRange{
				"teku-reth-1":  {{Start: 0, End: 50}, {Start: 150, End: 200}},
				"prysm-geth-1": {{Start: 50, End: 150}},
			},
			wantRanges: map[string][]*ValidatorRange{
				"teku-reth-1":  {{Start: 0, End: 50}, {Start: 150, End: 200}},
				"prysm-geth-1": {{Start: 50, End: 150}},
				"prysm-besu-1": {{Start: 200, End: 300}},
			},
			wantSource: map[string]string{"teku-reth-1": "ethpandaops"},
			wantTotal:  300,
		},
		{
			name: "inventory-only ranges overlapping config ranges are dropped",
			config: map[string][]*ValidatorRange{
				"teku-reth-1": {{Start: 0, End: 100}},
				"potuz":       {{Start: 150, End: 250}},
			},
			wantRanges: map[string][]*ValidatorRange{
				"teku-reth-1": {{Start: 0, End: 100}},
				"potuz":       {{Start: 150, End: 250}},
			},
			wantSource: map[string]string{"potuz": configSourceName},
			wantTotal:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := inventory()
			ApplyConfigRanges(vr, tt.config, "https://config.example.io/api/v1/nodes/validator-ranges", logrus.New())

			got := make(map[string][]*ValidatorRange, len(vr.Nodes))
			for name, node := range vr.Nodes {
				got[name] = node.ValidatorRanges
			}

			if !reflect.DeepEqual(got, tt.wantRanges) {
				t.Errorf("ranges = %v, want %v", got, tt.wantRanges)
			}

			for name, source := range tt.wantSource {
				if vr.Nodes[name].Source != source {
					t.Errorf("node %s source = %q, want %q", name, vr.Nodes[name].Source, source)
				}
			}

			if vr.Validators.TotalCount != tt.wantTotal {
				t.Errorf("total = %d, want %d", vr.Validators.TotalCount, tt.wantTotal)
			}

			if !reflect.DeepEqual(vr.Tags["cl:teku"], []string{"teku-reth-1"}) {
				t.Errorf("cl:teku tag = %v, want [teku-reth-1]", vr.Tags["cl:teku"])
			}

			if vr.Metadata.Sources[len(vr.Metadata.Sources)-1] != "https://config.example.io/api/v1/nodes/validator-ranges" {
				t.Errorf("sources = %v, want the config URL last", vr.Metadata.Sources)
			}
		})
	}
}
