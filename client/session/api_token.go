package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/shurcooL/graphql"
)

// If the token is about to expire, we'd rather exchange it now than risk having
// a stale one.
const timePadding = 30 * time.Second

func parseUnverifiedClaims(token string) (jwt.RegisteredClaims, error) {
	var claims jwt.RegisteredClaims

	_, _, err := (&jwt.Parser{}).ParseUnverified(token, &claims)
	if err != nil && !errors.Is(err, jwt.ErrTokenUnverifiable) {
		return claims, fmt.Errorf("could not parse the API token: %w", err)
	}

	return claims, nil
}

// FromAPIToken creates a session from a ready API token.
func FromAPIToken(_ context.Context, client *http.Client) func(string, string) (Session, error) {
	return func(endpoint, token string) (Session, error) {
		claims, err := parseUnverifiedClaims(token)
		if err != nil {
			return nil, err
		}

		if len(claims.Audience) != 1 {
			return nil, fmt.Errorf("unexpected audience: %v", claims.Audience)
		}

		apiEndpoint := claims.Audience[0]
		if endpoint != "" {
			apiEndpoint = endpoint
		}

		var validUntil time.Time
		if claims.ExpiresAt != nil {
			validUntil = claims.ExpiresAt.Time
		}

		return &apiToken{
			client:          client,
			endpoint:        apiEndpoint,
			jwt:             token,
			tokenValidUntil: validUntil,
			timer:           time.Now,
		}, nil
	}
}

type apiToken struct {
	client          *http.Client
	endpoint        string
	jwt             string
	tokenMutex      sync.RWMutex
	tokenValidUntil time.Time
	timer           func() time.Time
}

func (a *apiToken) BearerToken(ctx context.Context) (string, error) {
	a.tokenMutex.RLock()
	defer a.tokenMutex.RUnlock()

	return a.jwt, nil
}

func (a *apiToken) Type() CredentialsType {
	return CredentialsTypeAPIToken
}

func (a *apiToken) Endpoint() string {
	return strings.TrimRight(a.endpoint, "/") + "/graphql"
}

func (a *apiToken) isFresh() bool {
	a.tokenMutex.RLock()
	defer a.tokenMutex.RUnlock()

	return !tokenExpired(a.tokenValidUntil, a.timer())
}

func tokenExpired(validUntil, now time.Time) bool {
	return !now.Add(timePadding).Before(validUntil)
}

func (a *apiToken) mutate(ctx context.Context, m any, variables map[string]any) error {
	return graphql.NewClient(a.Endpoint(), a.client).Mutate(ctx, m, variables)
}

func (a *apiToken) setJWT(user *user) {
	a.tokenMutex.Lock()
	defer a.tokenMutex.Unlock()

	a.jwt = user.JWT
	a.tokenValidUntil = time.Unix(user.ValidUntil, 0)
}
