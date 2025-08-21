package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand(log *logrus.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cartographoor",
		Short: "A service that discovers active Ethereum networks in the ethpandaops ecosystem",
		Long:  `Cartographoor is a Go application that periodically scans and discovers active Ethereum networks maintained by the ethpandaops team.`,
	}

	// Add subcommands.
	cmd.AddCommand(newRunCmd(log))
	cmd.AddCommand(newInventoryCmd(log))

	return cmd
}
