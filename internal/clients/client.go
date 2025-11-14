package clients

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	models "ssl-manager/internal/models"
	utils "ssl-manager/internal/utils"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type Client struct {
	log     *utils.Logger
	cfg     *utils.Config
	Manager *autocert.Manager
}

func NewClient(log *utils.Logger, cfg *utils.Config) (*Client, error) {
	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(cfg.Certs.StorageDir),
		Email:  cfg.Certs.Email,
	}
	return &Client{
		log:     log,
		cfg:     cfg,
		Manager: manager,
	}, nil
}

func (c *Client) CreateCertificate(domain string) (*models.CertificateData, error) {
	cert, err := c.Manager.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for domain %s: %w", domain, err)
	}

	certPEM := cert.Certificate[0]
	keyBytes := cert.PrivateKey

	var validFrom, validTo time.Time
	if len(cert.Leaf.NotBefore.String()) > 0 {
		validFrom = cert.Leaf.NotBefore
		validTo = cert.Leaf.NotAfter
	}

	return &models.CertificateData{
		Cert:      certPEM,
		Key:       encodePrivateKeyToPEM(keyBytes),
		Chain:     joinCertChain(cert.Certificate),
		ValidFrom: validFrom,
		ValidTo:   validTo,
	}, nil
}

func (c *Client) SaveCertificateFiles(domain string, certData *models.CertificateData) (*models.CertificatePaths, error) {
	dir := filepath.Join(c.cfg.Certs.StorageDir, domain)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cert dir: %w", err)
	}

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	chainPath := filepath.Join(dir, "chain.pem")

	if err := os.WriteFile(certPath, certData.Cert, 0600); err != nil {
		return nil, fmt.Errorf("failed to write cert: %w", err)
	}
	if err := os.WriteFile(keyPath, certData.Key, 0600); err != nil {
		return nil, fmt.Errorf("failed to write key: %w", err)
	}
	if err := os.WriteFile(chainPath, certData.Chain, 0600); err != nil {
		return nil, fmt.Errorf("failed to write chain: %w", err)
	}

	return &models.CertificatePaths{
		Cert:  certPath,
		Key:   keyPath,
		Chain: chainPath,
	}, nil
}

func (c *Client) DeleteCertificateFiles(certPath, keyPath, chainPath string) error {
	paths := []string{certPath, keyPath, chainPath}

	for _, p := range paths {
		if p == "" {
			continue
		}

		if _, err := os.Stat(p); os.IsNotExist(err) {
			continue
		}

		if err := os.Remove(p); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", p, err)
		}
	}

	return nil
}

func joinCertChain(chain [][]byte) []byte {
	result := []byte{}
	for _, c := range chain {
		result = append(result, c...)
	}
	return result
}

func encodePrivateKeyToPEM(key interface{}) []byte {
	var (
		privBytes []byte
		err       error
		blockType string
	)

	switch k := key.(type) {
	case *rsa.PrivateKey:
		privBytes = x509.MarshalPKCS1PrivateKey(k)
		blockType = "RSA PRIVATE KEY"

	case *ecdsa.PrivateKey:
		privBytes, err = x509.MarshalECPrivateKey(k)
		if err != nil {
			panic(fmt.Errorf("failed to marshal ECDSA private key: %w", err))
		}
		blockType = "EC PRIVATE KEY"

	case ed25519.PrivateKey:
		privBytes, err = x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			panic(fmt.Errorf("failed to marshal Ed25519 private key: %w", err))
		}
		blockType = "PRIVATE KEY"

	default:
		panic(fmt.Errorf("unsupported private key type: %T", key))
	}

	block := &pem.Block{
		Type:  blockType,
		Bytes: privBytes,
	}
	return pem.EncodeToMemory(block)
}
