package variable

import "testing"

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
