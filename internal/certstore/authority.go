package certstore

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	caCertFile = "laohukuaipao-root-ca.pem"
	caKeyFile  = "laohukuaipao-root-ca-key.pem"
)

type Authority struct {
	mu       sync.Mutex
	dir      string
	certPath string
	keyPath  string
	certPEM  []byte
	caCert   *x509.Certificate
	caKey    crypto.Signer
	cache    map[string]*tls.Certificate
}

func LoadOrCreate(dir string) (*Authority, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	a := &Authority{
		dir:      dir,
		certPath: filepath.Join(dir, caCertFile),
		keyPath:  filepath.Join(dir, caKeyFile),
		cache:    map[string]*tls.Certificate{},
	}

	if err := a.load(); err == nil {
		return a, nil
	}

	if err := a.create(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Authority) CertPath() string {
	return a.certPath
}

func (a *Authority) CertPEM() []byte {
	return append([]byte(nil), a.certPEM...)
}

func (a *Authority) RootSubject() string {
	return a.caCert.Subject.String()
}

func (a *Authority) RootExpires() time.Time {
	return a.caCert.NotAfter
}

func (a *Authority) CertificateFor(host string) (*tls.Certificate, error) {
	name := normalizeHost(host)
	if name == "" {
		return nil, errors.New("empty server name")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if cert, ok := a.cache[name]; ok {
		return cert, nil
	}

	cert, err := a.issueLeaf(name)
	if err != nil {
		return nil, err
	}
	a.cache[name] = cert
	return cert, nil
}

func (a *Authority) load() error {
	certPEM, err := os.ReadFile(a.certPath)
	if err != nil {
		return err
	}
	keyPEM, err := os.ReadFile(a.keyPath)
	if err != nil {
		return err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return errors.New("invalid CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}

	key, err := parsePrivateKey(keyPEM)
	if err != nil {
		return err
	}

	a.certPEM = certPEM
	a.caCert = cert
	a.caKey = key
	return nil
}

func (a *Authority) create() error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serial, err := randomSerial()
	if err != nil {
		return err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "老虎快跑 Local Debugging Root CA",
			Organization: []string{"老虎快跑"},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(a.certPath, certPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(a.keyPath, keyPEM, 0o600); err != nil {
		return err
	}

	a.certPEM = certPEM
	a.caCert = template
	a.caKey = key
	return nil
}

func (a *Authority) issueLeaf(host string) (*tls.Certificate, error) {
	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:   now.Add(-time.Hour),
		NotAfter:    now.AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, a.caCert, &key.PublicKey, a.caKey)
	if err != nil {
		return nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	tlsCert.Leaf = template
	return &tlsCert, nil
}

func parsePrivateKey(keyPEM []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if signer, ok := key.(crypto.Signer); ok {
			return signer, nil
		}
	}
	return nil, fmt.Errorf("unsupported CA key type %q", block.Type)
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		if parsed, _, err := net.SplitHostPort(host); err == nil {
			host = parsed
		}
	}
	return strings.Trim(host, "[]")
}
