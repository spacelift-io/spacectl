package session

import (
	"context"
	"github.com/franela/goblin"
	"net/http"
	"net/http/httptest"
	"testing"
)

// lookupWithEnv creates a custom lookup func which will return desired test values
func lookupWithEnv(envSpaceliftAPIKeyEndpoint, envSpaceliftAPIToken, envSpaceliftAPIKeyID, envSpaceliftAPIKeySecret string) func(e string) (string, bool) {
	return func(e string) (string, bool) {
		s := ""
		b := false

		switch e {
		case EnvSpaceliftAPIToken:
			s = envSpaceliftAPIToken
			b = true
		case EnvSpaceliftAPIKeyEndpoint:
			s = envSpaceliftAPIKeyEndpoint
			b = true
		case EnvSpaceliftAPIKeyID:
			s = envSpaceliftAPIKeyID
			b = true
		case EnvSpaceliftAPIKeySecret:
			s = envSpaceliftAPIKeySecret
			b = true
		}

		return s, b
	}
}

func TestFromEnvironment(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("FromEnvironment", func() {
		g.Describe("EnvSpaceliftAPIKeyEndpoint is not set", func() {
			l := lookupWithEnv("", "", "abc123", "SuperSecret")
			g.It("expect an error to find api endpoint in environment", func() {
				_, err := FromEnvironment(context.TODO(), nil)(l)
				g.Assert(err).Equal(ErrEnvSpaceliftAPIKeyEndpoint)
			})
		})

		g.Describe("EnvSpaceliftAPIToken is set, EnvSpaceliftAPIKeyID is set, EnvSpaceliftAPIKeySecret is set", func() {
			g.It("expect EnvSpaceliftAPIToken to be used and EnvSpaceliftAPIKeyID and EnvSpaceliftAPIKeySecret to be ignored", func() {
				// Create a mock http server to handle JWT exchange
				server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					rw.Write([]byte(`{"data":{"apiKeyUser":{"jwt":"SuperSecretJWT","validUntil":123}}}`))
				}))
				// Close the server when test finishes
				defer server.Close()
				// API Token JWT generated at https://jwt.io/#debugger-io with contents:
				//     Header: {"alg": "HS256","typ": "JWT"}
				//     Payload: {"aud": "spacectl","exp": 1516239022}
				l := lookupWithEnv(server.URL, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJzcGFjZWN0bCIsImV4cCI6MTUxNjIzOTAyMn0.fsKd_N2TKXpx83JSPPw47zYzQ8sbSzGVPZcyGpwp05U", "abc123", "SuperSecret")
				s, err := FromEnvironment(context.TODO(), server.Client())(l)
				g.Assert(err).IsNil("expected no error when creating session")
				g.Assert(s.Type()).Equal(CredentialsTypeAPIToken)
			})
		})

		g.Describe("EnvSpaceliftAPIToken is an empty string", func() {
			g.Describe("EnvSpaceliftAPIKeyID is set, EnvSpaceliftAPIKeySecret is set", func() {
				g.It("expect a CredentialsTypeAPIKey session to be created", func() {
					// Create a mock http server to handle JWT exchange
					server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
						rw.Write([]byte(`{"data":{"apiKeyUser":{"jwt":"SuperSecretJWT","validUntil":123}}}`))
					}))
					// Close the server when test finishes
					defer server.Close()

					l := lookupWithEnv(server.URL, "", "abc123", "SuperSecret")

					s, err := FromEnvironment(context.TODO(), server.Client())(l)
					g.Assert(err).IsNil("expected no error when creating session")
					g.Assert(s.Type()).Equal(CredentialsTypeAPIKey)
				})
			})

			g.Describe("EnvSpaceliftAPIKeyID is an empty string, EnvSpaceliftAPIKeySecret is set", func() {
				g.It("expect an error to find api key id in environment", func() {
					l := lookupWithEnv("https://spacectl.app.spacelift.io", "", "", "SuperSecret")
					_, err := FromEnvironment(context.TODO(), nil)(l)
					g.Assert(err).Equal(ErrEnvSpaceliftAPIKeyID)
				})
			})

			g.Describe("EnvSpaceliftAPIKeyID is set, EnvSpaceliftAPIKeySecret is an empty string", func() {
				g.It("expect an error to find api key secret in environment", func() {
					l := lookupWithEnv("https://spacectl.app.spacelift.io", "", "abc123", "")
					_, err := FromEnvironment(context.TODO(), nil)(l)
					g.Assert(err).Equal(ErrEnvSpaceliftAPIKeySecret)
				})
			})
		})
	})
}
