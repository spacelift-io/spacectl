package session

import (
	"context"
	"github.com/franela/goblin"
	"strings"
	"testing"
)

// lookupWithEnv creates a custom lookup func which will return desired test values
func lookupWithEnv(envSpaceliftAPIToken, envSpaceliftAPIKeyID, envSpaceliftAPIKeySecret string) func(e string) (string, bool) {
	return func(e string) (string, bool) {
		s := ""
		b := false

		switch e {
		case EnvSpaceliftAPIToken:
			s = envSpaceliftAPIToken
			b = true
		case EnvSpaceliftAPIKeyEndpoint:
			s = "https://spacectl.app.spacelift.io"
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
			g.It("expect an error to find api endpoint in environment", func() {
				_, err := FromEnvironment(context.TODO(), nil)(func(s string) (string, bool) {
					return "", false
				})
				g.Assert(err).Equal(ErrEnvSpaceliftAPIKeyEndpoint)
			})
		})

		g.Describe("EnvSpaceliftAPIToken is set, EnvSpaceliftAPIKeyID is set, EnvSpaceliftAPIKeySecret is set", func() {
			l := lookupWithEnv("abc123", "abc123", "SuperSecret")
			g.It("expect EnvSpaceliftAPIToken to be used and EnvSpaceliftAPIKeyID and EnvSpaceliftAPIKeySecret to be ignored", func() {
				_, err := FromEnvironment(context.TODO(), nil)(l)
				// only looking at the error prefix because the suffix is set outside of this package and is out of our control
				g.Assert(strings.Split(err.Error(), ":")[0]).Equal("could not parse the API token")
			})
		})

		g.Describe("EnvSpaceliftAPIToken is an empty string", func() {
			g.Describe("EnvSpaceliftAPIKeyID is set, EnvSpaceliftAPIKeySecret is set", func() {
				l := lookupWithEnv("", "abc123", "SuperSecret")
				g.It("expect an exchange error because we're using fake credentials, this proves that EnvSpaceliftAPIKeyID and EnvSpaceliftAPIKeySecret are being used", func() {
					_, err := FromEnvironment(context.TODO(), nil)(l)
					// only looking at the error prefix because the suffix is set outside of this package and is out of our control
					g.Assert(strings.Split(err.Error(), ":")[0]).Equal("could not exchange API key and secret for token")
				})
			})

			g.Describe("EnvSpaceliftAPIKeyID is an empty string, EnvSpaceliftAPIKeySecret is set", func() {
				l := lookupWithEnv("", "", "SuperSecret")
				g.It("expect an error to find api key id in environment", func() {
					_, err := FromEnvironment(context.TODO(), nil)(l)
					g.Assert(err).Equal(ErrEnvSpaceliftAPIKeyID)
				})
			})

			g.Describe("EnvSpaceliftAPIKeyID is set, EnvSpaceliftAPIKeySecret is an empty string", func() {
				l := lookupWithEnv("", "abc123", "")
				g.It("expect an error to find api key secret in environment", func() {
					_, err := FromEnvironment(context.TODO(), nil)(l)
					g.Assert(err).Equal(ErrEnvSpaceliftAPIKeySecret)
				})
			})
		})
	})
}
