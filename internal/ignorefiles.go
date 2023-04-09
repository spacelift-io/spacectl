package internal

import (
	ignore "github.com/sabhiram/go-gitignore"
)

var skipDirs = map[string]struct{}{
	".git": struct{}{},
}

var isIgnoreFileName = map[string]struct{}{
	".gitignore":       struct{}{},
	".terraformignore": struct{}{},
}

var alwaysIgnoreName = map[string]struct{}{
	".git":       struct{}{},
	".terraform": struct{}{},
}

type IgnoreFile interface {
	// Matches returns `true` if the given path matches any of the patterns
	// defined by the ignore file.
	Matches(pathRelativeToIgnoreFile string) bool
}

type gitignoreFile struct {
	instance *ignore.GitIgnore
}

func (gif *gitignoreFile) Matches(pathRelativeToIgnoreFile string) bool {
	return gif.instance.MatchesPath(pathRelativeToIgnoreFile)
}
