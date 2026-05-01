package variable

import (
	"regexp"
)

var varRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// Pool manages a set of named variables and provides template substitution
type Pool struct {
	store map[string]string
}

// NewPool creates a new empty variable pool
func NewPool() *Pool {
	return &Pool{store: make(map[string]string)}
}

// Set stores a variable value
func (p *Pool) Set(name, value string) {
	p.store[name] = value
}

// Get retrieves a variable value. Returns false if not found.
func (p *Pool) Get(name string) (string, bool) {
	v, ok := p.store[name]
	return v, ok
}

// Replace substitutes all ${var} templates in input with their values.
// Undefined variables are left as-is.
func (p *Pool) Replace(input string) string {
	return varRegex.ReplaceAllStringFunc(input, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		if val, ok := p.store[name]; ok {
			return val
		}
		return match
	})
}
