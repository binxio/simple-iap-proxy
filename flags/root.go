package flags

import (
	"github.com/spf13/cobra"
	"log"
	"os"
	"strconv"
)

// RootCommand the root command with all the global flags
type RootCommand struct {
	cobra.Command
	Debug           bool
	Port            int
	ProjectID       string
	KeyFile         string
	CertificateFile string
}

// AddPersistentFlags adds all the persistent flags to the command
func (c *RootCommand) AddPersistentFlags() {
	c.PersistentFlags().SortFlags = false
	c.PersistentFlags().BoolVarP(&c.Debug, "debug", "d", false, "provide debug information")
	c.PersistentFlags().IntVarP(&c.Port, "port", "P", getPort(), "port to listen on")
	c.PersistentFlags().StringVarP(&c.ProjectID, "project", "p", "", "google project id to use")
	c.PersistentFlags().StringVarP(&c.KeyFile, "key-file", "k", "", "key file for serving https")
	c.PersistentFlags().StringVarP(&c.CertificateFile, "certificate-file", "c", "", "certificate of the server")
	c.MarkPersistentFlagRequired("key-file")
	c.MarkPersistentFlagFilename("key-file")
	c.MarkPersistentFlagRequired("certificate-file")
	c.MarkPersistentFlagFilename("certificate-file")
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
