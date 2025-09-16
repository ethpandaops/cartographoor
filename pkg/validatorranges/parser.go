package validatorranges

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

// ParseInventory parses an Ansible inventory INI file and extracts validator ranges.
func ParseInventory(content []byte, sourceURL string, sourceName string, rangeOffset int) (*ValidatorRanges, error) {
	// Pre-process content to remove standalone host entries like "localhost"
	// which are valid Ansible but not valid INI format
	lines := strings.Split(string(content), "\n")
	processedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip standalone host entries (lines without '=' that aren't sections)
		if trimmed != "" && !strings.HasPrefix(trimmed, "[") && !strings.Contains(trimmed, "=") {
			continue
		}

		processedLines = append(processedLines, line)
	}

	processedContent := strings.Join(processedLines, "\n")

	cfg, err := ini.Load([]byte(processedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI: %w", err)
	}

	nodes := make(map[string]*Node)

	// Parse all sections except DEFAULT
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		parseSection(section, nodes, sourceName, rangeOffset)
	}

	// Filter out nodes without validator ranges
	nodesWithValidators := make(map[string]*Node)

	for name, node := range nodes {
		if len(node.ValidatorRanges) > 0 {
			nodesWithValidators[name] = node
		}
	}

	// Build groups and tags from nodes with validators only
	groups, tags := buildGroupsAndTags(nodesWithValidators)

	// Calculate total validators
	totalValidators := calculateTotalValidators(nodesWithValidators)

	return &ValidatorRanges{
		Nodes: nodesWithValidators,
		Validators: &ValidatorSummary{
			TotalCount: totalValidators,
		},
		Groups: groups,
		Tags:   tags,
		Metadata: &Metadata{
			Sources: []string{sourceURL},
		},
	}, nil
}

// parseSection processes a single INI section and updates the nodes map.
func parseSection(section *ini.Section, nodes map[string]*Node, sourceName string, rangeOffset int) {
	sectionName := section.Name()

	// Extract group and tags from section name
	groupName := sectionName
	sectionTags := extractTags([]string{groupName})

	// Parse each key in the section
	for _, key := range section.Keys() {
		// Extract just the hostname (first part before space)
		keyName := key.Name()
		nodeName := keyName

		if idx := strings.Index(keyName, " "); idx > 0 {
			nodeName = keyName[:idx]
		}

		// Get or create the node
		node, exists := nodes[nodeName]
		if !exists {
			node = &Node{
				Groups:     []string{},
				Tags:       []string{},
				Attributes: make(map[string]interface{}),
				Source:     sourceName,
			}
			nodes[nodeName] = node
		}

		// Add group to node
		if !contains(node.Groups, groupName) {
			node.Groups = append(node.Groups, groupName)
		}

		// Add tags to node
		for _, tag := range sectionTags {
			if !contains(node.Tags, tag) {
				node.Tags = append(node.Tags, tag)
			}
		}

		// Extract validator range from key value
		if key.Value() != "" {
			validatorRange := extractValidatorRange(key, rangeOffset)
			if validatorRange != nil {
				// Initialize array if needed
				if node.ValidatorRanges == nil {
					node.ValidatorRanges = []*ValidatorRange{}
				}

				node.ValidatorRanges = append(node.ValidatorRanges, validatorRange)
			}

			// Store all attributes with camelCase keys (excluding validator_start/end)
			for _, attr := range strings.Fields(key.Value()) {
				if strings.Contains(attr, "=") {
					parts := strings.SplitN(attr, "=", 2)
					if len(parts) == 2 {
						// Skip validator_start and validator_end as they're in validatorRange
						if parts[0] == "validator_start" || parts[0] == "validator_end" {
							continue
						}

						camelKey := snakeToCamel(parts[0])

						// Special case: rename long attribute names
						if camelKey == "ethereumNodeClSupernodeEnabled" {
							camelKey = "isClSupernode"
						}

						// Parse boolean values
						value := parts[1]
						switch strings.ToLower(value) {
						case "true":
							node.Attributes[camelKey] = true
						case "false":
							node.Attributes[camelKey] = false
						default:
							node.Attributes[camelKey] = value
						}
					}
				}
			}
		}
	}
}

// extractValidatorRange extracts validator start and end from a key value.
func extractValidatorRange(key *ini.Key, rangeOffset int) *ValidatorRange {
	value := key.Value()

	var start, end int

	var hasStart, hasEnd bool

	// Parse attributes in the value
	for _, attr := range strings.Fields(value) {
		if strings.HasPrefix(attr, "validator_start=") {
			parts := strings.Split(attr, "=")
			if len(parts) == 2 {
				if val, err := strconv.Atoi(parts[1]); err == nil {
					start = val + rangeOffset
					hasStart = true
				}
			}
		} else if strings.HasPrefix(attr, "validator_end=") {
			parts := strings.Split(attr, "=")
			if len(parts) == 2 {
				if val, err := strconv.Atoi(parts[1]); err == nil {
					end = val + rangeOffset
					hasEnd = true
				}
			}
		}
	}

	if hasStart && hasEnd && end >= start {
		return &ValidatorRange{
			Start: start,
			End:   end,
		}
	}

	return nil
}

// extractTags extracts tags from group names.
func extractTags(groups []string) []string {
	tags := []string{}

	for _, group := range groups {
		// Check for execution layer tags
		if strings.Contains(group, "besu") {
			tags = append(tags, "el:besu")
		}

		if strings.Contains(group, "geth") {
			tags = append(tags, "el:geth")
		}

		if strings.Contains(group, "nethermind") {
			tags = append(tags, "el:nethermind")
		}

		if strings.Contains(group, "erigon") {
			tags = append(tags, "el:erigon")
		}

		if strings.Contains(group, "reth") {
			tags = append(tags, "el:reth")
		}

		// Check for consensus layer tags
		if strings.Contains(group, "lighthouse") {
			tags = append(tags, "cl:lighthouse")
		}

		if strings.Contains(group, "prysm") {
			tags = append(tags, "cl:prysm")
		}

		if strings.Contains(group, "teku") {
			tags = append(tags, "cl:teku")
		}

		if strings.Contains(group, "nimbus") {
			tags = append(tags, "cl:nimbus")
		}

		if strings.Contains(group, "lodestar") {
			tags = append(tags, "cl:lodestar")
		}

		if strings.Contains(group, "grandine") {
			tags = append(tags, "cl:grandine")
		}

		// Check for validator client tags
		if strings.Contains(group, "validator") {
			tags = append(tags, "vc:validator")
		}
	}

	return tags
}

// buildGroupsAndTags builds the groups and tags maps from nodes.
func buildGroupsAndTags(nodes map[string]*Node) (map[string][]string, map[string][]string) {
	groups := make(map[string][]string)
	tags := make(map[string][]string)

	for nodeName, node := range nodes {
		// Build groups map
		for _, group := range node.Groups {
			if _, exists := groups[group]; !exists {
				groups[group] = []string{}
			}

			groups[group] = append(groups[group], nodeName)
		}

		// Build tags map
		for _, tag := range node.Tags {
			if _, exists := tags[tag]; !exists {
				tags[tag] = []string{}
			}

			tags[tag] = append(tags[tag], nodeName)
		}
	}

	return groups, tags
}

// calculateTotalValidators calculates the total number of validators across all nodes.
func calculateTotalValidators(nodes map[string]*Node) int {
	total := 0

	for _, node := range nodes {
		for _, vRange := range node.ValidatorRanges {
			if vRange != nil {
				total += vRange.End - vRange.Start
			}
		}
	}

	return total
}

// contains checks if a string slice contains a specific string.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}

// snakeToCamel converts snake_case to camelCase.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return result
}
