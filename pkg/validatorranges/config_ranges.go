package validatorranges

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

const (
	configRangesPath = "/api/v1/nodes/validator-ranges"
	configSourceName = "config"
)

// configRangesURL returns the validator ranges endpoint a network's config service publishes, or "" if it has none.
func configRangesURL(network discovery.Network) string {
	if network.GenesisConfig == nil {
		return ""
	}

	for _, file := range network.GenesisConfig.API {
		if file.Path == configRangesPath && file.URL != "" {
			return file.URL
		}
	}

	return ""
}

// ParseConfigRanges parses a config service validator ranges document into exclusive-end ranges per name.
// Its "start-end" keys are inclusive on-chain validator indices.
func ParseConfigRanges(content []byte) (map[string][]*ValidatorRange, error) {
	var doc struct {
		Ranges map[string]string `json:"ranges"`
	}

	if err := json.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config ranges: %w", err)
	}

	if len(doc.Ranges) == 0 {
		return nil, fmt.Errorf("config ranges document has no ranges")
	}

	ranges := make(map[string][]*ValidatorRange)

	for key, name := range doc.Ranges {
		startText, endText, ok := strings.Cut(key, "-")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid config range %q: %q", key, name)
		}

		start, startErr := strconv.Atoi(startText)
		end, endErr := strconv.Atoi(endText)

		if startErr != nil || endErr != nil || start < 0 || end < start {
			return nil, fmt.Errorf("invalid config range %q", key)
		}

		ranges[name] = append(ranges[name], &ValidatorRange{Start: start, End: end + 1})
	}

	for _, nodeRanges := range ranges {
		slices.SortFunc(nodeRanges, func(a, b *ValidatorRange) int {
			return cmp.Compare(a.Start, b.Start)
		})
	}

	return ranges, nil
}

// ApplyConfigRanges makes the config service authoritative for every validator index it covers. Matching nodes keep
// their tags but take the config ranges, names only the config knows become untagged nodes, and inventory-only ranges
// survive unless they overlap a config range.
func ApplyConfigRanges(vr *ValidatorRanges, ranges map[string][]*ValidatorRange, sourceURL string, logger logrus.FieldLogger) {
	covered := make([]*ValidatorRange, 0, len(ranges))
	for _, nodeRanges := range ranges {
		covered = append(covered, nodeRanges...)
	}

	for name, node := range vr.Nodes {
		if configRanges, ok := ranges[name]; ok {
			if !slices.EqualFunc(sortedRanges(node.ValidatorRanges), configRanges, equalRange) {
				logger.WithField("node", name).Warn("Inventory validator ranges differ from the config service, using the config service")
			}

			node.ValidatorRanges = cloneRanges(configRanges)

			continue
		}

		kept := make([]*ValidatorRange, 0, len(node.ValidatorRanges))

		for _, r := range node.ValidatorRanges {
			if slices.ContainsFunc(covered, func(c *ValidatorRange) bool { return r.Start < c.End && c.Start < r.End }) {
				logger.WithFields(logrus.Fields{
					"node":  name,
					"start": r.Start,
					"end":   r.End,
				}).Warn("Dropping inventory validator range that the config service assigns elsewhere")

				continue
			}

			kept = append(kept, r)
		}

		if len(kept) == 0 {
			delete(vr.Nodes, name)

			continue
		}

		node.ValidatorRanges = kept
	}

	for name, configRanges := range ranges {
		if _, ok := vr.Nodes[name]; ok {
			continue
		}

		vr.Nodes[name] = &Node{
			Groups:          []string{},
			Tags:            []string{},
			Attributes:      make(map[string]any),
			ValidatorRanges: cloneRanges(configRanges),
			Source:          configSourceName,
		}
	}

	vr.Groups, vr.Tags = buildGroupsAndTags(vr.Nodes)
	vr.Metadata.Sources = deduplicateSlice(append(vr.Metadata.Sources, sourceURL))

	recalculateTotals(vr)
}

func sortedRanges(ranges []*ValidatorRange) []*ValidatorRange {
	sorted := slices.Clone(ranges)

	slices.SortFunc(sorted, func(a, b *ValidatorRange) int {
		return cmp.Compare(a.Start, b.Start)
	})

	return sorted
}

func cloneRanges(ranges []*ValidatorRange) []*ValidatorRange {
	cloned := make([]*ValidatorRange, 0, len(ranges))
	for _, r := range ranges {
		cloned = append(cloned, &ValidatorRange{Start: r.Start, End: r.End})
	}

	return cloned
}

func equalRange(a, b *ValidatorRange) bool {
	return a.Start == b.Start && a.End == b.End
}
