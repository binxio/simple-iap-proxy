package main

import (
	"log"

	"github.com/binxio/simple-iap-proxy/client"
	"github.com/binxio/simple-iap-proxy/cmd"
	"github.com/binxio/simple-iap-proxy/gkeserver"
	"github.com/spf13/cobra"
)

func newRootCmd() *cmd.RootCommand {
	c := cmd.RootCommand{
		Command: cobra.Command{
			Use:   "simple-iap-proxy",
			Short: "A simple proxy to forward requests over the Google Identity Aware Proxy",
			Long: `
This is a simple IAP proxy. It will intercept the required HTTPS request and 
inject the IAP proxy authorization header.
`,
		},
	}
	c.AddPersistentFlags()
	c.AddCommand(cmd.NewGenerateCertificateCmd())
	c.AddCommand(client.NewClientCmd())
	c.AddCommand(gkeserver.NewGKEServerCmd())
	return &c
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
