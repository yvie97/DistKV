# TLS Configuration Guide for DistKV

This guide explains how to set up and configure TLS (Transport Layer Security) for secure communication in DistKV clusters.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Certificate Generation](#certificate-generation)
- [Server Configuration](#server-configuration)
- [Client Configuration](#client-configuration)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)
- [Security Best Practices](#security-best-practices)

## Overview

DistKV supports TLS for encrypting communication between:
- **Clients and servers**: Protects client requests and responses
- **Server-to-server**: Secures inter-node replication and gossip traffic

TLS features include:
- **Encryption**: All data in transit is encrypted using TLS 1.2+
- **Authentication**: Verify server identity using certificates
- **Mutual TLS (mTLS)**: Optional client certificate authentication
- **Flexible Configuration**: Enable TLS cluster-wide or per-connection

## Quick Start

### 1. Generate Self-Signed Certificates (Development Only)

```bash
# Generate certificates for development/testing
./scripts/generate-certs.sh

# Certificates will be created in ./certs/ directory
# - ca-cert.pem: CA certificate (distribute to all nodes)
# - server-cert.pem & server-key.pem: Server certificate and key
# - client-cert.pem & client-key.pem: Client certificate and key
```

### 2. Start Server with TLS

```bash
./build/distkv-server \
  -node-id=node1 \
  -address=localhost:8080 \
  -data-dir=./data1 \
  -tls-enabled=true \
  -tls-cert-file=./certs/server-cert.pem \
  -tls-key-file=./certs/server-key.pem \
  -tls-ca-file=./certs/ca-cert.pem
```

### 3. Connect Client with TLS

```bash
./build/distkv-client \
  -server=localhost:8080 \
  -tls-enabled=true \
  -tls-ca-file=./certs/ca-cert.pem \
  put mykey "secure value"
```

## Certificate Generation

### Development Certificates

Use the provided script for quick testing:

```bash
# Default generation
./scripts/generate-certs.sh

# Custom certificate directory
CERT_DIR=./my-certs ./scripts/generate-certs.sh

# Custom validity period (default: 365 days)
VALIDITY_DAYS=730 ./scripts/generate-certs.sh

# Custom organization details
COUNTRY=US STATE=CA CITY="San Francisco" ORG="MyOrg" \
  ./scripts/generate-certs.sh
```

### Production Certificates

For production, use certificates from a trusted Certificate Authority (CA):

1. **Purchase from a CA**: Buy certificates from DigiCert, Let's Encrypt, etc.
2. **Generate CSR**: Create a Certificate Signing Request
3. **Submit to CA**: Submit CSR to your chosen CA
4. **Install certificates**: Place received certificates on your servers

Example generating a CSR:

```bash
# Generate private key
openssl genrsa -out server-key.pem 2048

# Generate CSR
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=US/ST=CA/L=San Francisco/O=MyOrg/CN=distkv.example.com"

# Submit server.csr to your CA and receive signed certificate
```

### Certificate Requirements

**Server Certificate:**
- Must include Subject Alternative Names (SANs) for all server addresses
- Should support both DNS names and IP addresses
- Extended Key Usage: serverAuth

**Client Certificate (for mTLS):**
- Can be generated from the same CA
- Extended Key Usage: clientAuth

## Server Configuration

### Basic TLS Configuration

```bash
distkv-server \
  -node-id=node1 \
  -address=localhost:8080 \
  -tls-enabled=true \
  -tls-cert-file=/path/to/server-cert.pem \
  -tls-key-file=/path/to/server-key.pem \
  -tls-ca-file=/path/to/ca-cert.pem
```

### TLS Configuration Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-tls-enabled` | Enable TLS | `false` | Yes (for TLS) |
| `-tls-cert-file` | Server certificate path | `""` | Yes (if TLS enabled) |
| `-tls-key-file` | Server private key path | `""` | Yes (if TLS enabled) |
| `-tls-ca-file` | CA certificate path | `""` | Recommended |
| `-tls-client-auth` | Client auth policy | `NoClientCert` | No |

### Client Authentication Policies

The `-tls-client-auth` flag controls client certificate verification:

- **NoClientCert** (default): No client certificate required
- **RequestClientCert**: Request client cert but don't require it
- **RequireAnyClientCert**: Require client cert, don't verify CA
- **VerifyClientCertIfGiven**: Verify client cert if provided
- **RequireAndVerifyClientCert**: Require and verify client cert (mTLS)

Example with mutual TLS:

```bash
distkv-server \
  -tls-enabled=true \
  -tls-cert-file=./certs/server-cert.pem \
  -tls-key-file=./certs/server-key.pem \
  -tls-ca-file=./certs/ca-cert.pem \
  -tls-client-auth=RequireAndVerifyClientCert
```

### Multi-Node Cluster with TLS

All nodes must use compatible TLS settings:

```bash
# Node 1
./build/distkv-server \
  -node-id=node1 \
  -address=localhost:8080 \
  -tls-enabled=true \
  -tls-cert-file=./certs/server-cert.pem \
  -tls-key-file=./certs/server-key.pem \
  -tls-ca-file=./certs/ca-cert.pem

# Node 2 (joins Node 1)
./build/distkv-server \
  -node-id=node2 \
  -address=localhost:8081 \
  -seed-nodes=localhost:8080 \
  -tls-enabled=true \
  -tls-cert-file=./certs/server-cert.pem \
  -tls-key-file=./certs/server-key.pem \
  -tls-ca-file=./certs/ca-cert.pem

# Node 3 (joins Node 1)
./build/distkv-server \
  -node-id=node3 \
  -address=localhost:8082 \
  -seed-nodes=localhost:8080 \
  -tls-enabled=true \
  -tls-cert-file=./certs/server-cert.pem \
  -tls-key-file=./certs/server-key.pem \
  -tls-ca-file=./certs/ca-cert.pem
```

## Client Configuration

### Basic Client TLS

```bash
distkv-client \
  -server=localhost:8080 \
  -tls-enabled=true \
  -tls-ca-file=./certs/ca-cert.pem \
  get mykey
```

### Client TLS Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-tls-enabled` | Enable TLS | `false` | Yes (for TLS) |
| `-tls-ca-file` | CA cert for server verification | `""` | Recommended |
| `-tls-cert-file` | Client cert (for mTLS) | `""` | No |
| `-tls-key-file` | Client private key (for mTLS) | `""` | No |
| `-tls-server-name` | Expected server name | `localhost` | No |
| `-tls-insecure-skip-verify` | Skip cert verification | `false` | No |

### Mutual TLS (mTLS)

When the server requires client certificates:

```bash
distkv-client \
  -server=localhost:8080 \
  -tls-enabled=true \
  -tls-ca-file=./certs/ca-cert.pem \
  -tls-cert-file=./certs/client-cert.pem \
  -tls-key-file=./certs/client-key.pem \
  put mykey "authenticated value"
```

### Testing with Self-Signed Certificates

For development only, you can skip certificate verification:

```bash
# WARNING: This is insecure and should NEVER be used in production
distkv-client \
  -server=localhost:8080 \
  -tls-enabled=true \
  -tls-insecure-skip-verify=true \
  get mykey
```

## Production Deployment

### Security Checklist

- [ ] Use certificates from a trusted CA
- [ ] Never use self-signed certificates in production
- [ ] Enable mutual TLS for inter-node communication
- [ ] Set appropriate file permissions (600 for keys, 644 for certs)
- [ ] Rotate certificates before expiration
- [ ] Store private keys securely (consider using a secret manager)
- [ ] Enable TLS 1.2 or higher (TLS 1.3 recommended)
- [ ] Never commit certificates/keys to version control
- [ ] Monitor certificate expiration dates
- [ ] Use strong key sizes (2048-bit RSA minimum, 256-bit ECC recommended)

### File Permissions

```bash
# Set secure permissions on private keys
chmod 600 certs/*-key.pem

# Set readable permissions on certificates
chmod 644 certs/*-cert.pem certs/ca-cert.pem

# Restrict directory access
chmod 700 certs/
```

### Docker Deployment with TLS

```dockerfile
# Mount certificates as secrets
docker run -d \
  --name distkv-node1 \
  -v /secure/certs:/certs:ro \
  distkv/server \
  -tls-enabled=true \
  -tls-cert-file=/certs/server-cert.pem \
  -tls-key-file=/certs/server-key.pem \
  -tls-ca-file=/certs/ca-cert.pem
```

### Kubernetes Deployment with TLS

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: distkv-tls
type: kubernetes.io/tls
data:
  ca.crt: <base64-encoded-ca-cert>
  tls.crt: <base64-encoded-server-cert>
  tls.key: <base64-encoded-server-key>
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: distkv
spec:
  template:
    spec:
      containers:
      - name: distkv
        image: distkv/server:latest
        args:
          - "-tls-enabled=true"
          - "-tls-cert-file=/certs/tls.crt"
          - "-tls-key-file=/certs/tls.key"
          - "-tls-ca-file=/certs/ca.crt"
        volumeMounts:
        - name: tls-certs
          mountPath: /certs
          readOnly: true
      volumes:
      - name: tls-certs
        secret:
          secretName: distkv-tls
```

## Troubleshooting

### Common Issues

#### 1. "certificate signed by unknown authority"

**Cause**: Client doesn't trust the server's CA certificate

**Solution**: Provide the CA certificate to the client:
```bash
-tls-ca-file=./certs/ca-cert.pem
```

#### 2. "x509: certificate is valid for localhost, not <hostname>"

**Cause**: Server certificate doesn't include the hostname you're connecting to

**Solution**:
- Use the correct hostname/IP that matches the certificate
- Regenerate certificate with correct SANs
- Use `-tls-server-name=localhost` to override verification

#### 3. "tls: bad certificate" (with mTLS)

**Cause**: Server rejected client certificate

**Solution**:
- Ensure client certificate is signed by the CA the server trusts
- Check that `-tls-ca-file` on server includes the CA that signed the client cert
- Verify client certificate hasn't expired

#### 4. "connection refused" or "connection timeout"

**Cause**: Network connectivity issues or server not listening on correct port

**Solution**:
- Verify server is running and listening on the specified address
- Check firewall rules
- Ensure TLS is enabled on both client and server

#### 5. "failed to load TLS credentials"

**Cause**: Certificate or key file not found or has incorrect permissions

**Solution**:
- Verify file paths are correct
- Check file permissions (keys should be 600)
- Ensure files exist and are readable

### Debugging TLS Issues

Enable verbose logging:

```bash
# Server with debug logging
GRPC_GO_LOG_VERBOSITY_LEVEL=99 GRPC_GO_LOG_SEVERITY_LEVEL=info \
  ./build/distkv-server -tls-enabled=true ...

# Verify certificate
openssl x509 -in certs/server-cert.pem -text -noout

# Test TLS connection
openssl s_client -connect localhost:8080 \
  -CAfile certs/ca-cert.pem \
  -cert certs/client-cert.pem \
  -key certs/client-key.pem
```

## Security Best Practices

### 1. Certificate Management

- **Expiration**: Set up monitoring for certificate expiration
- **Rotation**: Rotate certificates regularly (e.g., every 90 days)
- **Revocation**: Maintain a certificate revocation list (CRL)
- **Backup**: Securely backup certificates and keys

### 2. Private Key Protection

- Store keys in secure locations (HSM, secret manager)
- Never log private keys
- Use strong file permissions (600)
- Encrypt keys at rest
- Limit access to authorized personnel only

### 3. Network Security

- Use TLS 1.2 or higher (TLS 1.3 recommended)
- Disable weak cipher suites
- Enable Perfect Forward Secrecy (PFS)
- Use strong key sizes (2048+ bits for RSA)
- Consider mutual TLS for sensitive deployments

### 4. Operational Security

- Monitor TLS connections for anomalies
- Log certificate verification failures
- Implement certificate pinning for critical connections
- Regularly audit certificate usage
- Keep OpenSSL and gRPC libraries up to date

### 5. Development vs Production

| Aspect | Development | Production |
|--------|------------|------------|
| Certificates | Self-signed | CA-signed |
| Key size | 2048-bit | 4096-bit |
| Expiry | 365 days | 90 days |
| Skip verification | Acceptable (testing) | NEVER |
| Mutual TLS | Optional | Recommended |
| Key storage | Local files | Secret manager |

## Additional Resources

- [TLS Best Practices](https://www.ssllabs.com/projects/best-practices/)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [gRPC Authentication Guide](https://grpc.io/docs/guides/auth/)
- [NIST TLS Guidelines](https://csrc.nist.gov/publications/detail/sp/800-52/rev-2/final)

## Need Help?

If you encounter issues with TLS setup:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review server logs for specific error messages
3. Verify certificate validity with OpenSSL
4. Ensure all nodes have compatible TLS configurations
5. Open an issue on GitHub with detailed error information
