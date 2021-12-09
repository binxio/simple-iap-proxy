package cmd

import (
	"fmt"
	"github.com/binxio/simple-iap-proxy/flags"
	"github.com/binxio/simple-iap-proxy/gkeserver"
	"github.com/spf13/cobra"
)

func newGKEServerCmd() *cobra.Command {
	gkeServerCmd := gkeserver.ReverseProxy{
		RootCommand: flags.RootCommand{
			Command: cobra.Command{
				Use:   "gke-server",
				Short: "forwards requests from the load balancer to the appropriate GKE cluster",
				Long: `
reads the Host header of the http requests and if 
it matches the ip address of GKE cluster master endpoint, 
forwards the request to it.
`,
				RunE: runReverseProxy,
			},
		},
	}

	return &gkeServerCmd.Command
}

func runReverseProxy(c *cobra.Command, _ []string) error {
	s, ok := interface{}(c).(gkeserver.ReverseProxy)
	if !ok {
		return fmt.Errorf("command is not a reverse proxy command")
	}
	return s.Run()
}
