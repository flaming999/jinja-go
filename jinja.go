// Package jinja is a complete Jinja2 template engine implemented in Go.
package jinja

// Top-level convenience API.
// Most users should use New(), then env.TemplateFromString() or env.TemplateFromFile().

import (
	"os"
	"path/filepath"
	"sync"
)

// New creates a new Environment with sensible defaults.
func New(opts ...Option) *Environment {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return newEnvironment(cfg)
}

var (
	defaultEnv     *Environment
	defaultEnvOnce sync.Once
)

// Default returns the default Environment singleton with default configuration.
// Does not accept options; use New(opts...) for custom configurations.
func Default() *Environment {
	defaultEnvOnce.Do(func() {
		defaultEnv = newEnvironment(defaultConfig())
	})
	return defaultEnv
}


// ==================== Package-level Functions ====================

// TemplateFromString parses and returns a Template from a string.
// Uses Default() singleton when no options provided; otherwise creates new Environment.
func TemplateFromString(name, src string) (*Template, error) {
	env := Default()
	return env.TemplateFromString(name, src)
}

// TemplateFromFile reads a file and returns a parsed Template.
// Uses the Default() singleton Environment.
func TemplateFromFile(path string) (*Template, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	env := Default()
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	return env.TemplateFromString(filepath.Base(abs), string(data))
}


// RenderString renders a template string with data and returns the result.
// Uses the Default() singleton Environment.
func RenderString(src string, data interface{}) (string, error) {
	t, err := Default().TemplateFromString("<string>", src)
	if err != nil {
		return "", err
	}
	return t.Render(data)
}

// RenderFile renders a template file with data.
// Uses the Default() singleton Environment.
func RenderFile(name string, data interface{}) (string, error) {
	t, err := Default().TemplateFromFile(name)
	if err != nil {
		return "", err
	}
	return t.Render(data)
}

