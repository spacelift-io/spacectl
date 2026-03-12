package api

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

type stringSliceArgs struct {
	v []string
}

func (a *stringSliceArgs) Get(n int) string {
	if len(a.v) > n {
		return a.v[n]
	}
	return ""
}

func (a *stringSliceArgs) First() string { return a.Get(0) }

func (a *stringSliceArgs) Tail() []string {
	if len(a.v) <= 1 {
		return []string{}
	}
	ret := make([]string, len(a.v)-1)
	copy(ret, a.v[1:])
	return ret
}

func (a *stringSliceArgs) Len() int { return len(a.v) }

func (a *stringSliceArgs) Present() bool { return len(a.v) > 0 }

func (a *stringSliceArgs) Slice() []string {
	ret := make([]string, len(a.v))
	copy(ret, a.v)
	return ret
}

func TestResolveQueryFromMutualExclusion(t *testing.T) {
	_, err := resolveQueryFrom("query", "file.graphql", bytes.NewBufferString(""), true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveQueryFromQuery(t *testing.T) {
	got, err := resolveQueryFrom("query { viewer { id } }", "", bytes.NewBufferString(""), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "query { viewer { id } }" {
		t.Fatalf("unexpected query: %q", got)
	}
}

func TestResolveQueryFromQueryDash(t *testing.T) {
	got, err := resolveQueryFrom("-", "", bytes.NewBufferString("{ viewer { id } }"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{ viewer { id } }" {
		t.Fatalf("unexpected query: %q", got)
	}
}

func TestResolveQueryFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query.graphql")
	if err := os.WriteFile(path, []byte("{ viewer { id } }"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := resolveQueryFrom("", path, bytes.NewBufferString(""), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "{ viewer { id } }" {
		t.Fatalf("unexpected query: %q", got)
	}
}

func TestResolveQueryFromStdinEmpty(t *testing.T) {
	_, err := resolveQueryFrom("", "", bytes.NewBufferString(""), false)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveQueryFromRequiresQueryWhenTTY(t *testing.T) {
	_, err := resolveQueryFrom("", "", bytes.NewBufferString(""), true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseVariables(t *testing.T) {
	if v, err := parseVariables(""); err != nil || v != nil {
		t.Fatalf("expected nil, got %v, err=%v", v, err)
	}

	if _, err := parseVariables("not-json"); err == nil {
		t.Fatalf("expected error for invalid json")
	}

	if _, err := parseVariables("[]"); err == nil {
		t.Fatalf("expected error for non-object json")
	}

	vars, err := parseVariables("{\"key\":\"value\"}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["key"] != "value" {
		t.Fatalf("unexpected variables: %v", vars)
	}
}

func TestValidateSchemaArgs(t *testing.T) {
	if err := validateSchemaArgs(false, "", "", "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateSchemaArgs(true, "", "", "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateSchemaArgs(true, "query", "", "", "", ""); err == nil {
		t.Fatalf("expected error for query with schema")
	}
}

func TestGraphqlErrorMessage(t *testing.T) {
	if _, ok := graphqlErrorMessage([]byte(`{"data": {}}`)); ok {
		t.Fatalf("did not expect error")
	}

	msg, ok := graphqlErrorMessage([]byte(`{"errors":[{"message":"bad"}]}`))
	if !ok {
		t.Fatalf("expected error")
	}
	if msg != "bad" {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestSnippet(t *testing.T) {
	s := snippet([]byte("ok"))
	if s != "ok" {
		t.Fatalf("unexpected snippet: %q", s)
	}
}

func TestNormalizeQuery(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"{ viewer { id } }", "{ viewer { id } }"},
		{"query { viewer { id } }", "query { viewer { id } }"},
		{"mutation { viewer { id } }", "mutation { viewer { id } }"},
		{"subscription { viewer { id } }", "subscription { viewer { id } }"},
		{"availableBillingAddons{name,monthlyPrice}", "query { availableBillingAddons{name,monthlyPrice} }"},
	}

	for _, tc := range cases {
		if got := normalizeQuery(tc.in); got != tc.want {
			t.Fatalf("normalizeQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveRequestPartsPositionalQueryVariables(t *testing.T) {
	query, vars, operation, selector, err := resolveRequestParts(&commandWithArgs{
		flags: []cli.Flag{flagQuery, flagFile, flagVariables, flagOperation},
		args:  []string{"stack(id: $stack) { id }"},
		flagValues: map[string]string{
			flagVariables.Name: "{\"stack\":\"my-stack\"}",
		},
	}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selector != "" {
		t.Fatalf("unexpected selector: %q", selector)
	}
	if operation != "" {
		t.Fatalf("unexpected operation: %q", operation)
	}
	if query != "query { stack(id: $stack) { id } }" {
		t.Fatalf("unexpected query: %q", query)
	}
	if vars["stack"] != "my-stack" {
		t.Fatalf("unexpected variables: %v", vars)
	}
}

type commandWithArgs struct {
	cli.Command
	flags      []cli.Flag
	args       []string
	flagValues map[string]string
}

func (c *commandWithArgs) Args() cli.Args {
	return &stringSliceArgs{v: c.args}
}

func (c *commandWithArgs) String(name string) string {
	if v, ok := c.flagValues[name]; ok {
		return v
	}
	return ""
}

func (c *commandWithArgs) Bool(name string) bool {
	return false
}
