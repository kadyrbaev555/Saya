package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileAndEnvironmentPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if e := os.WriteFile(path, []byte("SAYA_TEST_VALUE='quoted # value'\nSAYA_TEST_BOOL=false\n"), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("ENV_FILE", path)
	r, e := Open()
	if e != nil {
		t.Fatal(e)
	}
	if r.String("SAYA_TEST_VALUE", "") != "quoted # value" {
		t.Fatal("quoted value lost")
	}
	if _, ok := os.LookupEnv("SAYA_TEST_VALUE"); ok {
		t.Fatal("loader mutated environment")
	}
	t.Setenv("SAYA_TEST_VALUE", "override")
	if r.Required("SAYA_TEST_VALUE") != "override" {
		t.Fatal("environment must override")
	}
	t.Setenv("SAYA_TEST_VALUE", "")
	r.Required("SAYA_TEST_VALUE")
	if r.Err() == nil {
		t.Fatal("explicit empty must fail required")
	}
}
func TestErrorsDoNotExposeValues(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing"))
	if _, e := Open(); e == nil {
		t.Fatal("explicit missing file accepted")
	}
	r := &Reader{values: map[string]string{"URL": "secret://private", "BOOL": "secret", "COUNT": "secret"}}
	r.URL("URL", "https")
	r.Bool("BOOL", "true")
	r.Uint("COUNT", "1", 1, 8)
	if r.Err() == nil || strings.Contains(r.Err().Error(), "secret") {
		t.Fatal("unsafe validation error")
	}
}
func TestMalformedDotenvIsRedacted(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("KEY='super-secret"), 0600)
	t.Setenv("ENV_FILE", path)
	if _, e := Open(); e == nil || strings.Contains(e.Error(), "super-secret") {
		t.Fatal("unsafe parse error")
	}
}
func TestTypedValidation(t *testing.T) {
	for _, v := range []string{"0", "9", "-1", "NaN", "18446744073709551616"} {
		r := &Reader{values: map[string]string{"N": v}}
		r.Uint("N", "2", 1, 8)
		if r.Err() == nil {
			t.Fatal("invalid uint accepted")
		}
	}
	r := &Reader{values: map[string]string{"B": "false", "N": "3"}}
	if r.Bool("B", "true") || r.Uint("N", "2", 1, 8) != 3 || r.Err() != nil {
		t.Fatal("valid values rejected")
	}
}
