package stack

import "testing"

func Test_cleanupRepositoryString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://github.com/username/tftest.git", "username/tftest"},
		{"git@github.com:username/spacelift-local.git", "username/spacelift-local"},
		{"ssh://git@github.com/username/tftest.git", "username/tftest"},
		{"https://gitlab.com/username/project.git", "username/project"},
		{"git@gitlab.com:username/project.git", "username/project"},
		{"ssh://git@gitlab.com/username/tftest.git", "username/tftest"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, _ := cleanupRepositoryString(test.input)
			if result != test.expected {
				t.Errorf("expected %q, got %q", test.expected, result)
			}
		})
	}
}
