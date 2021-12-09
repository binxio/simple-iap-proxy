package cmd

import (
	"log"

	"github.com/binxio/simple-iap-proxy/flags"
	"github.com/binxio/simple-iap-proxy/gkeclient"
	"github.com/spf13/cobra"
)

func newGKEClientCmd() *cobra.Command {
	c := gkeclient.Proxy{
		RootCommand: flags.RootCommand{
			Command: cobra.Command{
				Use:   "gke-client",
				Short: "starts a client side proxy, forwarding requests to the GKE cluster via the IAP",
				Long: `The client will start a real HTTP/S proxy and forward any requests for,
ip address of GKE cluster master endpoints, to the IAP proxy.`,
			},
		},
	}
	c.AddPersistentFlags()
	c.Flags().StringVarP(&c.TargetURL, "target-url", "t", "", "to forward requests to")
	c.Flags().StringVarP(&c.Audience, "iap-audience", "a", "", "of the IAP application")
	c.Flags().StringVarP(&c.ServiceAccount, "service-account", "s", "", "to impersonate")
	c.Flags().BoolVarP(&c.UseDefaultCredentials, "use-default-credentials", "u", false, "use default credentials instead of gcloud configuration")
	c.Flags().StringVarP(&c.ConfigurationName, "configuration", "C", "", "name of gcloud configuration to use for credentials")
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
