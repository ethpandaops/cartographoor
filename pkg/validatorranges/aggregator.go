package validatorranges

import (
	"github.com/sirupsen/logrus"
)

// AggregateRanges combines multiple ValidatorRanges into a single consolidated result.
func AggregateRanges(ranges []*ValidatorRanges) *ValidatorRanges {
	if len(ranges) == 0 {
		return &ValidatorRanges{
			Nodes:      make(map[string]*Node),
			Validators: &ValidatorSummary{TotalCount: 0},
			Groups:     make(map[string][]string),
			Tags:       make(map[string][]string),
			Metadata:   &Metadata{Sources: []string{}},
		}
	}

	// Start with an empty result
	result := &ValidatorRanges{
		Nodes:      make(map[string]*Node),
		Validators: &ValidatorSummary{TotalCount: 0},
		Groups:     make(map[string][]string),
		Tags:       make(map[string][]string),
		Metadata:   &Metadata{Sources: []string{}},
	}

	// Aggregate all ranges
	for _, vr := range ranges {
		if vr == nil {
			continue
		}

		// Merge nodes
		mergeNodes(result.Nodes, vr.Nodes)

		// Aggregate sources
		if vr.Metadata != nil {
			result.Metadata.Sources = append(result.Metadata.Sources, vr.Metadata.Sources...)
		}
	}

	// Rebuild groups and tags from the final merged nodes (after duplicates have been skipped)
	result.Groups, result.Tags = buildGroupsAndTags(result.Nodes)

	// Deduplicate sources
	result.Metadata.Sources = deduplicateSlice(result.Metadata.Sources)

	// Recalculate totals
	recalculateTotals(result)

	return result
}

// mergeNodes adds source nodes into target nodes map, logging and skipping duplicates.
func mergeNodes(target, source map[string]*Node) {
	logger := logrus.WithField("module", "validator_ranges_aggregator")

	for name, node := range source {
		if existingNode, exists := target[name]; exists {
			// Log warning about duplicate node and skip it
			logger.WithFields(logrus.Fields{
				"node":             name,
				"existing_source":  existingNode.Source,
				"duplicate_source": node.Source,
			}).Warn("Duplicate node name found, skipping - this may indicate a configuration issue")

			continue
		}

		// Deep copy the node
		newNode := &Node{
			Groups:          make([]string, len(node.Groups)),
			Tags:            make([]string, len(node.Tags)),
			Attributes:      make(map[string]interface{}),
			ValidatorRanges: make([]*ValidatorRange, len(node.ValidatorRanges)),
			Source:          node.Source,
		}

		copy(newNode.Groups, node.Groups)
		copy(newNode.Tags, node.Tags)
		copy(newNode.ValidatorRanges, node.ValidatorRanges)

		for k, v := range node.Attributes {
			newNode.Attributes[k] = v
		}

		target[name] = newNode
	}
}

// recalculateTotals recalculates the total validator count.
func recalculateTotals(vr *ValidatorRanges) {
	total := 0

	for _, node := range vr.Nodes {
		for _, vRange := range node.ValidatorRanges {
			if vRange != nil {
				total += vRange.End - vRange.Start
			}
		}
	}

	vr.Validators.TotalCount = total
}

// deduplicateSlice removes duplicate strings from a slice.
func deduplicateSlice(slice []string) []string {
	seen := make(map[string]bool, len(slice))
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
