package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// GeneratedCertificate command
type GenerateCertificate struct {
	RootCommand
	DNSName string
}

// NewGenerateCertificateCmd creates a generate certificate commmand
func NewGenerateCertificateCmd() *cobra.Command {
	c := GenerateCertificate{
		RootCommand: RootCommand{
			Command: cobra.Command{
				Use:   "generate-certificate",
				Short: "generates a self-signed localhost certificate",
				Long: `
generates an key and self-signed certificate which can be used to
serve over HTTPS.
`,
			},
		},
	}

	c.AddPersistentFlags()
	c.Flags().StringVarP(&c.DNSName, "dns-name", "", "localhost", "on the certificate")

	c.RunE = func(cmd *cobra.Command, args []string) error {
		return generateCertificate(c.KeyFile, c.CertificateFile, c.DNSName)
	}

	return &c.Command
}

func generateCertificate(keyFile, certificateFile, dnsName string) error {
	template := x509.Certificate{}
	template.Subject = pkix.Name{
		Organization: []string{"binx.io B.V."},
		Country:      []string{"NL"},
		CommonName:   dnsName,
	}

	template.NotBefore = time.Now()
	template.NotAfter = template.NotBefore.Add(10 * 365 * 10 * time.Hour)
	template.KeyUsage = x509.KeyUsageCertSign |
		x509.KeyUsageKeyEncipherment |
		x509.KeyUsageDigitalSignature |
		x509.KeyUsageCRLSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{
		x509.ExtKeyUsageClientAuth,
		x509.ExtKeyUsageServerAuth,
	}
	template.IsCA = true
	template.BasicConstraintsValid = true
	template.DNSNames = []string{dnsName}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %s", err)
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	template.SerialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %s", err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %s", err)
	}
	certOut, err := os.Create(certificateFile)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %s", certificateFile, err)
	}
	defer closeWithWarningOnError(certOut)

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing:%s", keyFile, err)
	}
	defer closeWithWarningOnError(keyOut)

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("failed to write certificate into %s: %s", certificateFile, err)
	}

	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err != nil {
		return fmt.Errorf("failed to write private key into %s: %s", keyFile, err)
	}

	return nil
}

func closeWithWarningOnError(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Printf("WARNING: failed to close file %s, %s", f.Name(), err)
	}
}
