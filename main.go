package main

import (
	"log"

	"github.com/binxio/simple-iap-proxy/cmd"
	"github.com/binxio/simple-iap-proxy/gkeclient"
	"github.com/binxio/simple-iap-proxy/gkeserver"
	"github.com/spf13/cobra"
)

func newRootCmd() *cmd.RootCommand {
	c := cmd.RootCommand{
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
	c.AddCommand(cmd.NewGenerateCertificateCmd())
	c.AddCommand(gkeclient.NewGKEClientCmd())
	c.AddCommand(gkeserver.NewGKEServerCmd())
	return &c
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
