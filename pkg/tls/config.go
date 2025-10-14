// Package tls provides TLS configuration and credential management for DistKV
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// Config holds TLS configuration options
type Config struct {
	// Enabled indicates whether TLS is enabled
	Enabled bool

	// CertFile is the path to the server/client certificate file
	CertFile string

	// KeyFile is the path to the server/client private key file
	KeyFile string

	// CAFile is the path to the CA certificate file for verifying peers
	CAFile string

	// ServerName is the expected server name for verification (client-side)
	ServerName string

	// ClientAuth determines the server's policy for TLS client authentication
	// Options: NoClientCert, RequestClientCert, RequireAnyClientCert,
	// VerifyClientCertIfGiven, RequireAndVerifyClientCert
	ClientAuth tls.ClientAuthType

	// InsecureSkipVerify controls whether to verify the server certificate (client-side)
	// WARNING: Only use this for testing/development
	InsecureSkipVerify bool
}

// LoadServerCredentials loads TLS credentials for the gRPC server
func LoadServerCredentials(config *Config) (credentials.TransportCredentials, error) {
	if config == nil || !config.Enabled {
		return nil, fmt.Errorf("TLS not enabled")
	}

	// Load server certificate and private key
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %v", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   config.ClientAuth,
		MinVersion:   tls.VersionTLS12, // Enforce minimum TLS 1.2
	}

	// Load CA certificate for client verification if provided
	if config.CAFile != "" {
		certPool := x509.NewCertPool()
		ca, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}
		if !certPool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.ClientCAs = certPool
	}

	return credentials.NewTLS(tlsConfig), nil
}

// LoadClientCredentials loads TLS credentials for gRPC clients
func LoadClientCredentials(config *Config) (credentials.TransportCredentials, error) {
	if config == nil || !config.Enabled {
		return nil, fmt.Errorf("TLS not enabled")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName:         config.ServerName,
		InsecureSkipVerify: config.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12, // Enforce minimum TLS 1.2
	}

	// Load CA certificate for server verification
	if config.CAFile != "" {
		certPool := x509.NewCertPool()
		ca, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}
		if !certPool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = certPool
	}

	// Load client certificate if provided (for mutual TLS)
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsConfig), nil
}

// ValidateConfig validates the TLS configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if !config.Enabled {
		return nil // No validation needed if TLS is disabled
	}

	// Validate certificate and key files exist
	if config.CertFile == "" {
		return fmt.Errorf("certificate file is required when TLS is enabled")
	}
	if config.KeyFile == "" {
		return fmt.Errorf("key file is required when TLS is enabled")
	}

	// Check if files exist
	if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file does not exist: %s", config.CertFile)
	}
	if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file does not exist: %s", config.KeyFile)
	}

	// Validate CA file if provided
	if config.CAFile != "" {
		if _, err := os.Stat(config.CAFile); os.IsNotExist(err) {
			return fmt.Errorf("CA file does not exist: %s", config.CAFile)
		}
	}

	return nil
}
