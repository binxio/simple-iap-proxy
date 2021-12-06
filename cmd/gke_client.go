package cmd

import (
	"fmt"
	"github.com/binxio/simple-iap-proxy/gke_client"
	"github.com/spf13/cobra"
	"log"
	"net/url"
)

var (
	audience              string
	serviceAccount        string
	configurationName     string
	useDefaultCredentials bool
	targetURL             string
)

func validateClientArguments(cmd *cobra.Command, args []string) error {
	// mis-using the positional argument validator here.
	if useDefaultCredentials && configurationName != "" {
		return fmt.Errorf("specify either --use-default-credentials or --configuration, not both")
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target-url %s, %s", targetURL, err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("target-url must be https")
	}

	return validateRootArguments(cmd, args)
}

func init() {
	gkeClientCmd.Flags().StringVarP(&targetURL, "target-url", "t", "", "to forward requests to")
	gkeClientCmd.Flags().StringVarP(&audience, "iap-audience", "a", "", "of the IAP application")
	gkeClientCmd.Flags().StringVarP(&serviceAccount, "service-account", "s", "", "to impersonate")
	gkeClientCmd.Flags().BoolVarP(&useDefaultCredentials, "use-default-credentials", "u", false, "use default credentials instead of gcloud configuration")
	gkeClientCmd.Flags().StringVarP(&configurationName, "configuration", "C", "", "name of gcloud configuration to use for credentials")
	if err := gkeClientCmd.MarkFlagRequired("iap-audience"); err != nil {
		log.Fatal(err)
	}
	gkeClientCmd.MarkFlagRequired("service-account")
	gkeClientCmd.MarkFlagRequired("target-url")
	gkeClientCmd.Flags().SortFlags = false
}

var gkeClientCmd = &cobra.Command{
	Use:   "gke-client",
	Short: "starts a client side proxy, forwarding requests to the GKE cluster via the IAP",
	Long: `The client will start a real HTTP/S proxy and forward any requests for
ip address of GKE cluster master endpoints, to the IAP proxy.
`,
	Args: validateClientArguments,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		certificate, err = loadCertificate(keyFile)
		if err != nil {
			log.Fatal(err)
		}

		c := gke_client.Proxy{
			Debug:                 debug,
			Port:                  port,
			ServiceAccount:        serviceAccount,
			ProjectID:             projectID,
			UseDefaultCredentials: useDefaultCredentials,
			ConfigurationName:     configurationName,
			Audience:              audience,
			TargetURL:             targetURL,
			KeyFile:               keyFile,
			CertificateFile:       certificateFile,
			Certificate:           certificate,
		}
		c.Run()
	},
}
