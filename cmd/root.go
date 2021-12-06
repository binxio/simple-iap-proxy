package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strconv"
)

var (
	debug           bool
	projectID       string
	port            int
	keyFile         string
	certificateFile string
	certificate     *tls.Certificate

	rootCmd = &cobra.Command{
		Use:   "simple-iap-proxy",
		Short: "A simple proxy to forward requests over IAP to GKE",
		Long: `This application allows you to gain access to GKE clusters
with a private master IP address via an IAP proxy. It consists of
a proxy which can be run on the client side, and a reverse-proxy which
is run inside the VPC.
`,

		Args: validateRootArguments,
	}
)

// Execute the main command
func Execute() error {
	return rootCmd.Execute()
}

func getPort() int {
	listenPort := os.Getenv("PORT")
	if listenPort == "" {
		return 8080
	}
	port, err := strconv.ParseUint(listenPort, 10, 64)
	if err != nil || port > 65535 {
		log.Fatalf("the environment variable PORT is not a valid port number")
	}
	return int(port)
}

func loadCertificate(keyFile string) (*tls.Certificate, error) {

	caKey, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("%s, %s", keyFile, err)
	}

	caCert, err := os.ReadFile(certificateFile)
	if err != nil {
		return nil, fmt.Errorf("%s, %s", certificateFile, err)
	}

	cert, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate, %s", err)
	}
	if cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
		return nil, fmt.Errorf("failed to parse certificate, %s", err)
	}

	return &cert, nil
}

func validateRootArguments(_ *cobra.Command, _ []string) error {
	if _, err := loadCertificate(keyFile); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "provide debug information")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "P", getPort(), "port to listen on")
	rootCmd.PersistentFlags().StringVarP(&projectID, "project", "p", "", "google project id to use")
	rootCmd.PersistentFlags().StringVarP(&keyFile, "key-file", "k", "", "key file for serving https")
	rootCmd.PersistentFlags().StringVarP(&certificateFile, "certificate-file", "c", "", "certificate of the server")
	rootCmd.MarkFlagFilename("key-file")
	rootCmd.MarkFlagFilename("certificate-file")
	rootCmd.MarkFlagRequired("key-file")
	rootCmd.MarkFlagRequired("certificate-file")
	rootCmd.PersistentFlags().SortFlags = false

	rootCmd.AddCommand(gkeClientCmd)
	rootCmd.AddCommand(gkeServerCmd)
}
