// Package ca provides easy to use certificate authority related functions.
// This is a lightweight wrapper around "crypto/x509" package for
// creating CA certs, client certs, signing requests, and more.
//
// This package is mostly based on the example code provided at:
// http://golang.org/src/crypto/tls/generate_cert.go
package ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// GenCA generates a self-signed CA certificate.
//
// validFor = Validity period for the certificate.
// keyLength = Key length for the new certificate.
// cn, org = Common name and organization fields of the certificate.
//
// Returns PEM encoded X.509 certificate and private key pair.
//
// Note: While writing the binary cert/key pair to file system, it is useful to use standard naming like: 'cert.pem', 'key.pem'.
func GenCA(subject pkix.Name, validFor time.Duration, keyLength int, cn, org string) (cert, key []byte, err error) {
	c, p, err := getBaseCert(subject, validFor, keyLength, cn, org)
	c.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	c.BasicConstraintsValid = true
	c.IsCA = true

	cert, key, err = signAndEncodeCert(c, p, c, p)
	return
}

// GenSigningCert generates an intermediate signing certificate for signing server or client certificates.
// Returns PEM encoded X.509 certificate and private key pair.
func GenSigningCert(subject pkix.Name, validFor time.Duration, keyLength int, cn, org string) (cert, key []byte, err error) {
	c, p, err := getBaseCert(subject, validFor, keyLength, cn, org)
	c.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign

	cert, key, err = signAndEncodeCert(c, p, c, p)
	return
}

// GenServerCert generates a hosting certificate for servers using TLS.
// Returns PEM encoded X.509 certificate and private key pair.
func GenServerCert(subject pkix.Name, host string, validFor time.Duration, keyLength int, cn, org string) (cert, key []byte, err error) {
	c, p, err := getBaseCert(subject, validFor, keyLength, cn, org)
	c.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	setHosts(host, c)

	cert, key, err = signAndEncodeCert(c, p, c, p)
	return
}

// GenClientCert generates a client certificate signed by the provided signing certificate.
// Generated certificate will have its extended key usage set to 'client authentication' and will be ready for use in TLS client authentication.
// Returns PEM encoded X.509 certificate and private key pair.
func GenClientCert(subject pkix.Name, validFor time.Duration, keyLength int, cn, org string) (cert, key []byte, err error) {
	c, p, err := getBaseCert(subject, validFor, keyLength, cn, org)
	c.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	c.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	cert, key, err = signAndEncodeCert(c, p, c, p)
	return
}

// getBaseCert creates and returns x509.Certificate (unsigned) and rsa.PrivateKey objects with basic paramters set.
func getBaseCert(subject pkix.Name, validFor time.Duration, keyLength int, cn, org string) (*x509.Certificate, *rsa.PrivateKey, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate certificate private key using RSA: %v", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate the certificate serial number: %v", err)
	}

	cert := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	return &cert, privKey, nil
}

// setHosts parses the comma separated host name / IP list and adds them to Subject Alternate Name list of a server/hosting certificate.
func setHosts(host string, cert *x509.Certificate) {
	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, h)
		}
	}
}

// signAndEncodeCert signs a given certificate with given signing cert/key pair and encodes resulting signed cert and private key in PEM format and returns.
func signAndEncodeCert(signingCert *x509.Certificate, signingKey *rsa.PrivateKey, c *x509.Certificate, k *rsa.PrivateKey) (cert, key []byte, err error) {
	certDerBytes, err := x509.CreateCertificate(rand.Reader, c, signingCert, &k.PublicKey, signingKey)
	if err != nil {
		return
	}

	cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDerBytes})
	key = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
	return
}
