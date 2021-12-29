package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

func loadCertificate(keyFile, certificateFile string) (*tls.Certificate, error) {
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
