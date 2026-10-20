package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/spacelift-io/spacectl/client/session"
)

const (
	// EnvSpaceliftAPIClientTLSCert represents the path to a client certificate for
	// communicating with the Spacelift API endpoint.
	EnvSpaceliftAPIClientTLSCert = "SPACELIFT_API_TLS_CERT"

	// EnvSpaceliftAPIClientTLSKey represents the path to a client key for
	// communicating with the Spacelift API endpoint.
	EnvSpaceliftAPIClientTLSKey = "SPACELIFT_API_TLS_KEY"

	// EnvSpaceliftAPIClientCA represents the path to a CA bundle for
	// verifying Spacelift API endpoint.
	EnvSpaceliftAPIClientCA = "SPACELIFT_API_TLS_CA"
)

// ErrEnvSpaceliftAPIClientCAParse is returned when the CA bundle referenced by
// SPACELIFT_API_TLS_CA cannot be parsed.
var ErrEnvSpaceliftAPIClientCAParse = fmt.Errorf("failed to parse %s", EnvSpaceliftAPIClientCA)

var (
	httpClient = &http.Client{
		Timeout: 60 * time.Second,
	}
	httpClientOnce sync.Once
	httpClientErr  error
)

// GetHTTPClient returns the HTTP client shared by everything that talks to
// Spacelift. On first use its transport is configured from the environment
// (TLS settings) and from the current profile (proxy authentication), so that
// every caller sees the same settings.
func GetHTTPClient() (*http.Client, error) {
	httpClientOnce.Do(func() {
		httpClientErr = configureTransport(httpClient)
	})

	return httpClient, httpClientErr
}

// configureTransport sets up the transport chain on httpClient: TLS first,
// then proxy authentication wrapping it, so that the proxy credential is
// attached to requests sent over the TLS-configured transport.
func configureTransport(httpClient *http.Client) error {
	if err := configureTLS(httpClient); err != nil {
		return err
	}

	return configureProxyAuth(httpClient)
}

// configureTLS configures client TLS from the environment.
func configureTLS(httpClient *http.Client) error {
	clientTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Check SPACELIFT_API_TLS_CA first, then fall back to SSL_CERT_FILE.
	// Setting RootCAs to a non-nil pool forces Go's pure-Go certificate
	// verifier, bypassing Security.framework (which fails in sandboxed
	// environments where macOS Keychain is inaccessible).
	caFile := ""
	if f, ok := os.LookupEnv(EnvSpaceliftAPIClientCA); ok && f != "" {
		caFile = f
	} else if f, ok := os.LookupEnv("SSL_CERT_FILE"); ok && f != "" {
		caFile = f
	}

	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return ErrEnvSpaceliftAPIClientCAParse
		}

		clientTLS.RootCAs = caCertPool
	}

	keyFile, keyOk := os.LookupEnv(EnvSpaceliftAPIClientTLSKey)
	certFile, certOk := os.LookupEnv(EnvSpaceliftAPIClientTLSCert)

	if keyOk && keyFile != "" && certOk && certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}

		clientTLS.Certificates = []tls.Certificate{cert}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = clientTLS

	httpClient.Transport = transport

	return nil
}

// configureProxyAuth wraps the transport with proxy authentication when a
// proxy token source is configured, either through the environment or the
// current profile's proxy_auth configuration. A missing or unreadable profile
// is not treated as an error here: the session layer reports that with a more
// helpful message.
func configureProxyAuth(httpClient *http.Client) error {
	var (
		cfg        *session.ProxyAuthConfig
		configFile string
	)

	if manager, err := session.UserProfileManager(); err == nil {
		if profile, err := manager.CurrentValidated(); err == nil && profile != nil && profile.Credentials != nil {
			cfg = profile.Credentials.ProxyAuth
			configFile = manager.ConfigurationFile
		}
	}

	transport, err := WrapWithProxyAuth(httpClient.Transport, cfg, configFile)
	if err != nil {
		return err
	}

	httpClient.Transport = transport

	return nil
}
