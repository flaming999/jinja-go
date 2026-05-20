package jinja

import (
	"fmt"
	"html/template"
	"strings"
	"sync"
)

// Environment holds configuration and caches for template rendering.
type Environment struct {
	config        config
	loader        Loader
	filterFuncs   map[string]FilterFunc
	testFuncs     map[string]TestFunc
	globalFuncs   map[string]interface{}
	extensions    map[string]ExtensionFunc
	mu            sync.RWMutex
	templateCache map[string]*Template
}

func newEnvironment(cfg config) *Environment {
	env := &Environment{
		config:        cfg,
		loader:        cfg.loader,
		filterFuncs:   make(map[string]FilterFunc),
		testFuncs:     make(map[string]TestFunc),
		globalFuncs:   make(map[string]interface{}),
		extensions:    make(map[string]ExtensionFunc),
		templateCache: make(map[string]*Template),
	}
	registerBuiltinFilters(env)
	registerBuiltinTests(env)
	registerBuiltinGlobals(env)
	return env
}

// Option configures an Environment.
type Option func(*config)

type config struct {
	loader              Loader
	leftDelimiter       string
	rightDelimiter      string
	commentDelimiter    string
	blockDelimiter      string
	variableDelimiter   string
	trimBlocks          bool
	lstripBlocks        bool
	autoEscape          bool
	autoEscapeStrategy  string // "html", "json", "none"
	undefinedBehavior   string // "strict", "chainable", "undefined"
	keepTrailingNewline bool
	newlineSequence     string
}

func defaultConfig() config {
	return config{
		leftDelimiter:       "{{",
		rightDelimiter:      "}}",
		commentDelimiter:    "{#",
		blockDelimiter:      "{%",
		variableDelimiter:   "{{",
		trimBlocks:          false,
		lstripBlocks:        false,
		autoEscape:          false,
		autoEscapeStrategy:  "html",
		undefinedBehavior:   "undefined",
		keepTrailingNewline: false,
		newlineSequence:     "\n",
		loader:              NewMemoryLoader(nil),
	}
}

// WithLoader sets the template loader.
func WithLoader(l Loader) Option {
	return func(c *config) { c.loader = l }
}

// WithAutoEscape enables HTML auto-escaping.
func WithAutoEscape() Option {
	return func(c *config) { c.autoEscape = true }
}

// WithAutoEscapeStrategy sets the auto-escape strategy ("html", "json", "none").
func WithAutoEscapeStrategy(s string) Option {
	return func(c *config) {
		c.autoEscape = s != "none"
		c.autoEscapeStrategy = s
	}
}

// WithTrimBlocks removes the first newline after a block tag.
func WithTrimBlocks() Option {
	return func(c *config) { c.trimBlocks = true }
}

// WithLstripBlocks strips leading whitespace from block lines.
func WithLstripBlocks() Option {
	return func(c *config) { c.lstripBlocks = true }
}

// WithDelimiters sets the delimiters for variable, block, and comment tags.
func WithDelimiters(variable, block, comment string) Option {
	return func(c *config) {
		c.variableDelimiter = variable
		c.blockDelimiter = block
		c.commentDelimiter = comment
		c.leftDelimiter = variable
		c.rightDelimiter = variable
	}
}

// WithKeepTrailingNewline preserves trailing newline in templates.
func WithKeepTrailingNewline() Option {
	return func(c *config) { c.keepTrailingNewline = true }
}

// WithUndefined sets the undefined variable behavior ("strict", "chainable", "undefined").
func WithUndefined(b string) Option {
	return func(c *config) { c.undefinedBehavior = b }
}

// WithNewlineSequence sets the newline sequence used for template joining.
func WithNewlineSequence(s string) Option {
	return func(c *config) { c.newlineSequence = s }
}

// TemplateFromString parses a template from a string.
func (env *Environment) TemplateFromString(name, src string) (*Template, error) {
	return env.templateFromString(name, src, false)
}

// templateFromString is the internal parse method, skipLock avoids deadlock during inheritance resolution.
func (env *Environment) templateFromString(name, src string, skipLock bool) (*Template, error) {
	if !skipLock {
		env.mu.Lock()
		defer env.mu.Unlock()
	}

	if t, ok := env.templateCache[name]; ok && t.source == src {
		return t, nil
	}

	tmpl, err := env.parse(name, src)
	if err != nil {
		return nil, err
	}
	env.templateCache[name] = tmpl
	return tmpl, nil
}

// TemplateFromFile loads and parses a template from the loader by name.
func (env *Environment) TemplateFromFile(name string) (*Template, error) {
	src, err := env.loader.Load(name)
	if err != nil {
		return nil, fmt.Errorf("jinja: cannot load template %q: %w", name, err)
	}
	return env.TemplateFromString(name, src)
}

// AddFilter registers a custom filter function.
func (env *Environment) AddFilter(name string, fn FilterFunc) {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.filterFuncs[name] = fn
}

// AddTest registers a custom test function.
func (env *Environment) AddTest(name string, fn TestFunc) {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.testFuncs[name] = fn
}

// AddGlobal registers a global variable/function.
func (env *Environment) AddGlobal(name string, value interface{}) {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.globalFuncs[name] = value
}

// GetFilter returns a filter by name.
func (env *Environment) GetFilter(name string) (FilterFunc, bool) {
	env.mu.RLock()
	defer env.mu.RUnlock()
	f, ok := env.filterFuncs[name]
	return f, ok
}

// GetTest returns a test function by name.
func (env *Environment) GetTest(name string) (TestFunc, bool) {
	env.mu.RLock()
	defer env.mu.RUnlock()
	t, ok := env.testFuncs[name]
	return t, ok
}

// GetGlobal returns a global value by name.
func (env *Environment) GetGlobal(name string) (interface{}, bool) {
	env.mu.RLock()
	defer env.mu.RUnlock()
	v, ok := env.globalFuncs[name]
	return v, ok
}

// ClearCache clears the template cache.
func (env *Environment) ClearCache() {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.templateCache = make(map[string]*Template)
}

// FilterFunc is the type for template filter functions.
// The first argument is the value to filter, rest are additional args.
type FilterFunc func(e *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error)

// TestFunc is the type for template test functions (used in {% if x is testname %}).
type TestFunc func(e *ExecutionContext, v interface{}, args ...interface{}) (bool, error)

// ExtensionFunc allows extending the environment.
type ExtensionFunc func(env *Environment)

// autoEscape applies the configured auto-escape strategy.
func (env *Environment) autoEscape(s string) string {
	if !env.config.autoEscape {
		return s
	}
	switch env.config.autoEscapeStrategy {
	case "html":
		return template.HTMLEscapeString(s)
	case "json":
		return template.JSEscapeString(s)
	default:
		return s
	}
}

// parse performs the full parse pipeline: lex → parse → resolve inheritance.
func (env *Environment) parse(name, src string) (*Template, error) {
	l := newLexer(src, lexerConfig{
		leftDelim:  env.config.variableDelimiter,
		rightDelim: env.config.variableDelimiter,
		blockDelim: env.config.blockDelimiter,
		commentDelim: env.config.commentDelimiter,
		trimBlocks: env.config.trimBlocks,
		lstripBlocks: env.config.lstripBlocks,
	})

	tokens, err := l.tokenize()
	if err != nil {
		return nil, fmt.Errorf("jinja: lexer error in %s: %w", name, err)
	}

	p := newParser(tokens, parserConfig{
		env: env,
	})

	tree, err := p.parse()
	if err != nil {
		return nil, fmt.Errorf("jinja: parse error in %s: %w", name, err)
	}

	tmpl := &Template{
		env:    env,
		name:   name,
		source: src,
		tree:   tree,
	}

	// Resolve template inheritance
	if err := resolveInheritance(tmpl, env); err != nil {
		return nil, err
	}

	return tmpl, nil
}

// resolveInheritance resolves extends/block inheritance chains.
func resolveInheritance(tmpl *Template, env *Environment) error {
	if tmpl.resolved {
		return nil
	}

	// Collect all nodes that have extends
	extendsNode := findExtends(tmpl.tree)
	if extendsNode == nil {
		tmpl.resolved = true
		return nil
	}

	// Load parent template
	parentSrc, err := env.loader.Load(extendsNode.templateName)
	if err != nil {
		return fmt.Errorf("jinja: cannot load parent template %q: %w", extendsNode.templateName, err)
	}

	parent, err := env.templateFromString(extendsNode.templateName, parentSrc, true)
	if err != nil {
		return fmt.Errorf("jinja: error parsing parent template %q: %w", extendsNode.templateName, err)
	}

	// Resolve parent first
	if err := resolveInheritance(parent, env); err != nil {
		return err
	}

	// Collect block definitions from child
	childBlocks := collectBlocks(tmpl.tree)

	// Override parent blocks with child blocks
	overrideBlocks(parent.tree, childBlocks)

	tmpl.tree = parent.tree
	tmpl.parent = parent
	tmpl.resolved = true
	return nil
}

// findExtends finds the first ExtendsNode in the tree.
func findExtends(tree []Node) *ExtendsNode {
	for _, n := range tree {
		switch node := n.(type) {
		case *ExtendsNode:
			return node
		}
	}
	return nil
}

// collectBlocks collects all BlockNode definitions from the tree.
func collectBlocks(tree []Node) map[string]*BlockNode {
	blocks := make(map[string]*BlockNode)
	for _, n := range tree {
		collectBlocksFrom(n, blocks)
	}
	return blocks
}

func collectBlocksFrom(n Node, blocks map[string]*BlockNode) {
	switch node := n.(type) {
	case *BlockNode:
		blocks[node.name] = node
		// Check nested blocks inside this block
		for _, child := range node.body {
			collectBlocksFrom(child, blocks)
		}
	case *IfNode:
		for _, child := range node.body {
			collectBlocksFrom(child, blocks)
		}
		for _, elif := range node.elifBranches {
			for _, child := range elif.body {
				collectBlocksFrom(child, blocks)
			}
		}
		for _, child := range node.elseBody {
			collectBlocksFrom(child, blocks)
		}
	case *ForNode:
		for _, child := range node.body {
			collectBlocksFrom(child, blocks)
		}
		for _, child := range node.elseBody {
			collectBlocksFrom(child, blocks)
		}
	case *MacroNode:
		// Skip macro bodies - blocks inside macros don't count
	}
}

// overrideBlocks replaces block nodes in the parent tree with child overrides.
func overrideBlocks(tree []Node, childBlocks map[string]*BlockNode) {
	for i, n := range tree {
		switch node := n.(type) {
		case *BlockNode:
			if child, ok := childBlocks[node.name]; ok {
				// Replace with child block, but keep parent block as super
				child.super = node
				tree[i] = child
			}
			// Recurse into block body
			overrideBlocks(node.body, childBlocks)
		case *IfNode:
			overrideBlocks(node.body, childBlocks)
			for _, elif := range node.elifBranches {
				overrideBlocks(elif.body, childBlocks)
			}
			overrideBlocks(node.elseBody, childBlocks)
		case *ForNode:
			overrideBlocks(node.body, childBlocks)
			overrideBlocks(node.elseBody, childBlocks)
		}
	}
}

// IsUndefined checks if a value represents an undefined variable.
func IsUndefined(v interface{}) bool {
	if v == nil {
		return true
	}
	_, ok := v.(*UndefinedValue)
	return ok
}

// UndefinedValue represents an undefined template variable.
type UndefinedValue struct {
	Name string
	Msg  string
}

func (u *UndefinedValue) Error() string {
	return u.Msg
}

func (u *UndefinedValue) String() string {
	return u.Msg
}

// SafeString is a string that should not be auto-escaped.
type SafeString string

// String returns the raw string.
func (s SafeString) String() string {
	return string(s)
}

// MarkSafe wraps a string to prevent auto-escaping.
func MarkSafe(s string) SafeString {
	return SafeString(s)
}

// IsSafe checks if a value is a SafeString.
func IsSafe(v interface{}) bool {
	_, ok := v.(SafeString)
	return ok
}

// outputString converts a value to a string, respecting SafeString.
func outputString(env *Environment, v interface{}) string {
	if s, ok := v.(SafeString); ok {
		return string(s)
	}
	s := stringify(v)
	if env != nil {
		return env.autoEscape(s)
	}
	return s
}

// stringify converts any value to a string.
func stringify(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case SafeString:
		return string(val)
	case bool:
		if val {
			return "True"
		}
		return "False"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		s := fmt.Sprintf("%g", val)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	case fmt.Stringer:
		return val.String()
	case error:
		return val.Error()
	default:
		return fmt.Sprintf("%v", val)
	}
}
