// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DefaultDevice != "" || len(c.Devices) != 0 {
		t.Fatalf("want empty config, got %+v", c)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want := &Config{
		DefaultDevice: "HUAWEI FreeArc",
		Devices:       map[string]string{"HUAWEI FreeArc": "30:96:10:FD:B6:88"},
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "blue-select", "config.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("perm = %v, want 0644", info.Mode().Perm())
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DefaultDevice != want.DefaultDevice || got.Devices["HUAWEI FreeArc"] != want.Devices["HUAWEI FreeArc"] {
		t.Fatalf("roundtrip mismatch: got %+v", got)
	}
}
