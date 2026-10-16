package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacectl/client/session"
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

func TestConfigureTransport_AppliesTLSFromEnvironment(t *testing.T) {
	writeTestProfile(t, nil)
	t.Setenv(EnvSpaceliftAPIClientCA, writeTempCA(t))

	httpClient := &http.Client{}
	require.NoError(t, configureTransport(httpClient))

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
	assert.ErrorIs(t, err, ErrEnvSpaceliftAPIClientCAParse)
}

func TestConfigureTransport_WithoutProxyAuth(t *testing.T) {
	writeTestProfile(t, nil)

	httpClient := &http.Client{}
	require.NoError(t, configureTransport(httpClient))

	_, ok := httpClient.Transport.(*http.Transport)
	assert.True(t, ok, "expected a plain *http.Transport, got %T", httpClient.Transport)
}

func TestConfigureTransport_WithProxyAuth_WrapsTLSTransport(t *testing.T) {
	writeTestProfile(t, &session.ProxyAuthConfig{Command: []string{"/usr/local/bin/test-auth", "--cached"}})

	httpClient := &http.Client{}
	require.NoError(t, configureTransport(httpClient))

	proxy, ok := httpClient.Transport.(*ProxyAuthTransport)
	require.True(t, ok, "expected proxy auth as the outermost transport, got %T", httpClient.Transport)

	inner, ok := proxy.Base().(*http.Transport)
	require.True(t, ok, "expected *http.Transport inside the proxy auth wrapper, got %T", proxy.Base())
	assert.Equal(t, uint16(tls.VersionTLS12), inner.TLSClientConfig.MinVersion, "TLS settings must apply underneath proxy auth")
}

func TestConfigureTransport_NoProfile(t *testing.T) {
	t.Setenv(session.EnvSpaceliftConfigDirectory, t.TempDir())
	t.Setenv(session.EnvSpaceliftProfile, "")
	t.Setenv(EnvSpaceliftAPIClientCA, "")
	t.Setenv("SSL_CERT_FILE", "")

	httpClient := &http.Client{}
	require.NoError(t, configureTransport(httpClient))

	_, ok := httpClient.Transport.(*http.Transport)
	assert.True(t, ok, "expected a plain *http.Transport, got %T", httpClient.Transport)
}

// writeTestProfile points SPACELIFT_CONFIG_DIR at a temporary directory holding
// a single, selected profile with the given proxy auth configuration.
func writeTestProfile(t *testing.T, proxyAuth *session.ProxyAuthConfig) {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(session.EnvSpaceliftConfigDirectory, dir)
	t.Setenv(session.EnvSpaceliftProfile, "")
	t.Setenv(EnvSpaceliftAPIClientCA, "")
	t.Setenv("SSL_CERT_FILE", "")

	config := map[string]interface{}{
		"currentProfileAlias": "test",
		"profiles": map[string]*session.Profile{
			"test": {
				Alias: "test",
				Credentials: &session.StoredCredentials{
					Type:        session.CredentialsTypeAPIToken,
					Endpoint:    "https://spacelift.example.com",
					AccessToken: "token",
					ProxyAuth:   proxyAuth,
				},
			},
		},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, session.ConfigFileName), data, 0o600))
}

// writeTempCA generates a self-signed CA certificate and writes it to a temp file.
func writeTempCA(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "testca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tmpFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(tmpFile, certPEM, 0o600))
	return tmpFile
}
