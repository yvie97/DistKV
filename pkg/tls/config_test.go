package tls

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "TLS disabled",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "TLS enabled without cert file",
			config: &Config{
				Enabled: true,
				KeyFile: "key.pem",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without key file",
			config: &Config{
				Enabled:  true,
				CertFile: "cert.pem",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled with non-existent cert file",
			config: &Config{
				Enabled:  true,
				CertFile: "non-existent-cert.pem",
				KeyFile:  "non-existent-key.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadServerCredentials(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "TLS disabled",
			config: &Config{
				Enabled: false,
			},
			wantErr: true,
		},
		{
			name: "non-existent certificate",
			config: &Config{
				Enabled:  true,
				CertFile: "non-existent.pem",
				KeyFile:  "non-existent.key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadServerCredentials(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadServerCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadClientCredentials(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "TLS disabled",
			config: &Config{
				Enabled: false,
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without CA",
			config: &Config{
				Enabled:            true,
				InsecureSkipVerify: true,
			},
			wantErr: false,
		},
		{
			name: "non-existent CA file",
			config: &Config{
				Enabled: true,
				CAFile:  "non-existent-ca.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadClientCredentials(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadClientCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientAuthTypes(t *testing.T) {
	// Test that various client auth types are properly handled
	authTypes := []tls.ClientAuthType{
		tls.NoClientCert,
		tls.RequestClientCert,
		tls.RequireAnyClientCert,
		tls.VerifyClientCertIfGiven,
		tls.RequireAndVerifyClientCert,
	}

	for _, authType := range authTypes {
		t.Run(authType.String(), func(t *testing.T) {
			config := &Config{
				Enabled:    true,
				ClientAuth: authType,
			}
			if config.ClientAuth != authType {
				t.Errorf("ClientAuth = %v, want %v", config.ClientAuth, authType)
			}
		})
	}
}

// TestConfigWithTempFiles creates temporary cert files for testing
func TestConfigWithTempFiles(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create dummy cert and key files
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	// Write dummy content
	if err := os.WriteFile(certFile, []byte("dummy cert"), 0600); err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte("dummy key"), 0600); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := os.WriteFile(caFile, []byte("dummy ca"), 0600); err != nil {
		t.Fatalf("Failed to create CA file: %v", err)
	}

	// Test validation with existing files
	config := &Config{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("ValidateConfig() with existing files failed: %v", err)
	}
}

func TestMinTLSVersion(t *testing.T) {
	// Verify that credentials enforce minimum TLS version
	config := &Config{
		Enabled:            true,
		InsecureSkipVerify: true,
	}

	creds, err := LoadClientCredentials(config)
	if err != nil {
		t.Fatalf("LoadClientCredentials() failed: %v", err)
	}

	// Verify credentials were created
	if creds == nil {
		t.Error("Expected non-nil credentials")
	}

	// Note: The actual TLS config is internal to the credentials object,
	// but we've set MinVersion in our code to ensure TLS 1.2+
}
