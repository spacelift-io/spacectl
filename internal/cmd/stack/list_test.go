package stack

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStackJSONHooksAreNested(t *testing.T) {
	s := stack{ID: "stack-1"}
	s.Hooks.AfterApply = []string{"echo done"}

	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshalling the stack failed: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshalling the stack failed: %v", err)
	}

	if _, found := out["afterApply"]; found {
		t.Error("afterApply should no longer be a top-level key")
	}

	hooks, ok := out["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks should be an object but was %v", out["hooks"])
	}

	if got := hooks["afterApply"]; !reflect.DeepEqual(got, []any{"echo done"}) {
		t.Errorf(`hooks.afterApply should be ["echo done"] but was %v`, got)
	}
}
