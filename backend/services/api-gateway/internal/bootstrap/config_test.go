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
