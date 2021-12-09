package cmd

import (
	"github.com/binxio/simple-iap-proxy/flags"
	"github.com/spf13/cobra"
)

func newRootCmd() *flags.RootCommand {
	c := flags.RootCommand{
		Command: cobra.Command{
			Use:   "simple-iap-proxy",
			Short: "A simple proxy to forward requests over IAP to GKE",
			Long: `
This application allows you to gain access to GKE clusters
with a private master IP address via an IAP proxy. It consists of
a proxy which can be run on the client side, and a reverse-proxy which
is run inside the VPC.
`,
		},
	}
	c.AddPersistentFlags()
	c.AddCommand(newGKEClientCmd())
	c.AddCommand(newGKEServerCmd())
	return &c
}

// Execute the root command
func Execute() error {
	return newRootCmd().Execute()
}
