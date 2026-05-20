package jinja

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Loader is the interface for loading templates from a source.
type Loader interface {
	Load(name string) (string, error)
}

// MemoryLoader loads templates from an in-memory map.
type MemoryLoader struct {
	templates map[string]string
	mu        sync.RWMutex
}

// NewMemoryLoader creates a new MemoryLoader.
func NewMemoryLoader(templates map[string]string) *MemoryLoader {
	if templates == nil {
		templates = make(map[string]string)
	}
	return &MemoryLoader{templates: templates}
}

// Load returns the template source by name.
func (l *MemoryLoader) Load(name string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	src, ok := l.templates[name]
	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}
	return src, nil
}

// AddTemplate adds or updates a template in memory.
func (l *MemoryLoader) AddTemplate(name, src string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.templates[name] = src
}

// FileSystemLoader loads templates from the filesystem.
type FileSystemLoader struct {
	dirs      []string
	extensions []string
	followSym bool
}

// NewFileSystemLoader creates a new FileSystemLoader that searches the given directories.
func NewFileSystemLoader(dirs ...string) *FileSystemLoader {
	return &FileSystemLoader{
		dirs:      dirs,
		extensions: []string{".html", ".htm", ".txt", ".xml", ".j2", ".jinja", ".jinja2"},
	}
}

// Load searches for and loads a template by name from the filesystem.
func (l *FileSystemLoader) Load(name string) (string, error) {
	// Try exact path first
	for _, dir := range l.dirs {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}

	// Try with extensions
	for _, dir := range l.dirs {
		for _, ext := range l.extensions {
			path := filepath.Join(dir, name+ext)
			data, err := os.ReadFile(path)
			if err == nil {
				return string(data), nil
			}
		}
	}

	return "", fmt.Errorf("template %q not found in %v", name, l.dirs)
}

// ChoiceLoader tries multiple loaders in order.
type ChoiceLoader struct {
	loaders []Loader
}

// NewChoiceLoader creates a loader that tries each loader in order.
func NewChoiceLoader(loaders ...Loader) *ChoiceLoader {
	return &ChoiceLoader{loaders: loaders}
}

// Load tries each loader until one succeeds.
func (l *ChoiceLoader) Load(name string) (string, error) {
	var lastErr error
	for _, loader := range l.loaders {
		src, err := loader.Load(name)
		if err == nil {
			return src, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("template %q not found in any loader: %w", name, lastErr)
}

// PrefixLoader adds a prefix to template names before loading.
type PrefixLoader struct {
	prefix string
	loader Loader
}

// NewPrefixLoader creates a loader that prepends a prefix to template names.
func NewPrefixLoader(prefix string, loader Loader) *PrefixLoader {
	return &PrefixLoader{prefix: prefix, loader: loader}
}

// Load loads a template with the prefix prepended.
func (l *PrefixLoader) Load(name string) (string, error) {
	return l.loader.Load(l.prefix + name)
}

// CachedLoader wraps a loader with caching.
type CachedLoader struct {
	loader Loader
	cache  map[string]string
	mu     sync.RWMutex
}

// NewCachedLoader creates a cached loader.
func NewCachedLoader(loader Loader) *CachedLoader {
	return &CachedLoader{
		loader: loader,
		cache:  make(map[string]string),
	}
}

// Load returns a cached template or loads and caches it.
func (l *CachedLoader) Load(name string) (string, error) {
	l.mu.RLock()
	if src, ok := l.cache[name]; ok {
		l.mu.RUnlock()
		return src, nil
	}
	l.mu.RUnlock()

	src, err := l.loader.Load(name)
	if err != nil {
		return "", err
	}

	l.mu.Lock()
	l.cache[name] = src
	l.mu.Unlock()

	return src, nil
}

// FSLoader is an alias for FileSystemLoader using io/fs.
type FSLoader struct {
	fsys fs.FS
	dir  string
}

// NewFSLoader creates a loader from an fs.FS.
func NewFSLoader(fsys fs.FS, dir string) *FSLoader {
	return &FSLoader{fsys: fsys, dir: dir}
}

// Load reads a template from the embedded filesystem.
func (l *FSLoader) Load(name string) (string, error) {
	path := filepath.Join(l.dir, name)
	data, err := fs.ReadFile(l.fsys, path)
	if err == nil {
		return string(data), nil
	}

	// Try with common extensions
	for _, ext := range []string{".html", ".htm", ".txt", ".xml", ".j2", ".jinja2"} {
		data, err = fs.ReadFile(l.fsys, path+ext)
		if err == nil {
			return string(data), nil
		}
	}

	return "", fmt.Errorf("template %q not found", name)
}

// FunctionLoader loads templates from a function.
type FunctionLoader struct {
	fn func(name string) (string, error)
}

// NewFunctionLoader creates a loader from a function.
func NewFunctionLoader(fn func(string) (string, error)) *FunctionLoader {
	return &FunctionLoader{fn: fn}
}

// Load calls the function to get template source.
func (l *FunctionLoader) Load(name string) (string, error) {
	return l.fn(name)
}

// init handles import for "strings" used in loader.
var _ = strings.TrimSpace
