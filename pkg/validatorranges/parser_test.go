package validatorranges

import (
	"reflect"
	"testing"
)

func TestParseInventory(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		sourceURL   string
		sourceName  string
		rangeOffset int
		want        *ValidatorRanges
		wantErr     bool
	}{
		{
			name: "basic ethpandaops inventory without offset",
			content: `[lighthouse_geth]
lighthouse-geth-1 ansible_host=192.168.1.1 validator_start=100 validator_end=108 cloud=aws cloud_region=us-east-1
lighthouse-geth-2 ansible_host=192.168.1.2 validator_start=108 validator_end=116 cloud=aws cloud_region=us-west-2

[prysm_nethermind]
prysm-nethermind-1 ansible_host=192.168.1.3 validator_start=200 validator_end=208 ipv6=2001:db8::1
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lighthouse-geth-1": {
						Groups: []string{"lighthouse_geth"},
						Tags:   []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud":       "aws",
							"cloudRegion": "us-east-1",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 100, End: 108},
						},
						Source: "ethpandaops",
					},
					"lighthouse-geth-2": {
						Groups: []string{"lighthouse_geth"},
						Tags:   []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud":       "aws",
							"cloudRegion": "us-west-2",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 108, End: 116},
						},
						Source: "ethpandaops",
					},
					"prysm-nethermind-1": {
						Groups: []string{"prysm_nethermind"},
						Tags:   []string{"el:nethermind", "cl:prysm"},
						Attributes: map[string]interface{}{
							"ipv6": "2001:db8::1",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 200, End: 208},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 24, // (108-100) + (116-108) + (208-200) = 8 + 8 + 8 = 24
				},
				Groups: map[string][]string{
					"lighthouse_geth":  {"lighthouse-geth-1", "lighthouse-geth-2"},
					"prysm_nethermind": {"prysm-nethermind-1"},
				},
				Tags: map[string][]string{
					"el:geth":       {"lighthouse-geth-1", "lighthouse-geth-2"},
					"cl:lighthouse": {"lighthouse-geth-1", "lighthouse-geth-2"},
					"el:nethermind": {"prysm-nethermind-1"},
					"cl:prysm":      {"prysm-nethermind-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "testinprod inventory with rangeOffset",
			content: `[lighthouse_besu]
lighthouse-besu-1 ansible_host=10.0.0.1 validator_start=0 validator_end=8 cloud=digitalocean cloud_region=nyc1
lighthouse-besu-2 ansible_host=10.0.0.2 validator_start=8 validator_end=16 cloud=digitalocean cloud_region=sfo3

[teku_geth]
teku-geth-1 ansible_host=10.0.0.3 validator_start=100 validator_end=108 ethereum_node_cl_supernode_enabled=True
`,
			sourceURL:   "https://testinprod.com/inventory.ini",
			sourceName:  "testinprod",
			rangeOffset: 51312, // Apply offset to map local ranges to global
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lighthouse-besu-1": {
						Groups: []string{"lighthouse_besu"},
						Tags:   []string{"el:besu", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud":       "digitalocean",
							"cloudRegion": "nyc1",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 51312, End: 51320}, // 0+51312 to 8+51312
						},
						Source: "testinprod",
					},
					"lighthouse-besu-2": {
						Groups: []string{"lighthouse_besu"},
						Tags:   []string{"el:besu", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud":       "digitalocean",
							"cloudRegion": "sfo3",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 51320, End: 51328}, // 8+51312 to 16+51312
						},
						Source: "testinprod",
					},
					"teku-geth-1": {
						Groups: []string{"teku_geth"},
						Tags:   []string{"el:geth", "cl:teku"},
						Attributes: map[string]interface{}{
							"isClSupernode": true,
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 51412, End: 51420}, // 100+51312 to 108+51312
						},
						Source: "testinprod",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 24, // 8 + 8 + 8 = 24
				},
				Groups: map[string][]string{
					"lighthouse_besu": {"lighthouse-besu-1", "lighthouse-besu-2"},
					"teku_geth":       {"teku-geth-1"},
				},
				Tags: map[string][]string{
					"el:besu":       {"lighthouse-besu-1", "lighthouse-besu-2"},
					"cl:lighthouse": {"lighthouse-besu-1", "lighthouse-besu-2"},
					"el:geth":       {"teku-geth-1"},
					"cl:teku":       {"teku-geth-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://testinprod.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "nodes without validator ranges are filtered out",
			content: `[bootnode]
bootnode-1 ansible_host=192.168.1.1 cloud=aws

[lighthouse_geth]
lighthouse-geth-1 ansible_host=192.168.1.2 validator_start=100 validator_end=108
lighthouse-geth-2 ansible_host=192.168.1.3

[monitoring]
grafana ansible_host=192.168.1.4
prometheus ansible_host=192.168.1.5
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lighthouse-geth-1": {
						Groups:     []string{"lighthouse_geth"},
						Tags:       []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{},
						ValidatorRanges: []*ValidatorRange{
							{Start: 100, End: 108},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 8,
				},
				Groups: map[string][]string{
					"lighthouse_geth": {"lighthouse-geth-1"},
				},
				Tags: map[string][]string{
					"el:geth":       {"lighthouse-geth-1"},
					"cl:lighthouse": {"lighthouse-geth-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "boolean attribute parsing",
			content: `[nimbus_reth]
nimbus-reth-1 ansible_host=192.168.1.1 validator_start=0 validator_end=8 ethereum_node_cl_supernode_enabled=True
nimbus-reth-2 ansible_host=192.168.1.2 validator_start=8 validator_end=16 ethereum_node_cl_supernode_enabled=false
nimbus-reth-3 ansible_host=192.168.1.3 validator_start=16 validator_end=24 ethereum_node_cl_supernode_enabled=TRUE
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"nimbus-reth-1": {
						Groups: []string{"nimbus_reth"},
						Tags:   []string{"el:reth", "cl:nimbus"},
						Attributes: map[string]interface{}{
							"isClSupernode": true,
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops",
					},
					"nimbus-reth-2": {
						Groups: []string{"nimbus_reth"},
						Tags:   []string{"el:reth", "cl:nimbus"},
						Attributes: map[string]interface{}{
							"isClSupernode": false,
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 8, End: 16},
						},
						Source: "ethpandaops",
					},
					"nimbus-reth-3": {
						Groups: []string{"nimbus_reth"},
						Tags:   []string{"el:reth", "cl:nimbus"},
						Attributes: map[string]interface{}{
							"isClSupernode": true,
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 16, End: 24},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 24, // 8 + 8 + 8 = 24
				},
				Groups: map[string][]string{
					"nimbus_reth": {"nimbus-reth-1", "nimbus-reth-2", "nimbus-reth-3"},
				},
				Tags: map[string][]string{
					"el:reth":   {"nimbus-reth-1", "nimbus-reth-2", "nimbus-reth-3"},
					"cl:nimbus": {"nimbus-reth-1", "nimbus-reth-2", "nimbus-reth-3"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "camelCase attribute conversion",
			content: `[lodestar_erigon]
lodestar-erigon-1 ansible_host=192.168.1.1 validator_start=0 validator_end=8 cloud_region=us-east-1 some_long_attribute=value bandwidth=100
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lodestar-erigon-1": {
						Groups: []string{"lodestar_erigon"},
						Tags:   []string{"el:erigon", "cl:lodestar"},
						Attributes: map[string]interface{}{
							"cloudRegion":       "us-east-1",
							"someLongAttribute": "value",
							"bandwidth":         "100",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 8,
				},
				Groups: map[string][]string{
					"lodestar_erigon": {"lodestar-erigon-1"},
				},
				Tags: map[string][]string{
					"el:erigon":   {"lodestar-erigon-1"},
					"cl:lodestar": {"lodestar-erigon-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "grandine client tag extraction",
			content: `[grandine_besu]
grandine-besu-1 ansible_host=192.168.1.1 validator_start=0 validator_end=8
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"grandine-besu-1": {
						Groups:     []string{"grandine_besu"},
						Tags:       []string{"el:besu", "cl:grandine"},
						Attributes: map[string]interface{}{},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 8,
				},
				Groups: map[string][]string{
					"grandine_besu": {"grandine-besu-1"},
				},
				Tags: map[string][]string{
					"el:besu":     {"grandine-besu-1"},
					"cl:grandine": {"grandine-besu-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "validator client tag",
			content: `[validator_lighthouse]
validator-lighthouse-1 ansible_host=192.168.1.1 validator_start=0 validator_end=8
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"validator-lighthouse-1": {
						Groups:     []string{"validator_lighthouse"},
						Tags:       []string{"cl:lighthouse", "vc:validator"},
						Attributes: map[string]interface{}{},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 8,
				},
				Groups: map[string][]string{
					"validator_lighthouse": {"validator-lighthouse-1"},
				},
				Tags: map[string][]string{
					"cl:lighthouse": {"validator-lighthouse-1"},
					"vc:validator":  {"validator-lighthouse-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name:        "empty inventory",
			content:     ``,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes:      map[string]*Node{},
				Validators: &ValidatorSummary{TotalCount: 0},
				Groups:     map[string][]string{},
				Tags:       map[string][]string{},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
		{
			name: "inventory with standalone hosts (should be filtered)",
			content: `localhost

[lighthouse_geth]
lighthouse-geth-1 ansible_host=192.168.1.1 validator_start=0 validator_end=8

some-random-host

[teku_besu]
teku-besu-1 ansible_host=192.168.1.2 validator_start=100 validator_end=108
`,
			sourceURL:   "https://example.com/inventory.ini",
			sourceName:  "ethpandaops",
			rangeOffset: 0,
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lighthouse-geth-1": {
						Groups:     []string{"lighthouse_geth"},
						Tags:       []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops",
					},
					"teku-besu-1": {
						Groups:     []string{"teku_besu"},
						Tags:       []string{"el:besu", "cl:teku"},
						Attributes: map[string]interface{}{},
						ValidatorRanges: []*ValidatorRange{
							{Start: 100, End: 108},
						},
						Source: "ethpandaops",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 16, // 8 + 8 = 16
				},
				Groups: map[string][]string{
					"lighthouse_geth": {"lighthouse-geth-1"},
					"teku_besu":       {"teku-besu-1"},
				},
				Tags: map[string][]string{
					"el:geth":       {"lighthouse-geth-1"},
					"cl:lighthouse": {"lighthouse-geth-1"},
					"el:besu":       {"teku-besu-1"},
					"cl:teku":       {"teku-besu-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://example.com/inventory.ini"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInventory([]byte(tt.content), tt.sourceURL, tt.sourceName, tt.rangeOffset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInventory() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !tt.wantErr {
				// Check individual fields instead of using DeepEqual on the whole struct
				// because map iteration order is not guaranteed

				// Check nodes
				if !reflect.DeepEqual(got.Nodes, tt.want.Nodes) {
					t.Errorf("Nodes mismatch")
					for name, node := range got.Nodes {
						if wantNode, exists := tt.want.Nodes[name]; exists {
							if !reflect.DeepEqual(node, wantNode) {
								t.Errorf("Node %s:\ngot  %+v\nwant %+v", name, node, wantNode)
							}
						} else {
							t.Errorf("Unexpected node: %s", name)
						}
					}
					for name := range tt.want.Nodes {
						if _, exists := got.Nodes[name]; !exists {
							t.Errorf("Missing node: %s", name)
						}
					}
				}

				// Check validators
				if got.Validators.TotalCount != tt.want.Validators.TotalCount {
					t.Errorf("TotalCount = %v, want %v", got.Validators.TotalCount, tt.want.Validators.TotalCount)
				} else {
					t.Logf("TotalCount matches: %v", got.Validators.TotalCount)
				}

				// Check groups - compare each group's nodes without caring about order
				if len(got.Groups) != len(tt.want.Groups) {
					t.Errorf("Groups count mismatch: got %d, want %d", len(got.Groups), len(tt.want.Groups))
				}
				for group, nodes := range tt.want.Groups {
					if gotNodes, exists := got.Groups[group]; exists {
						if !slicesContainSameElements(gotNodes, nodes) {
							t.Errorf("Group %s nodes mismatch:\ngot  %+v\nwant %+v", group, gotNodes, nodes)
						}
					} else {
						t.Errorf("Missing group: %s", group)
					}
				}

				// Check tags - compare each tag's nodes without caring about order
				if len(got.Tags) != len(tt.want.Tags) {
					t.Errorf("Tags count mismatch: got %d, want %d", len(got.Tags), len(tt.want.Tags))
				}
				for tag, nodes := range tt.want.Tags {
					if gotNodes, exists := got.Tags[tag]; exists {
						if !slicesContainSameElements(gotNodes, nodes) {
							t.Errorf("Tag %s nodes mismatch:\ngot  %+v\nwant %+v", tag, gotNodes, nodes)
						}
					} else {
						t.Errorf("Missing tag: %s", tag)
					}
				}

				// Check metadata
				if !reflect.DeepEqual(got.Metadata, tt.want.Metadata) {
					t.Errorf("Metadata mismatch:\ngot  %+v\nwant %+v", got.Metadata, tt.want.Metadata)
				}
			}
		})
	}
}
