package cmd

import (
	"fmt"
	"github.com/binxio/simple-iap-proxy/client"
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

	if u, err := url.Parse(targetURL); err != nil {
		return fmt.Errorf("invalid target-url %s, %s", targetURL, err)
	} else {
		if u.Scheme != "https" {
			return fmt.Errorf("target-url must be https")
		}
	}

	return validateRootArguments(cmd, args)
}

func init() {
	clientCmd.Flags().StringVarP(&targetURL, "target-url", "t", "", "to forward requests to")
	clientCmd.Flags().StringVarP(&audience, "iap-audience", "a", "", "of the IAP application")
	clientCmd.Flags().StringVarP(&serviceAccount, "service-account", "s", "", "to impersonate")
	clientCmd.Flags().BoolVarP(&useDefaultCredentials, "use-default-credentials", "u", false, "use default credentials instead of gcloud configuration")
	clientCmd.Flags().StringVarP(&configurationName, "configuration", "C", "", "name of gcloud configuration to use for credentials")
	clientCmd.MarkFlagRequired("iap-audience")
	clientCmd.MarkFlagRequired("service-account")
	clientCmd.MarkFlagRequired("target-url")
	clientCmd.Flags().SortFlags = false
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "starts a client side proxy, forwarding requests to the GKE cluster via the IAP",
	Long: `The client will start a real HTTP/S proxy and forward any requests for
ip address of GKE cluster master endpoints, to the IAP proxy.
`,
	Args: validateClientArguments,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if keyFile != "" {
			certificate, err = loadCertificate(keyFile)
			if err != nil {
				log.Fatal(err)
			}
		}
		c := client.Proxy{
			Debug:                 debug,
			Port:                  port,
			ServiceAccount:        serviceAccount,
			ProjectId:             projectID,
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
