package security

import "testing"

func TestHashToken(t *testing.T) {
	got1, err := HashToken("refresh-token-value", "secret-key")
	if err != nil {
		t.Fatalf("HashToken() error = %v", err)
	}
	got2, err := HashToken("refresh-token-value", "secret-key")
	if err != nil {
		t.Fatalf("HashToken() error = %v", err)
	}

	if got1 == "" {
		t.Fatalf("HashToken() returned empty hash")
	}
	if got1 != got2 {
		t.Fatalf("HashToken() should be deterministic, got %q and %q", got1, got2)
	}
}
