package gkeserver

import (
	"github.com/binxio/simple-iap-proxy/cmd"
	"github.com/spf13/cobra"
)

// NewGKEServerCmd create a gke server command
func NewGKEServerCmd() *cobra.Command {
	c := ReverseProxy{
		RootCommand: cmd.RootCommand{
			Command: cobra.Command{
				Use:   "gke-server",
				Short: "forward requests to GKE clusters",
				Long: `
Reads the Host header of the http requests. If it matches the ip address of a GKE cluster master endpoint,
forwards the request to it. Reject requests for any other endpoint.
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
