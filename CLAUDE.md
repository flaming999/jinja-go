# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a complete Jinja2 template engine implemented in Go. It provides full Jinja2 syntax support including variables, filters, tests, expressions, control structures, template inheritance, macros, and auto-escaping.

## Build and Test Commands

```bash
# Build the package
make build
# or: go build ./...

# Run all tests
make test
# or: go test -v ./... -count=1

# Run a specific test
go test -v -run TestBasicVariable ./...

# Run tests with coverage
make test-cover

# Format code
make fmt

# Run go vet
make vet

# Run linter (requires golangci-lint)
make lint

# Tidy modules
make tidy
```

## Architecture

The template engine follows a classic compilation pipeline:

1. **Lexer** (`lexer.go`) - Tokenizes template source into tokens (tokData, tokVariableBegin, tokBlockBegin, tokName, tokString, tokOperator, etc.)

2. **Parser** (`parser.go`) - Builds an AST from tokens. Parses block tags (`{% if %}`, `{% for %}`, `{% block %}`, etc.) and expressions with proper precedence handling.

3. **AST Nodes** (`nodes.go`) - Defines node types (OutputNode, ExprNode, IfNode, ForNode, BlockNode, etc.) and expression types (LiteralExpr, NameExpr, GetAttrExpr, CallExpr, FilterExpr, etc.)

4. **Runtime** (`runtime.go`) - Executes the AST. `ExecutionContext` holds variable scope, `executeNodes` traverses the tree, `evalExpr` evaluates expressions.

5. **Template** (`template.go`) - Holds a parsed template ready for rendering via `Render(data)`. Also contains package-level functions: `RenderString`, `RenderFile`, `Render`.

6. **Environment** (`environment.go`) - Central configuration hub. Holds filters, tests, globals, loaders, and template cache. Created with `jinja.New(opts...)`.

7. **jinja.go** - Top-level convenience API. Package-level `TemplateFromString`, `TemplateFromFile` functions.

## Key Components

### Entry Points

- `jinja.New(opts...)` - Create an Environment with options
- `env.TemplateFromString(name, src)` - Parse template from string
- `env.TemplateFromFile(name)` - Load and parse from loader
- `jinja.TemplateFromString(name, src, opts...)` - Package-level parse from string
- `jinja.TemplateFromFile(path, opts...)` - Package-level load from file
- `RenderString(src, data)` - Package-level one-shot render (uses default Environment)
- `RenderFile(name, data)` - Package-level render from file
- `tmpl.Render(data)` - Render a parsed template

### Extending the Engine

- `env.AddFilter(name, fn)` - Register custom filter (`FilterFunc` type: `func(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error)`)
- `env.AddTest(name, fn)` - Register custom test (`TestFunc` type: `func(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error)`)
- `env.AddGlobal(name, value)` - Register global variable/function

### Template Loaders (`loader.go`)

- `MemoryLoader` - Templates from a `map[string]string`
- `FileSystemLoader` - Templates from filesystem directories
- `ChoiceLoader` - Tries multiple loaders in order
- `PrefixLoader` - Adds prefix to template names
- `CachedLoader` - Wraps any loader with caching
- `FSLoader` - Loads from `io/fs.FS` (embedded files)
- `FunctionLoader` - Loads via custom function

### Built-in Filters (`filters.go`)

| Category | Filters |
|----------|---------|
| **String** | `upper`, `lower`, `capitalize`, `title`, `trim`, `ltrim`, `rtrim`, `striptags`, `escape`, `e`, `safe`, `forceescape`, `urlencode`, `urlize`, `wordcount`, `truncate`, `wordwrap`, `center`, `ljust`, `rjust`, `zfill`, `indent`, `nl2br`, `replace`, `format` |
| **Collection** | `join`, `first`, `last`, `length`, `count`, `list`, `dict`, `batch`, `slice`, `sort`, `unique`, `reverse`, `min`, `max`, `sum`, `map`, `select`, `reject`, `selectattr`, `rejectattr`, `attr`, `items`, `groupby` |
| **Numeric** | `abs`, `round`, `int`, `float`, `bool` |
| **Type** | `string`, `default`, `d`, `json`, `tojson`, `pprint` |
| **Format** | `filesizeformat`, `date`, `datetime`, `yesno`, `random`, `xmlattr` |

### Built-in Tests (`tests.go`)

| Test | Description |
|------|-------------|
| `defined`, `undefined` | Check if variable is defined |
| `none` | Check if value is nil |
| `string`, `number`, `integer`, `float`, `boolean`, `true`, `false` | Type checks |
| `callable`, `iterable`, `mapping`, `sequence`, `namespace` | Structure checks |
| `even`, `odd` | Numeric parity |
| `divisibleby(n)` | Divisibility check |
| `lower`, `upper` | Case checks |
| `startswith(s)`, `endswith(s)` | String prefix/suffix |
| `in` | Containment check |
| `sameas`, `equalto` | Identity/equality |
| `gt`, `gte`, `lt`, `lte` | Ordering comparisons |
| `all`, `any` | Iterable predicates |
| `escaped`, `filter`, `containment`, `matching`, `subset`, `superset` | Special tests |

### Built-in Globals (`globals.go`)

- `range(start, stop, step)` - Generate range of numbers
- `lipsum(n, html, min, max)` - Generate lorem ipsum paragraphs
- `dict(...)` - Create dictionary from key-value pairs
- `namespace()` - Create namespace object for storing values
- `cycler(...)` - Create value cycler
- `joiner(sep)` - Create joiner utility
- `super()` - Access parent block content

### Configuration Options

```go
jinja.WithLoader(loader)          // Template loader
jinja.WithAutoEscape()            // Enable HTML auto-escaping
jinja.WithAutoEscapeStrategy(s)   // "html", "json", or "none"
jinja.WithTrimBlocks()            // Remove newline after block tags
jinja.WithLstripBlocks()          // Strip leading whitespace from blocks
jinja.WithKeepTrailingNewline()   // Keep trailing newline in output
jinja.WithDelimiters(var, block, comment)  // Custom delimiters
jinja.WithUndefined(s)            // "strict", "chainable", or "undefined" behavior
jinja.WithNewlineSequence(s)      // Newline sequence for joining
```

## Template Inheritance

Template inheritance is resolved during parsing (`environment.go:resolveInheritance`). When a template has `{% extends "parent" %}`, the parser:
1. Loads the parent template
2. Recursively resolves parent's inheritance
3. Collects block definitions from child
4. Overrides parent blocks with child blocks (keeping parent as `super`)

## Expression Parsing

The parser uses precedence climbing for expressions (parser.go:parseExpression → parseTernary → parseOr → parseAnd → parseNot → parseCompare → parseAdd → parseMul → parseUnary → parsePostfix → parsePrimary).

Postfix operators (filters `|`, attribute access `.`, subscript `[]`, calls `()`) are parsed iteratively in `parsePostfix`.

## Testing Patterns

Tests use table-driven patterns with template string + data + expected output. See `jinja_test.go` for examples.

## Error Messages

Error messages follow the format: `jinja: <context>: <error>`. For example:
- `jinja: lexer error in <name>: unclosed tag`
- `jinja: parse error in <name>: unknown block tag "foo"`
- `jinja: render error in <name>: undefined variable 'x'`