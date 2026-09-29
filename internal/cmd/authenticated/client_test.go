package authenticated

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacectl/client"
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

// configureProxyAuth wraps the transport with proxy auth; verify it does when
// Airbnb defaults apply to the endpoint.
func TestConfigureProxyAuth_WrapsTransport(t *testing.T) {
	httpClient := &http.Client{Transport: http.DefaultTransport}
	cfg := &session.ProxyAuthConfig{
		Command:   []string{"/usr/local/bin/iap-auth", "--cache-only"},
		SetupHint: "Install iap-auth.",
	}
	httpClient.Transport = client.WrapWithProxyAuth(httpClient.Transport, cfg)
	if !client.IsWrappedWithProxyAuth(httpClient.Transport) {
		t.Errorf("expected outermost transport to be the proxy-auth wrapper, got %T", httpClient.Transport)
	}
}

// Verify that configureProxyAuth wraps on top of configureTLS, not underneath.
func TestConfigureTLS_ThenProxyAuth_CorrectOrder(t *testing.T) {
	t.Setenv(EnvSpaceliftAPIClientCA, "")
	t.Setenv("SSL_CERT_FILE", "")

	httpClient := &http.Client{}
	err := configureTLS(httpClient)
	require.NoError(t, err)

	cfg := &session.ProxyAuthConfig{
		Command: []string{"/usr/local/bin/test-auth"},
	}
	httpClient.Transport = client.WrapWithProxyAuth(httpClient.Transport, cfg)

	if !client.IsWrappedWithProxyAuth(httpClient.Transport) {
		t.Fatal("expected proxy-auth as outermost wrapper")
	}
	inner := client.UnwrapProxyAuth(httpClient.Transport)
	if _, ok := inner.(*http.Transport); !ok {
		t.Errorf("expected *http.Transport inside proxy-auth wrapper, got %T", inner)
	}
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
