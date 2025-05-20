package authenticated

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client"
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

var (
	errEnvSpaceliftAPIClientCAParse = fmt.Errorf("failed to parse %s", EnvSpaceliftAPIClientCA)
)

// Client is the authenticated client that can be used by all CLI commands.
var Client client.Client

// Ensure is a way of ensuring that the Client exists, and it meant to be used
// as a Before action for commands that need it.
func Ensure(ctx context.Context, _ *cli.Command) (context.Context, error) {
	httpClient := session.Defaults()

	if err := configureTLS(httpClient); err != nil {
		return ctx, err
	}

	session, err := session.New(ctx, httpClient)
	if err != nil {
		return ctx, err
	}

	Client = client.New(httpClient, session)

	return ctx, nil
}

// configureTLS configures client TLS from the environment.
func configureTLS(httpClient *http.Client) error {
	clientTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if caFile, ok := os.LookupEnv(EnvSpaceliftAPIClientCA); ok && caFile != "" {
		caCert, err := os.ReadFile("cacert")
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return errEnvSpaceliftAPIClientCAParse
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
