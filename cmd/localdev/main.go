package main

import (
	"log"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "localdev",
		Short: "Utility for spinning up local development environments",
	}

	cmd.AddCommand(newKindCommand())

	return cmd
}
