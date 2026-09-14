package stack

import "testing"

func Test_parseRunPriorityPreset(t *testing.T) {
	valid := map[string]string{
		"high":    "high",
		"normal":  "normal",
		"low":     "low",
		"HIGH":    "high",
		"  Low  ": "low",
	}

	for input, expected := range valid {
		t.Run(input, func(t *testing.T) {
			result, err := parseRunPriorityPreset(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil || string(*result) != expected {
				t.Errorf("expected %q, got %v", expected, result)
			}
		})
	}

	for _, input := range []string{"", "urgent", "999", "hi gh"} {
		t.Run("rejects "+input, func(t *testing.T) {
			if _, err := parseRunPriorityPreset(input); err == nil {
				t.Errorf("expected an error for %q", input)
			}
		})
	}
}
