package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=value\nFOO=bar\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "value")
	}
	if vars["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", vars["FOO"], "bar")
	}
	if len(vars) != 2 {
		t.Errorf("got %d vars, want 2", len(vars))
	}
}

func TestLoadQuoted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte(`KEY="hello world"`+"\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "hello world" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "hello world")
	}
}

func TestLoadCommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("# this is a comment\n\nKEY=val\n# another comment\n\nFOO=bar\n")
	os.WriteFile(path, content, 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "val" || vars["FOO"] != "bar" || len(vars) != 2 {
		t.Errorf("unexpected vars: %v", vars)
	}
}

func TestLoadDuplicateKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=first\nKEY=second\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "second" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "second")
	}
}

func TestLoadEmptyValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "" {
		t.Errorf("KEY = %q, want empty", vars["KEY"])
	}
}

func TestLoadMalformedLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=val\nbadline\nFOO=bar\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "val" || vars["FOO"] != "bar" || len(vars) != 2 {
		t.Errorf("unexpected vars: %v", vars)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
