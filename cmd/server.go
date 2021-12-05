package cmd

import (
	"github.com/binxio/simple-iap-proxy/server"
	"github.com/spf13/cobra"
)

func init() {
	serverCmd.MarkFlagRequired("key-file")
	serverCmd.MarkFlagRequired("certificate-file")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "forwards requests from the load balancer to the appropriate GKE cluster",
	Long: `reads the Host header of the http requests and if
  it matches the ip address of GKE cluster master endpoint, forwards the request to it.
`,
	TraverseChildren: true,
	Args:             validateRootArguments,
	Run: func(cmd *cobra.Command, args []string) {
		s := server.ReverseProxy{
			Debug:           debug,
			Port:            port,
			ProjectID:       projectID,
			KeyFile:         keyFile,
			CertificateFile: certificateFile,
		}
		s.Run()
	},
}
