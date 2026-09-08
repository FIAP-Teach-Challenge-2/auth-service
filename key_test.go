package main

import "testing"

func TestGenerateAPIKey(t *testing.T) {
	key1, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned an error: %v", err)
	}
	if len(key1) == 0 {
		t.Fatal("generateAPIKey returned an empty string")
	}
	if key1[:7] != "tm_key_" {
		t.Fatalf("expected key to start with 'tm_key_', got %q", key1[:7])
	}

	key2, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned an error: %v", err)
	}
	if key1 == key2 {
		t.Fatal("expected two generated keys to be different, got identical keys")
	}
}

func TestHashAPIKey(t *testing.T) {
	hash1 := hashAPIKey("tm_key_abc123")
	hash2 := hashAPIKey("tm_key_abc123")
	if hash1 != hash2 {
		t.Fatalf("expected hashAPIKey to be deterministic, got %q and %q", hash1, hash2)
	}
	if len(hash1) != 64 {
		t.Fatalf("expected a 64-character hex SHA-256 hash, got %d characters", len(hash1))
	}

	hash3 := hashAPIKey("tm_key_different")
	if hash1 == hash3 {
		t.Fatal("expected different inputs to produce different hashes")
	}
}
