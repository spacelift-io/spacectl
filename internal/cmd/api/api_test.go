package api

import (
	"testing"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"{ viewer { id } }", "{ viewer { id } }"},
		{"query { viewer { id } }", "query { viewer { id } }"},
		{"mutation { deleteStack(id: \"x\") { id } }", "mutation { deleteStack(id: \"x\") { id } }"},
		{"subscription { runs { id } }", "subscription { runs { id } }"},
		{"stacks { id name }", "query { stacks { id name } }"},
	}
	for _, tc := range tests {
		if got := normalizeQuery(tc.in); got != tc.want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseVariables(t *testing.T) {
	if v, err := parseVariables(""); v != nil || err != nil {
		t.Errorf("empty: got %v, %v", v, err)
	}
	if _, err := parseVariables("not-json"); err == nil {
		t.Error("expected error for invalid json")
	}
	if _, err := parseVariables("[1,2]"); err == nil {
		t.Error("expected error for non-object json")
	}
	v, err := parseVariables(`{"id":"my-stack"}`)
	if err != nil {
		t.Fatal(err)
	}
	if v["id"] != "my-stack" {
		t.Errorf("got %v", v)
	}
}

func TestGraphqlErrors(t *testing.T) {
	if msg := graphqlErrors([]byte(`{"data":{}}`)); msg != "" {
		t.Errorf("expected empty, got %q", msg)
	}
	if msg := graphqlErrors([]byte(`{"errors":[{"message":"bad"}]}`)); msg != "bad" {
		t.Errorf("got %q", msg)
	}
	if msg := graphqlErrors([]byte(`{"errors":[{"message":"a"},{"message":"b"}]}`)); msg != "a; b" {
		t.Errorf("got %q", msg)
	}
}

func TestIsMutation(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"mutation { deleteStack(id: \"x\") { id } }", true},
		{"  Mutation { foo }", true},
		{"query { stacks { id } }", false},
		{"{ viewer { id } }", false},
		{"stacks { id }", false},
	}
	for _, tc := range tests {
		if got := isMutation(tc.in); got != tc.want {
			t.Errorf("isMutation(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if s := truncate([]byte("short"), 100); s != "short" {
		t.Errorf("got %q", s)
	}
	if s := truncate([]byte("abcdef"), 3); s != "abc..." {
		t.Errorf("got %q", s)
	}
}
