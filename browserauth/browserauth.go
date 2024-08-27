package browserauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal"
)

const (
	cliBrowserPath     = "/cli_login"
	cliAuthSuccessPage = "/auth_success"
	cliAuthFailurePage = "/auth_failure"
)

// Browser-based authentication callback handler. When using browser-based authentication,
// the user is given a link to follow which handles authentication w/ spacelift.io. Afterwards,
// the user is redirected to a page hosted on localhost which receives an encrypted API token.
// This type will handle the local authentication callback, and store the token in the given
// profile after completion. This helper package does not save the profile after updating the
// token. The caller of this package should call manager.Create(profile) to save the updated
// profile if a new valid token was received.
type Handler struct {
	Credentials       *session.StoredCredentials // Profile which is being authenticated
	Host              string                     // The address to which the local callback is bound
	Port              int                        // The port to which the local callback is bound
	AuthenticationURL string                     // URL where the user should be redirected
	key               *rsa.PrivateKey            // Key pair used to encrypt token handshake
	server            *http.Server               // The auth callback server
	endpoint          *url.URL                   // Parsed endpoint URL
	callbackChannel   chan error                 // Channel used to return success or failure after a callback
}

func Begin(credentials *session.StoredCredentials) (*Handler, error) {
	return BeginWithBindAddress(credentials, "localhost", 0)
}

func BeginWithBindAddress(credentials *session.StoredCredentials, host string, port int) (*Handler, error) {
	// Only API token credentials can be updated w/ browser based authentication
	if credentials == nil || credentials.Type != session.CredentialsTypeAPIToken {
		return nil, errors.New("can only use browser authentication with API token profiles")
	}

	// Pre-parse the endpoint now before starting any servers. If the endpoint is malformed,
	// we would rather catch it earlier, and having it preparsed makes building the auth
	// URL and redirect URLs easier.
	endpoint, err := url.Parse(credentials.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse endpoint url")
	}

	// Generate a private key for transferring the token
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate RSA key")
	}

	// Generate public key ASN1
	pubASN1, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal PKIX public key")
	}

	// Generate PEM-encoded public key
	var pubBuffer bytes.Buffer
	if err := pem.Encode(&pubBuffer, &pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubASN1}); err != nil {
		return nil, errors.Wrap(err, "could not pem-encode public key")
	}

	// Encode the public key for inclusion in URL
	pubKey := base64.RawURLEncoding.EncodeToString(pubBuffer.Bytes())

	// Construct the handler object. The server is not ready yet, as no handler has been
	// assigned, but the handler uses a method of this object, so we construct it a little
	// out of order initially.
	handler := &Handler{
		Credentials:     credentials,
		Host:            host,
		Port:            port,
		key:             key,
		endpoint:        endpoint,
		server:          &http.Server{ReadHeaderTimeout: 5 * time.Second},
		callbackChannel: make(chan error, 1),
	}

	// Setup the http server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.authCallback)
	handler.server.Handler = mux

	// Start our listening socket
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, errors.Wrap(err, "could not start callback server")
	}

	// Update the host and port so the caller knows where we're listening
	handler.Host = listener.Addr().(*net.TCPAddr).IP.String()
	handler.Port = listener.Addr().(*net.TCPAddr).Port

	// Build authentication URL
	authURL := endpoint.JoinPath(cliBrowserPath)

	// Build URL query values
	query := url.Values{}
	query.Add("key", pubKey)
	query.Add("port", fmt.Sprint(handler.Port))
	authURL.RawQuery = query.Encode()

	// Save the authentication URL
	handler.AuthenticationURL = authURL.String()

	// Start the HTTP server
	go handler.serveHttp(listener)

	return handler, nil
}

func (h *Handler) Cancel() {
	h.server.Close()
}

// Wait for a token to be received via the local callback endpoint or the
// given context to expire. If no error is returned here, you should have
// a token in profile.Credentials.AccessToken.
func (h *Handler) Wait(ctx context.Context) error {
	select {
	case callbackErr := <-h.callbackChannel:
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		err := h.server.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("could not stop local auth server: %s", err)
		}

		return callbackErr
	case <-ctx.Done():
		h.Cancel()
		return ctx.Err()
	}
}

// Start the local auth callback server on the given network listener. This will
// server forever/until the server is shutdown/closed. This method is assumed to
// be started as a background routine, and logs any startup errors w/ log.Printf.
func (h *Handler) serveHttp(listener net.Listener) {
	err := h.server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("could not start local auth server: %s", err)
	}
}

// Handler for the "/" route on the local auth callback server. Used to catch the result of
// interactive browser authentication and save the token. It will send the token extraction
// result back to Handler.Wait() via the callbackChannel.
func (h *Handler) authCallback(w http.ResponseWriter, r *http.Request) {
	// Extract and decrypt the token
	err := h.extractToken(r)
	if err != nil {
		// There was a problem, so redirect the user to a failure page on spacelift.io
		http.Redirect(w, r, h.endpoint.JoinPath(cliAuthFailurePage).String(), http.StatusTemporaryRedirect)
	} else {
		// We have a token, so just redirect the user back to a success page on spacelift.io
		http.Redirect(w, r, h.endpoint.JoinPath(cliAuthSuccessPage).String(), http.StatusTemporaryRedirect)
	}

	// Regardless of success or failure, send the error back upstream
	h.callbackChannel <- err
}

// Extract and decrypt a token from an interactive authentication callback using our
// internally generated RSA private key to decrypt the AES key, and then finally
// decrypt our shiny new token. The token is stored in the profile passed to Begin*()
func (h *Handler) extractToken(r *http.Request) error {
	// Retreive the base64-encoded encrypted token
	base64Token := r.URL.Query().Get("token")
	if base64Token == "" {
		return errors.New("missing token parameter")
	}

	// Retrieve the base64-encoded encrypted AES key
	base64Key := r.URL.Query().Get("key")
	if base64Key == "" {
		return errors.New("missing key parameter")
	}

	// Decode the token to an encrypted byte stream
	encToken, err := base64.RawURLEncoding.DecodeString(base64Token)
	if err != nil {
		return errors.Wrap(err, "could not decode session token")
	}

	// Decode the key to an encrypted byte stream
	encKey, err := base64.RawURLEncoding.DecodeString(base64Key)
	if err != nil {
		return errors.Wrap(err, "could not decode key")
	}

	// Decrypt the token AES key using our private key
	key, err := rsa.DecryptOAEP(sha512.New(), rand.Reader, h.key, encKey, nil)
	if err != nil {
		return errors.Wrap(err, "could not decrypt key")
	}

	// Decrypt the token using the decrypted AES key
	jwt, err := internal.DecryptAES(key, encToken)
	if err != nil {
		return errors.Wrap(err, "could not decrypt session token")
	}

	// Store the access token in the profile
	h.Credentials.AccessToken = string(jwt)

	return nil
}
