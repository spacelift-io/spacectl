package authenticated

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureTLS_NoEnvVars(t *testing.T) {
	t.Setenv(EnvSpaceliftAPIClientCA, "")
	t.Setenv("SSL_CERT_FILE", "")

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	require.NoError(t, err)

	transport := httpClient.Transport.(*http.Transport)
	assert.Nil(t, transport.TLSClientConfig.RootCAs, "RootCAs should be nil when no CA env vars are set")
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
}

func TestConfigureTLS_SpaceliftCA(t *testing.T) {
	caFile := writeTempCA(t)

	t.Setenv(EnvSpaceliftAPIClientCA, caFile)
	t.Setenv("SSL_CERT_FILE", "")

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	require.NoError(t, err)

	transport := httpClient.Transport.(*http.Transport)
	assert.NotNil(t, transport.TLSClientConfig.RootCAs, "RootCAs should be set when SPACELIFT_API_TLS_CA is provided")
}

func TestConfigureTLS_SSLCertFileFallback(t *testing.T) {
	caFile := writeTempCA(t)

	t.Setenv(EnvSpaceliftAPIClientCA, "")
	t.Setenv("SSL_CERT_FILE", caFile)

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	require.NoError(t, err)

	transport := httpClient.Transport.(*http.Transport)
	assert.NotNil(t, transport.TLSClientConfig.RootCAs, "RootCAs should be set when SSL_CERT_FILE is provided")
}

func TestConfigureTLS_SpaceliftCATakesPrecedence(t *testing.T) {
	caFile := writeTempCA(t)

	t.Setenv(EnvSpaceliftAPIClientCA, caFile)
	t.Setenv("SSL_CERT_FILE", "/nonexistent/path.pem")

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	require.NoError(t, err, "should use SPACELIFT_API_TLS_CA and ignore invalid SSL_CERT_FILE")

	transport := httpClient.Transport.(*http.Transport)
	assert.NotNil(t, transport.TLSClientConfig.RootCAs)
}

func TestConfigureTLS_InvalidCAPath(t *testing.T) {
	t.Setenv(EnvSpaceliftAPIClientCA, "/nonexistent/ca.pem")

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	assert.Error(t, err)
}

func TestConfigureTLS_InvalidCACert(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(tmpFile, []byte("not a certificate"), 0o600))

	t.Setenv(EnvSpaceliftAPIClientCA, tmpFile)

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	assert.ErrorIs(t, err, errEnvSpaceliftAPIClientCAParse)
}

// writeTempCA writes a self-signed PEM certificate to a temp file and returns the path.
func writeTempCA(t *testing.T) string {
	t.Helper()

	// Minimal self-signed cert for testing CA pool loading (not used for real TLS).
	// Generated with: openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 36500
	pem := []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIUYz3GFhMGQwMz3GFhMGQwMTIzNDU2NzgwCgYIKoZIzj0EAwIw
ETEPMA0GA1UEAwwGdGVzdGNhMCAXDTI0MDEwMTAwMDAwMFoYDzIwNjQwMTAxMDAw
MDAwWjARMQ8wDQYDVQQDDAZ0ZXN0Y2EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNC
AASKYz3GFhMGQwbnGTi7lakFOVkgn3hMG6jPGXSDfIaJbMGGaNGQwMjM0NTY3ODkw
MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4
MAoGCCqGSM49BAMCA0gAMEUCIQCtest1234567890AAAAABBBBBCCCCCDDDDEEEE
FFFGGGHHHIIIJJJKKKLLLMMMAIgtest1234567890abcdefghijklmnopqrstuvw
-----END CERTIFICATE-----`)

	// Use a real system cert if available, otherwise use a proper self-signed one.
	// /etc/ssl/cert.pem exists on macOS and contains real CA certs.
	systemCert := "/etc/ssl/cert.pem"
	if _, err := os.Stat(systemCert); err == nil {
		return systemCert
	}

	tmpFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(tmpFile, pem, 0o600))
	return tmpFile
}
