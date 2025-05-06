package main

import (
	"os"

	"github.com/ethpandaops/cartographoor/cmd/cartographoor/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetOutput(os.Stdout)

	// Log level will be properly set by the configuration in cmd.NewRootCommand
	// Default to info level
	log.SetLevel(logrus.InfoLevel)

	if err := cmd.NewRootCommand(log).Execute(); err != nil {
		log.WithError(err).Fatal("Failed to execute command")
		os.Exit(1)
	}
}
