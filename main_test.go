package main

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spacelift-io/spacectl/client"
)

func TestInstanceTypeWarning_ProxyAuthFailureIsExplained(t *testing.T) {
	// The transport error surfaces wrapped by net/http and the GraphQL client.
	cause := &client.ProxyAuthError{Err: errors.New(`command "/usr/local/bin/proxy-auth" failed: exit status 1`)}
	err := fmt.Errorf("error executing request: %w", &url.Error{Op: "Post", URL: "https://spacelift.example.com/graphql", Err: cause})

	warning := instanceTypeWarning(err)
	assert.Contains(t, warning, "Unable to determine Spacelift instance type")
	assert.Contains(t, warning, `proxy auth: command "/usr/local/bin/proxy-auth" failed: exit status 1`)
	assert.NotContains(t, warning, "until you authenticate", "a proxy auth failure is not a missing login")
}

func TestInstanceTypeWarning_OtherFailuresSuggestLoggingIn(t *testing.T) {
	for name, err := range map[string]error{
		"no error":     nil,
		"other error":  errors.New("no credentials"),
		"wrapped auth": fmt.Errorf("unauthorized: %w", errors.New("expired")),
	} {
		t.Run(name, func(t *testing.T) {
			warning := instanceTypeWarning(err)
			assert.Contains(t, warning, "until you authenticate with Spacelift")
			assert.NotContains(t, warning, "proxy auth")
		})
	}
}
