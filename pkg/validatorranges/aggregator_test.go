package validatorranges

import (
	"reflect"
	"testing"
)

func TestAggregateRanges(t *testing.T) {
	tests := []struct {
		name   string
		ranges []*ValidatorRanges
		want   *ValidatorRanges
	}{
		{
			name: "aggregate multiple sources",
			ranges: []*ValidatorRanges{
				{
					Nodes: map[string]*Node{
						"lighthouse-geth-1": {
							Groups: []string{"lighthouse_geth"},
							Tags:   []string{"el:geth", "cl:lighthouse"},
							Attributes: map[string]interface{}{
								"cloud": "aws",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 0, End: 8},
							},
							Source: "ethpandaops",
						},
						"prysm-besu-1": {
							Groups: []string{"prysm_besu"},
							Tags:   []string{"el:besu", "cl:prysm"},
							Attributes: map[string]interface{}{
								"region": "us-east-1",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 100, End: 108},
							},
							Source: "ethpandaops",
						},
					},
					Validators: &ValidatorSummary{TotalCount: 16},
					Groups: map[string][]string{
						"lighthouse_geth": {"lighthouse-geth-1"},
						"prysm_besu":      {"prysm-besu-1"},
					},
					Tags: map[string][]string{
						"el:geth":       {"lighthouse-geth-1"},
						"cl:lighthouse": {"lighthouse-geth-1"},
						"el:besu":       {"prysm-besu-1"},
						"cl:prysm":      {"prysm-besu-1"},
					},
					Metadata: &Metadata{
						Sources: []string{"https://ethpandaops.com/inventory.ini"},
					},
				},
				{
					Nodes: map[string]*Node{
						"lighthouse-besu-1": {
							Groups: []string{"lighthouse_besu"},
							Tags:   []string{"el:besu", "cl:lighthouse"},
							Attributes: map[string]interface{}{
								"cloud": "digitalocean",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 51312, End: 51320},
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
								{Start: 51400, End: 51408},
							},
							Source: "testinprod",
						},
					},
					Validators: &ValidatorSummary{TotalCount: 16},
					Groups: map[string][]string{
						"lighthouse_besu": {"lighthouse-besu-1"},
						"teku_geth":       {"teku-geth-1"},
					},
					Tags: map[string][]string{
						"el:besu":       {"lighthouse-besu-1"},
						"cl:lighthouse": {"lighthouse-besu-1"},
						"el:geth":       {"teku-geth-1"},
						"cl:teku":       {"teku-geth-1"},
					},
					Metadata: &Metadata{
						Sources: []string{"https://testinprod.com/inventory.ini"},
					},
				},
			},
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lighthouse-geth-1": {
						Groups: []string{"lighthouse_geth"},
						Tags:   []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud": "aws",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops",
					},
					"prysm-besu-1": {
						Groups: []string{"prysm_besu"},
						Tags:   []string{"el:besu", "cl:prysm"},
						Attributes: map[string]interface{}{
							"region": "us-east-1",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 100, End: 108},
						},
						Source: "ethpandaops",
					},
					"lighthouse-besu-1": {
						Groups: []string{"lighthouse_besu"},
						Tags:   []string{"el:besu", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud": "digitalocean",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 51312, End: 51320},
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
							{Start: 51400, End: 51408},
						},
						Source: "testinprod",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 32, // 8 + 8 + 8 + 8
				},
				Groups: map[string][]string{
					"lighthouse_geth": {"lighthouse-geth-1"},
					"prysm_besu":      {"prysm-besu-1"},
					"lighthouse_besu": {"lighthouse-besu-1"},
					"teku_geth":       {"teku-geth-1"},
				},
				Tags: map[string][]string{
					"el:geth":       {"lighthouse-geth-1", "teku-geth-1"},
					"cl:lighthouse": {"lighthouse-geth-1", "lighthouse-besu-1"},
					"el:besu":       {"prysm-besu-1", "lighthouse-besu-1"},
					"cl:prysm":      {"prysm-besu-1"},
					"cl:teku":       {"teku-geth-1"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://ethpandaops.com/inventory.ini", "https://testinprod.com/inventory.ini"},
				},
			},
		},
		{
			name: "duplicate nodes are skipped not merged",
			ranges: []*ValidatorRanges{
				{
					Nodes: map[string]*Node{
						"lighthouse-geth-1": {
							Groups: []string{"lighthouse_geth"},
							Tags:   []string{"el:geth", "cl:lighthouse"},
							Attributes: map[string]interface{}{
								"cloud": "aws",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 0, End: 8},
							},
							Source: "ethpandaops",
						},
					},
					Validators: &ValidatorSummary{TotalCount: 8},
					Groups: map[string][]string{
						"lighthouse_geth": {"lighthouse-geth-1"},
					},
					Tags: map[string][]string{
						"el:geth":       {"lighthouse-geth-1"},
						"cl:lighthouse": {"lighthouse-geth-1"},
					},
					Metadata: &Metadata{
						Sources: []string{"https://source1.com/inventory.ini"},
					},
				},
				{
					Nodes: map[string]*Node{
						"lighthouse-geth-1": {
							Groups: []string{"lighthouse_geth", "validators"},
							Tags:   []string{"el:geth", "cl:lighthouse", "vc:validator"},
							Attributes: map[string]interface{}{
								"cloud":  "digitalocean",
								"region": "nyc1",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 51312, End: 51320},
							},
							Source: "testinprod", // Different source - should be skipped
						},
						"lighthouse-geth-2": {
							Groups: []string{"lighthouse_geth"},
							Tags:   []string{"el:geth", "cl:lighthouse"},
							Attributes: map[string]interface{}{
								"cloud": "digitalocean",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 51320, End: 51328},
							},
							Source: "testinprod",
						},
					},
					Validators: &ValidatorSummary{TotalCount: 16},
					Groups: map[string][]string{
						"lighthouse_geth": {"lighthouse-geth-1", "lighthouse-geth-2"},
						"validators":      {"lighthouse-geth-1"},
					},
					Tags: map[string][]string{
						"el:geth":       {"lighthouse-geth-1", "lighthouse-geth-2"},
						"cl:lighthouse": {"lighthouse-geth-1", "lighthouse-geth-2"},
						"vc:validator":  {"lighthouse-geth-1"},
					},
					Metadata: &Metadata{
						Sources: []string{"https://source2.com/inventory.ini"},
					},
				},
			},
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"lighthouse-geth-1": {
						Groups: []string{"lighthouse_geth"},
						Tags:   []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud": "aws",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "ethpandaops", // First one wins, duplicate is skipped
					},
					"lighthouse-geth-2": {
						Groups: []string{"lighthouse_geth"},
						Tags:   []string{"el:geth", "cl:lighthouse"},
						Attributes: map[string]interface{}{
							"cloud": "digitalocean",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 51320, End: 51328},
						},
						Source: "testinprod",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 16, // 8 + 8 (duplicate lighthouse-geth-1 from testinprod is skipped)
				},
				Groups: map[string][]string{
					"lighthouse_geth": {"lighthouse-geth-1", "lighthouse-geth-2"},
					// Note: "validators" group is NOT included because the duplicate node with that group was skipped
				},
				Tags: map[string][]string{
					"el:geth":       {"lighthouse-geth-1", "lighthouse-geth-2"},
					"cl:lighthouse": {"lighthouse-geth-1", "lighthouse-geth-2"},
					// Note: "vc:validator" tag is NOT included because the duplicate node with that tag was skipped
				},
				Metadata: &Metadata{
					Sources: []string{"https://source1.com/inventory.ini", "https://source2.com/inventory.ini"},
				},
			},
		},
		{
			name:   "empty ranges",
			ranges: []*ValidatorRanges{},
			want: &ValidatorRanges{
				Nodes:      make(map[string]*Node),
				Validators: &ValidatorSummary{TotalCount: 0},
				Groups:     make(map[string][]string),
				Tags:       make(map[string][]string),
				Metadata:   &Metadata{Sources: []string{}},
			},
		},
		{
			name: "nil ranges in input",
			ranges: []*ValidatorRanges{
				{
					Nodes: map[string]*Node{
						"node1": {
							Groups: []string{"group1"},
							Tags:   []string{"tag1"},
							Attributes: map[string]interface{}{
								"attr": "value",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 0, End: 8},
							},
							Source: "source1",
						},
					},
					Validators: &ValidatorSummary{TotalCount: 8},
					Groups: map[string][]string{
						"group1": {"node1"},
					},
					Tags: map[string][]string{
						"tag1": {"node1"},
					},
					Metadata: &Metadata{
						Sources: []string{"https://source1.com"},
					},
				},
				nil, // nil entry should be handled gracefully
				{
					Nodes: map[string]*Node{
						"node2": {
							Groups: []string{"group2"},
							Tags:   []string{"tag2"},
							Attributes: map[string]interface{}{
								"attr2": "value2",
							},
							ValidatorRanges: []*ValidatorRange{
								{Start: 100, End: 108},
							},
							Source: "source2",
						},
					},
					Validators: &ValidatorSummary{TotalCount: 8},
					Groups: map[string][]string{
						"group2": {"node2"},
					},
					Tags: map[string][]string{
						"tag2": {"node2"},
					},
					Metadata: &Metadata{
						Sources: []string{"https://source2.com"},
					},
				},
			},
			want: &ValidatorRanges{
				Nodes: map[string]*Node{
					"node1": {
						Groups: []string{"group1"},
						Tags:   []string{"tag1"},
						Attributes: map[string]interface{}{
							"attr": "value",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 0, End: 8},
						},
						Source: "source1",
					},
					"node2": {
						Groups: []string{"group2"},
						Tags:   []string{"tag2"},
						Attributes: map[string]interface{}{
							"attr2": "value2",
						},
						ValidatorRanges: []*ValidatorRange{
							{Start: 100, End: 108},
						},
						Source: "source2",
					},
				},
				Validators: &ValidatorSummary{
					TotalCount: 16, // 8 + 8
				},
				Groups: map[string][]string{
					"group1": {"node1"},
					"group2": {"node2"},
				},
				Tags: map[string][]string{
					"tag1": {"node1"},
					"tag2": {"node2"},
				},
				Metadata: &Metadata{
					Sources: []string{"https://source1.com", "https://source2.com"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateRanges(tt.ranges)

			// Check nodes
			if !reflect.DeepEqual(got.Nodes, tt.want.Nodes) {
				for name, node := range got.Nodes {
					if wantNode, exists := tt.want.Nodes[name]; exists {
						if !reflect.DeepEqual(node, wantNode) {
							t.Errorf("Node %s mismatch:\ngot  %+v\nwant %+v", name, node, wantNode)
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

			// Check tags - tags may have different order in slices
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
			if !slicesContainSameElements(got.Metadata.Sources, tt.want.Metadata.Sources) {
				t.Errorf("Sources mismatch:\ngot  %+v\nwant %+v", got.Metadata.Sources, tt.want.Metadata.Sources)
			}
		})
	}
}

// slicesContainSameElements checks if two slices contain the same elements regardless of order.
func slicesContainSameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]int)
	for _, s := range a {
		aMap[s]++
	}

	bMap := make(map[string]int)
	for _, s := range b {
		bMap[s]++
	}

	return reflect.DeepEqual(aMap, bMap)
}
