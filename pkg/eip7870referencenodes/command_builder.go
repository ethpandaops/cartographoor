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
func (b *CommandBuilder) BuildCommand(
	client string,
	baseArgs []string,
	clientOverlay *ClientOverlay,
	featureOverlay *FeatureOverlay,
) *ClientCommand {
	cmd := &ClientCommand{
		Client:      client,
		DisplayName: ClientDisplayNames[client],
		ArgsBreakdown: ArgsBreakdown{
			Base:           baseArgs,
			ClientSpecific: make([]string, 0),
			Feature7870:    make([]string, 0),
		},
		EnvVars: make(map[string]string),
	}

	// Set client-specific args
	if clientOverlay != nil {
		cmd.ArgsBreakdown.ClientSpecific = clientOverlay.Args
		cmd.Image = clientOverlay.Image
	}

	// Set feature args and env vars
	if featureOverlay != nil {
		cmd.ArgsBreakdown.Feature7870 = featureOverlay.Args

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

	// Add notes about Keel auto-updates if applicable
	if len(cmd.EnvVars) > 0 {
		cmd.Notes = "Image auto-updates via Keel every 60 minutes. OTLP tracing enabled for EIP-7870."
	} else {
		cmd.Notes = "Image auto-updates via Keel every 60 minutes"
	}

	// Remove empty env vars map
	if len(cmd.EnvVars) == 0 {
		cmd.EnvVars = nil
	}

	return cmd
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
