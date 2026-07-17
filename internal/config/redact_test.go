package config

import "testing"

func TestConfigRedacted_RedactsSecrets(t *testing.T) {
	cfg := Config{}
	cfg.GitPassword = "git-password"
	cfg.ApiKey = "api-key"
	cfg.ApiSignature = "api-signature"
	cfg.GitUrl = "https://user:secret@example.com/repository.git"
	cfg.HTTPHeaders = []string{
		"Authorization:Bearer top-secret",
		"Content-Type:application/json",
		"X-API-Key:abc123",
		"X-Signature:sha256=test",
		"just-a-token",
	}

	redacted := cfg.Redacted()

	if redacted.GitPassword != redactedPlaceholder {
		t.Fatalf("expected git password to be redacted, got %q", redacted.GitPassword)
	}
	if redacted.ApiKey != redactedPlaceholder {
		t.Fatalf("expected api key to be redacted, got %q", redacted.ApiKey)
	}
	if redacted.ApiSignature != redactedPlaceholder {
		t.Fatalf("expected api signature to be redacted, got %q", redacted.ApiSignature)
	}
	if redacted.GitUrl != "https://user:%3Credacted%3E@example.com/repository.git" {
		t.Fatalf("expected git url user info to be redacted, got %q", redacted.GitUrl)
	}

	expectedHeaders := []string{
		"Authorization:<redacted>",
		"Content-Type:application/json",
		"X-API-Key:<redacted>",
		"X-Signature:<redacted>",
		"<redacted>",
	}
	for i := range expectedHeaders {
		if redacted.HTTPHeaders[i] != expectedHeaders[i] {
			t.Fatalf("expected header %d to be %q, got %q", i, expectedHeaders[i], redacted.HTTPHeaders[i])
		}
	}

	if cfg.GitPassword != "git-password" {
		t.Fatalf("expected original config to stay unchanged, got %q", cfg.GitPassword)
	}
}
