package variable

import (
	"os"
	"testing"
)

func TestReplace(t *testing.T) {
	p := NewPool()
	p.Set("name", "world")
	p.Set("count", "42")

	tests := []struct {
		input    string
		expected string
	}{
		{"hello ${name}", "hello world"},
		{"${name} ${name}", "world world"},
		{"count=${count}", "count=42"},
		{"${undefined}", "${undefined}"},
		{"no vars", "no vars"},
		{"${name}/${count}", "world/42"},
	}

	for _, tt := range tests {
		result := p.Replace(tt.input)
		if result != tt.expected {
			t.Errorf("Replace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGet(t *testing.T) {
	p := NewPool()
	p.Set("key", "value")

	val, ok := p.Get("key")
	if !ok {
		t.Fatal("Get('key') returned false")
	}
	if val != "value" {
		t.Errorf("Get('key') = %q, want %q", val, "value")
	}

	_, ok = p.Get("nonexistent")
	if ok {
		t.Error("Get('nonexistent') should return false")
	}
}

func TestOverwrite(t *testing.T) {
	p := NewPool()
	p.Set("key", "old")
	p.Set("key", "new")

	val, _ := p.Get("key")
	if val != "new" {
		t.Errorf("after overwrite, Get('key') = %q, want %q", val, "new")
	}
}

func TestReplaceWithDotenv(t *testing.T) {
	p := NewPool()
	p.dotenv = map[string]string{"BASE_URL": "https://api.example.com"}

	result := p.Replace("${BASE_URL}/users")
	if result != "https://api.example.com/users" {
		t.Errorf("Replace = %q, want %q", result, "https://api.example.com/users")
	}
}

func TestReplaceWithOSEnv(t *testing.T) {
	t.Setenv("MY_VAR", "from_os")
	p := NewPool()
	p.LoadOSEnv()

	result := p.Replace("prefix-${MY_VAR}-suffix")
	if result != "prefix-from_os-suffix" {
		t.Errorf("Replace = %q, want %q", result, "prefix-from_os-suffix")
	}
}

func TestReplacePriority(t *testing.T) {
	t.Setenv("THE_VAR", "from_os")
	p := NewPool()
	p.dotenv = map[string]string{"THE_VAR": "from_dotenv"}
	p.Set("THE_VAR", "from_store")

	result := p.Replace("${THE_VAR}")
	if result != "from_store" {
		t.Errorf("Replace = %q, want %q", result, "from_store")
	}
}

func TestLoadDotenv(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	os.WriteFile(path, []byte("DB_URL=postgres://localhost\n"), 0644)

	p := NewPool()
	if err := p.LoadDotenv(path); err != nil {
		t.Fatalf("LoadDotenv failed: %v", err)
	}

	result := p.Replace("${DB_URL}")
	if result != "postgres://localhost" {
		t.Errorf("Replace = %q, want %q", result, "postgres://localhost")
	}
}

func TestLoadDotenvFileNotFound(t *testing.T) {
	p := NewPool()
	err := p.LoadDotenv("/nonexistent/.env")
	if err != nil {
		t.Errorf("expected nil for missing file, got %v", err)
	}
}
