package eip7870referencenodes

import (
	"strings"
)

// CommandBuilder builds the final startup command from multiple sources.
type CommandBuilder struct{}

// NewCommandBuilder creates a new CommandBuilder.
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{}
}

// BuildCommand merges base args, client-specific args, and feature args into a complete command.
// Later sources (clientSpecific, feature) override earlier sources (base).
// Duplicate args are deduplicated, with later values taking precedence.
func (b *CommandBuilder) BuildCommand(
	client string,
	baseArgs []string,
	clientOverlay *ClientOverlay,
	featureOverlay *FeatureOverlay,
) *ClientCommand {
	var clientArgs, featureArgs []string

	if clientOverlay != nil {
		clientArgs = clientOverlay.Args
	}

	if featureOverlay != nil {
		featureArgs = featureOverlay.Args
	}

	// Deduplicate args: later sources override earlier ones
	// Order of precedence: feature > clientSpecific > base
	deduped := b.deduplicateArgs(baseArgs, clientArgs, featureArgs)

	cmd := &ClientCommand{
		Client:      client,
		DisplayName: ClientDisplayNames[client],
		ArgsBreakdown: ArgsBreakdown{
			Base:           deduped.base,
			ClientSpecific: deduped.clientSpecific,
			Feature7870:    deduped.feature,
		},
		EnvVars: make(map[string]string),
	}

	// Set image from client overlay
	if clientOverlay != nil {
		cmd.Image = clientOverlay.Image
	}

	// Set env vars and image override from feature overlay
	if featureOverlay != nil {
		for k, v := range featureOverlay.EnvVars {
			cmd.EnvVars[k] = v
		}

		// Feature can override image
		if featureOverlay.Image != nil {
			if featureOverlay.Image.Repository != "" {
				cmd.Image.Repository = featureOverlay.Image.Repository
			}

			if featureOverlay.Image.Tag != "" {
				cmd.Image.Tag = featureOverlay.Image.Tag
			}
		}
	}

	// Build the complete startup command string
	cmd.StartupCommand = b.buildCommandString(client, cmd.ArgsBreakdown)

	// Remove empty env vars map
	if len(cmd.EnvVars) == 0 {
		cmd.EnvVars = nil
	}

	return cmd
}

// dedupedArgs holds the deduplicated args for each source.
type dedupedArgs struct {
	base           []string
	clientSpecific []string
	feature        []string
}

// deduplicateArgs removes duplicate args across sources.
// Later sources override earlier ones. Args are identified by their flag name.
func (b *CommandBuilder) deduplicateArgs(base, clientSpecific, feature []string) dedupedArgs {
	// Build a set of flag names from higher-priority sources
	clientFlags := make(map[string]bool, len(clientSpecific))
	for _, arg := range clientSpecific {
		clientFlags[extractFlagName(arg)] = true
	}

	featureFlags := make(map[string]bool, len(feature))
	for _, arg := range feature {
		featureFlags[extractFlagName(arg)] = true
	}

	// Filter base args: remove any that are overridden by clientSpecific or feature
	filteredBase := make([]string, 0, len(base))
	seenBaseFlags := make(map[string]bool, len(base))

	for _, arg := range base {
		flagName := extractFlagName(arg)

		// Skip if overridden by higher priority source
		if clientFlags[flagName] || featureFlags[flagName] {
			continue
		}

		// Skip duplicates within base
		if seenBaseFlags[flagName] {
			continue
		}

		seenBaseFlags[flagName] = true

		filteredBase = append(filteredBase, arg)
	}

	// Filter clientSpecific args: remove any that are overridden by feature, and deduplicate
	filteredClient := make([]string, 0, len(clientSpecific))
	seenClientFlags := make(map[string]bool, len(clientSpecific))

	for _, arg := range clientSpecific {
		flagName := extractFlagName(arg)

		// Skip if overridden by feature
		if featureFlags[flagName] {
			continue
		}

		// Skip duplicates within clientSpecific
		if seenClientFlags[flagName] {
			continue
		}

		seenClientFlags[flagName] = true

		filteredClient = append(filteredClient, arg)
	}

	// Deduplicate feature args
	filteredFeature := make([]string, 0, len(feature))
	seenFeatureFlags := make(map[string]bool, len(feature))

	for _, arg := range feature {
		flagName := extractFlagName(arg)

		if seenFeatureFlags[flagName] {
			continue
		}

		seenFeatureFlags[flagName] = true

		filteredFeature = append(filteredFeature, arg)
	}

	return dedupedArgs{
		base:           filteredBase,
		clientSpecific: filteredClient,
		feature:        filteredFeature,
	}
}

// extractFlagName extracts the flag name from an argument.
// e.g., "--http" -> "--http", "--http.port=8545" -> "--http.port", "--http=false" -> "--http".
func extractFlagName(arg string) string {
	// Find the position of '=' if present
	if idx := strings.Index(arg, "="); idx != -1 {
		return arg[:idx]
	}

	return arg
}

// buildCommandString creates the complete command string from args breakdown.
func (b *CommandBuilder) buildCommandString(client string, args ArgsBreakdown) string {
	// Preallocate with capacity: 1 (client) + base + client-specific + feature args
	capacity := 1 + len(args.Base) + len(args.ClientSpecific) + len(args.Feature7870)
	allArgs := make([]string, 0, capacity)

	// Start with the client binary
	allArgs = append(allArgs, client)

	// Add base args
	allArgs = append(allArgs, args.Base...)

	// Add client-specific args
	allArgs = append(allArgs, args.ClientSpecific...)

	// Add feature args
	allArgs = append(allArgs, args.Feature7870...)

	return strings.Join(allArgs, " ")
}
