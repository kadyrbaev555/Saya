package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMissingRequiredSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if e := os.WriteFile(path, nil, 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("ENV_FILE", path)
	for _, k := range []string{"DATABASE_URL", "RABBIT_URL", "JWT_SEED", "JWT_PUBLIC"} {
		t.Setenv(k, "")
	}
	if _, e := LoadConfig(); e == nil {
		t.Fatal("missing required settings accepted")
	}
}

func TestProviderConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, nil, 0600)
	t.Setenv("ENV_FILE", path)
	t.Setenv("RABBIT_URL", "amqp://localhost:5672/")
	t.Setenv("ASSESSMENT_ADDRESS", "localhost:9000")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_ENDPOINT", "")
	for _, provider := range []string{"gemini", "http", "unknown", ""} {
		t.Setenv("LLM_PROVIDER", provider)
		if _, e := LoadConfig(); e == nil {
			t.Fatalf("%s should fail", provider)
		}
	}
	t.Setenv("LLM_PROVIDER", "mock")
	if _, e := LoadConfig(); e != nil {
		t.Fatal(e)
	}
}
