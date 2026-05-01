package variable

import (
	"errors"
	"os"
	"regexp"

	"github.com/zend/yamlit/pkg/dotenv"
)

var varRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// Pool manages a set of named variables and provides template substitution
type Pool struct {
	store  map[string]string
	dotenv map[string]string // .env and OS env vars
}

// NewPool creates a new empty variable pool
func NewPool() *Pool {
	return &Pool{
		store:  make(map[string]string),
		dotenv: make(map[string]string),
	}
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

// LoadOSEnv injects all OS environment variables into the pool's dotenv layer.
func (p *Pool) LoadOSEnv() {
	for _, e := range os.Environ() {
		idx := indexByte(e, '=')
		if idx == -1 {
			continue
		}
		p.dotenv[e[:idx]] = e[idx+1:]
	}
}

// LoadDotenv parses a .env file and loads key-value pairs into the pool.
// If path is empty or the file does not exist, it is silently ignored.
func (p *Pool) LoadDotenv(path string) error {
	if path == "" {
		return nil
	}
	vars, err := dotenv.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // silent skip
		}
		return err
	}
	for k, v := range vars {
		p.dotenv[k] = v
	}
	return nil
}

// Replace substitutes all ${var} templates in input with their values.
// Lookup order: store (step-extracted) > dotenv (.env + OS env) > os.Environ
// Undefined variables are left as-is.
func (p *Pool) Replace(input string) string {
	return varRegex.ReplaceAllStringFunc(input, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		// 1. Step-extracted vars (highest priority)
		if val, ok := p.store[name]; ok {
			return val
		}
		// 2. .env / OS env vars (medium priority)
		if val, ok := p.dotenv[name]; ok {
			return val
		}
		// 3. OS environment (lowest priority)
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return match
	})
}

// indexByte finds the first occurrence of c in s.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
