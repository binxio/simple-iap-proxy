package gkeserver

import (
	"github.com/binxio/simple-iap-proxy/cmd"
	"github.com/spf13/cobra"
)

func NewGKEServerCmd() *cobra.Command {
	c := ReverseProxy{
		RootCommand: cmd.RootCommand{
			Command: cobra.Command{
				Use:   "gke-server",
				Short: "forwards requests from the load balancer to the appropriate GKE cluster",
				Long: `
reads the Host header of the http requests and if 
it matches the ip address of GKE cluster master endpoint, 
forwards the request to it.
`,
			},
		},
	}
	c.AddPersistentFlags()
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return c.Run()
	}

	return &c.Command
}
