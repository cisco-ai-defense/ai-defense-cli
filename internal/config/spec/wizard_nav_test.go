package spec

import "testing"

func TestSectionSpec_previousRequiredKey(t *testing.T) {
	sec := RequiredSection()
	keys := sec.RequiredFieldKeys()
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 required keys, got %v", keys)
	}

	prev, ok := sec.previousRequiredKey(keys[0])
	if ok {
		t.Fatalf("first key %q should have no previous", keys[0])
	}
	if prev != "" {
		t.Fatalf("expected empty previous, got %q", prev)
	}

	prev, ok = sec.previousRequiredKey(keys[1])
	if !ok {
		t.Fatalf("expected previous for %q", keys[1])
	}
	if prev != keys[0] {
		t.Fatalf("previous of %q = %q, want %q", keys[1], prev, keys[0])
	}
}
