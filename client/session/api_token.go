package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/shurcooL/graphql"
)

// If the token is about to expire, we'd rather exchange it now than risk having
// a stale one.
const timePadding = 30 * time.Second

// FromAPIToken creates a session from a ready API token.
func FromAPIToken(_ context.Context, client *http.Client) func(string) (Session, error) {
	return func(token string) (Session, error) {
		var claims jwt.StandardClaims

		_, _, err := (&jwt.Parser{}).ParseUnverified(token, &claims)
		if unverifiable := new(jwt.UnverfiableTokenError); err != nil && !errors.As(err, &unverifiable) {
			return nil, fmt.Errorf("could not parse the API token: %w", err)
		}

		if len(claims.Audience) != 1 {
			return nil, fmt.Errorf("unexpected audience: %v", claims.Audience)
		}

		return &apiToken{
			client:          client,
			endpoint:        claims.Audience[0],
			jwt:             token,
			tokenValidUntil: claims.ExpiresAt.Time,
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

func (a *apiToken) Endpoint() string {
	return strings.TrimRight(a.endpoint, "/") + "/graphql"
}

func (a *apiToken) isFresh() bool {
	a.tokenMutex.RLock()
	defer a.tokenMutex.RUnlock()

	return a.timer().Add(timePadding).Before(a.tokenValidUntil)
}

func (a *apiToken) mutate(ctx context.Context, m interface{}, variables map[string]interface{}) error {
	return graphql.NewClient(a.Endpoint(), a.client).Mutate(ctx, m, variables)
}

func (a *apiToken) setJWT(user *user) {
	a.tokenMutex.Lock()
	defer a.tokenMutex.Unlock()

	a.jwt = user.JWT
	a.tokenValidUntil = time.Unix(user.ValidUntil, 0)
}
