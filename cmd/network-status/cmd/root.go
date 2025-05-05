package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand(log *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network-status",
		Short: "A service that discovers active Ethereum networks in the ethpandaops ecosystem",
		Long:  `Network Status is a Go application that periodically scans and discovers active Ethereum networks maintained by the ethpandaops team.`,
	}

	// Add subcommands
	cmd.AddCommand(newRunCmd(log))

	return cmd
}