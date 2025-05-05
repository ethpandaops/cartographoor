package main

import (
	"os"

	"github.com/ethpandaops/network-status/cmd/network-status/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	if err := cmd.NewRootCommand(log).Execute(); err != nil {
		log.WithError(err).Fatal("Failed to execute command")
		os.Exit(1)
	}
}