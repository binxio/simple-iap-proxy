package client

import (
	"log"

	"github.com/binxio/simple-iap-proxy/cmd"
	"github.com/spf13/cobra"
)

// NewClientCmd create a gke client command
func NewClientCmd() *cobra.Command {
	c := Proxy{
		RootCommand: cmd.RootCommand{
			Command: cobra.Command{
				Use:   "client",
				Short: "starts a client side proxy, forwarding requests via an IAP endpoint",
				Long: `The client will start a real HTTP/S proxy and forward any requests for
ip addresses of GKE cluster master endpoints or specified hostnames to the IAP proxy. 
Adds the required ID token as the Proxy-Authorization header in the request. Generates self-signed 
certificates for the targeted hosts on the fly.`,
			},
		},
	}
	c.AddPersistentFlags()
	c.Flags().StringVarP(&c.TargetURL, "target-url", "t", "", "to forward requests to")
	c.Flags().StringVarP(&c.Audience, "iap-audience", "a", "", "of the IAP application")
	c.Flags().StringVarP(&c.ServiceAccount, "service-account", "s", "", "to impersonate")
	c.Flags().BoolVarP(&c.UseDefaultCredentials, "use-default-credentials", "u", false, "use default credentials instead of gcloud configuration")
	c.Flags().StringVarP(&c.ConfigurationName, "configuration", "C", "", "name of gcloud configuration to use for credentials")
	c.Flags().BoolVarP(&c.ToGKEClusters, "to-gke", "G", false, "proxy to GKE clusters in the project")
	c.Flags().StringSliceVarP(&c.HostNames, "to-host", "H", []string{}, "proxy to these hosts, specified as regular expression")
	c.Flags().BoolVarP(&c.HTTPProtocol, "http-protocol", "", false, "proxy listens using HTTP instead of HTTPS")
	if err := c.MarkFlagRequired("iap-audience"); err != nil {
		log.Fatal(err)
	}
	c.MarkFlagRequired("service-account")
	c.MarkFlagRequired("target-url")
	c.Flags().SortFlags = false

	c.RunE = func(cmd *cobra.Command, args []string) error {
		return c.Run()
	}

	return &c.Command
}
