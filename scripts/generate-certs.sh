#!/bin/bash

# DistKV TLS Certificate Generation Script
# This script generates self-signed certificates for development and testing.
# For production, use certificates from a trusted Certificate Authority (CA).

set -e

# Configuration
CERT_DIR="${CERT_DIR:-./certs}"
VALIDITY_DAYS="${VALIDITY_DAYS:-365}"
KEY_SIZE="${KEY_SIZE:-2048}"

# Certificate details
COUNTRY="${COUNTRY:-US}"
STATE="${STATE:-California}"
CITY="${CITY:-San Francisco}"
ORG="${ORG:-DistKV}"
OU="${OU:-Development}"
CN="${CN:-localhost}"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "DistKV TLS Certificate Generator"
echo "================================================"
echo ""

# Create certificate directory
mkdir -p "$CERT_DIR"
echo -e "${GREEN}Created certificate directory: $CERT_DIR${NC}"

# Generate CA private key
echo ""
echo "Generating CA private key..."
openssl genrsa -out "$CERT_DIR/ca-key.pem" $KEY_SIZE

# Generate CA certificate
echo "Generating CA certificate..."
openssl req -new -x509 -days $VALIDITY_DAYS \
    -key "$CERT_DIR/ca-key.pem" \
    -out "$CERT_DIR/ca-cert.pem" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=DistKV-CA"

echo -e "${GREEN}CA certificate generated: $CERT_DIR/ca-cert.pem${NC}"

# Generate server private key
echo ""
echo "Generating server private key..."
openssl genrsa -out "$CERT_DIR/server-key.pem" $KEY_SIZE

# Generate server certificate signing request
echo "Generating server CSR..."
openssl req -new \
    -key "$CERT_DIR/server-key.pem" \
    -out "$CERT_DIR/server-csr.pem" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=$CN"

# Create server certificate extension file for SANs
cat > "$CERT_DIR/server-ext.cnf" <<EOF
subjectAltName = DNS:localhost,DNS:*.local,IP:127.0.0.1,IP:0.0.0.0
extendedKeyUsage = serverAuth
EOF

# Sign server certificate with CA
echo "Signing server certificate..."
openssl x509 -req -days $VALIDITY_DAYS \
    -in "$CERT_DIR/server-csr.pem" \
    -CA "$CERT_DIR/ca-cert.pem" \
    -CAkey "$CERT_DIR/ca-key.pem" \
    -CAcreateserial \
    -out "$CERT_DIR/server-cert.pem" \
    -extfile "$CERT_DIR/server-ext.cnf"

echo -e "${GREEN}Server certificate generated: $CERT_DIR/server-cert.pem${NC}"

# Generate client private key
echo ""
echo "Generating client private key..."
openssl genrsa -out "$CERT_DIR/client-key.pem" $KEY_SIZE

# Generate client certificate signing request
echo "Generating client CSR..."
openssl req -new \
    -key "$CERT_DIR/client-key.pem" \
    -out "$CERT_DIR/client-csr.pem" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=DistKV-Client"

# Create client certificate extension file
cat > "$CERT_DIR/client-ext.cnf" <<EOF
extendedKeyUsage = clientAuth
EOF

# Sign client certificate with CA
echo "Signing client certificate..."
openssl x509 -req -days $VALIDITY_DAYS \
    -in "$CERT_DIR/client-csr.pem" \
    -CA "$CERT_DIR/ca-cert.pem" \
    -CAkey "$CERT_DIR/ca-key.pem" \
    -CAcreateserial \
    -out "$CERT_DIR/client-cert.pem" \
    -extfile "$CERT_DIR/client-ext.cnf"

echo -e "${GREEN}Client certificate generated: $CERT_DIR/client-cert.pem${NC}"

# Clean up temporary files
rm -f "$CERT_DIR/server-csr.pem" "$CERT_DIR/server-ext.cnf"
rm -f "$CERT_DIR/client-csr.pem" "$CERT_DIR/client-ext.cnf"
rm -f "$CERT_DIR/ca-cert.srl"

# Set appropriate permissions
chmod 600 "$CERT_DIR"/*-key.pem
chmod 644 "$CERT_DIR"/*-cert.pem "$CERT_DIR"/ca-cert.pem

echo ""
echo "================================================"
echo -e "${GREEN}Certificate generation complete!${NC}"
echo "================================================"
echo ""
echo "Generated files in $CERT_DIR:"
echo "  ca-cert.pem      - CA certificate (distribute to all nodes)"
echo "  ca-key.pem       - CA private key (keep secure!)"
echo "  server-cert.pem  - Server certificate"
echo "  server-key.pem   - Server private key"
echo "  client-cert.pem  - Client certificate"
echo "  client-key.pem   - Client private key"
echo ""
echo -e "${YELLOW}IMPORTANT:${NC}"
echo "  - Keep all *-key.pem files secure and never commit to version control"
echo "  - For production, use certificates from a trusted CA"
echo "  - These self-signed certificates are for development/testing only"
echo ""
echo "To verify certificates:"
echo "  openssl x509 -in $CERT_DIR/server-cert.pem -text -noout"
echo "  openssl verify -CAfile $CERT_DIR/ca-cert.pem $CERT_DIR/server-cert.pem"
echo ""
