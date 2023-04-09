package internal_test

import (
	"testing"

	"github.com/spacelift-io/spacectl/internal"
)

func TestParentDirectory(t *testing.T) {
	type Result struct {
		Path string
		OK   bool
	}

	tt := map[string]*Result{
		"/hello/world/": &Result{
			Path: "/hello",
			OK:   true,
		},
		"/hello/world": &Result{
			Path: "/hello",
			OK:   true,
		},
		"/hello/": &Result{
			Path: "/",
			OK:   true,
		},
		"/hello": &Result{
			Path: "/",
			OK:   true,
		},
		"/": &Result{
			Path: "",
			OK:   false,
		},
		"hello/world": &Result{
			Path: "hello",
			OK:   true,
		},
		"hello": &Result{
			Path: ".",
			OK:   true,
		},
		"./hello": &Result{
			Path: ".",
			OK:   true,
		},

		"../../": &Result{
			Path: "",
			OK:   false,
		},
		"../..": &Result{
			Path: "",
			OK:   false,
		},
		"../": &Result{
			Path: "",
			OK:   false,
		},
		"..": &Result{
			Path: "",
			OK:   false,
		},
		"./": &Result{
			Path: "",
			OK:   false,
		},
		".": &Result{
			Path: "",
			OK:   false,
		},
		"": &Result{
			Path: "",
			OK:   false,
		},
	}

	for input, wantResult := range tt {
		wantPath := wantResult.Path
		wantOK := wantResult.OK

		gotPath, gotOK := internal.ParentDirectory(input)
		if gotPath != wantPath || gotOK != wantOK {
			t.Errorf("internal.ParentDirectory(%q) = (%#v, %#v); want (%#v, %#v)", input, gotPath, gotOK, wantPath, wantOK)
		}
	}
}

func TestPathAncestors_Absolute(t *testing.T) {
	const input = "/hello/world/.gitignore"

	possiblePaths := internal.PathAncestors(input)
	wantPaths := []string{
		"/hello/world",
		"/hello",
		"/",
	}

	if got, want := len(possiblePaths), len(wantPaths); got != want {
		t.Fatalf("len(internal.PathAncestors(%q)) = %d; want %d", input, got, want)
	}

	for i, got := range possiblePaths {
		want := wantPaths[i]

		if got != want {
			t.Errorf("internal.PathAncestors(%q)[%d] = %q; want %q", input, i, got, want)
		}
	}
}

func TestPathAncestors_Relative(t *testing.T) {
	const input = "hello/world/.gitignore"

	possiblePaths := internal.PathAncestors(input)
	wantPaths := []string{
		"hello/world",
		"hello",
		".",
	}

	if got, want := len(possiblePaths), len(wantPaths); got != want {
		t.Fatalf("len(internal.PathAncestors(%q)) = %d; want %d", input, got, want)
	}

	for i, got := range possiblePaths {
		want := wantPaths[i]

		if got != want {
			t.Errorf("internal.PathAncestors(%q)[%d] = %q; want %q", input, i, got, want)
		}
	}
}
