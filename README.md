# jinja-go

A complete [Jinja2](https://jinja.palletsprojects.com/) template engine implemented in Go.

## Features

- Full Jinja2 template syntax support
- Variables, filters, tests, and expressions
- Control structures: `{% if %}`, `{% for %}`, `{% set %}`, `{% macro %}`
- Template inheritance: `{% extends %}`, `{% block %}`
- `{% include %}` for including other templates
- Auto-escaping for HTML and JSON
- Custom filters, tests, and globals
- Multiple template loaders (file system, memory)
- Comprehensive built-in filters (60+)
- Built-in test functions (30+)

## Installation

```bash
go get github.com/flaming999/jinja-go
```

## Quickstart

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/flaming999/jinja-go"
)

func main() {
    // Simple one-liner
    result, err := jinja.RenderString("Hello {{ name }}!", map[string]interface{}{
        "name": "World",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(result) // Hello World!
}
```

### Parse and Render

```go
env := jinja.New()

// Parse template from string
tmpl, err := env.TemplateFromString("greeting", "Hello {{ user.name }}!")
if err != nil {
    panic(err)
}

// Render with data
data := map[string]interface{}{
    "user": map[string]interface{}{
        "name": "Alice",
    },
}
result, err := tmpl.Render(data)
fmt.Println(result) // Hello Alice!
```

### Load from File

```go
// Load template from file system
loader := jinja.NewFileSystemLoader("./templates")
env := jinja.New(jinja.WithLoader(loader))

tmpl, err := env.TemplateFromFile("index.html")
if err != nil {
    panic(err)
}
result, err := tmpl.Render(map[string]interface{}{
    "title": "Welcome",
})
```

### Using Filters

```go
tmpl := "{{ items | join(', ') | upper }}"
result, _ := jinja.RenderString(tmpl, map[string]interface{}{
    "items": []interface{}{"apple", "banana", "cherry"},
})
fmt.Println(result) // APPLE, BANANA, CHERRY
```

### Conditionals and Loops

```go
tmpl := `{% for item in items %}
{% if loop.first %}First: {% endif %}{{ loop.index }}. {{ item }}
{% endfor %}`

result, _ := jinja.RenderString(tmpl, map[string]interface{}{
    "items": []interface{}{"a", "b", "c"},
})
```

### Template Inheritance

```go
// base.html
base := `<html>
<head><title>{% block title %}Default{% endblock %}</title></head>
<body>{% block content %}{% endblock %}</body>
</html>`

// child.html
child := `{% extends 'base.html' %}
{% block title %}My Page{% endblock %}
{% block content %}Hello World{% endblock %}`

loader := jinja.NewMemoryLoader(map[string]string{
    "base.html":  base,
    "child.html": child,
})
env := jinja.New(jinja.WithLoader(loader))

result, _ := env.TemplateFromFile("child.html").Render(nil)
// Output: <html><head><title>My Page</title></head><body>Hello World</body></html>
```

### Auto-Escaping

```go
env := jinja.New(jinja.WithAutoEscape())
tmpl, _ := env.TemplateFromString("test", "{{ content }}")
result, _ := tmpl.Render(map[string]interface{}{
    "content": "<script>alert('xss')</script>",
})
fmt.Println(result)
// &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;

// Use |safe filter to bypass escaping
tmpl2, _ := env.TemplateFromString("test", "{{ content | safe }}")
result2, _ := tmpl2.Render(map[string]interface{}{
    "content": "<b>bold</b>",
})
fmt.Println(result2) // <b>bold</b>
```

### Custom Filters

```go
env := jinja.New()

// Add custom filter
env.AddFilter("reverse", func(ctx *jinja.ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
    s := fmt.Sprintf("%v", v)
    runes := []rune(s)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return string(runes), nil
})

tmpl, _ := env.TemplateFromString("test", "{{ name | reverse }}")
result, _ := tmpl.Render(map[string]interface{}{
    "name": "hello",
})
fmt.Println(result) // olleh
```

### Custom Tests

```go
env := jinja.New()

// Add custom test: is_admin
env.AddTest("admin", func(ctx *jinja.ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
    if m, ok := v.(map[string]interface{}); ok {
        role, _ := m["role"].(string)
        return role == "admin", nil
    }
    return false, nil
})

tmpl := `{% if user is admin %}Admin Panel{% else %}Regular User{% endif %}`
tmpl2, _ := env.TemplateFromString("test", tmpl)
result, _ := tmpl2.Render(map[string]interface{}{
    "user": map[string]interface{}{"role": "admin"},
})
fmt.Println(result) // Admin Panel
```

### Custom Globals

```go
env := jinja.New()

// Add global function
env.AddGlobal("now", func() string {
    return time.Now().Format("2006-01-02")
})

tmpl, _ := env.TemplateFromString("test", "Today: {{ now() }}")
result, _ := tmpl.Render(nil)
fmt.Println(result) // Today: 2024-01-15
```

## Configuration Options

```go
env := jinja.New(
    jinja.WithLoader(jinja.NewFileSystemLoader("./templates")),
    jinja.WithAutoEscape(),                    // Enable HTML auto-escaping
    jinja.WithAutoEscapeStrategy("json"),      // Use JSON escaping
    jinja.WithTrimBlocks(),                    // Remove first newline after blocks
    jinja.WithLstripBlocks(),                  // Strip leading whitespace from block lines
    jinja.WithKeepTrailingNewline(),           // Keep trailing newline
    jinja.WithDelimiters("${", "}", "{#"),     // Custom delimiters
    jinja.WithUndefined("strict"),             // Strict undefined behavior (errors on missing vars)
)
```

## Built-in Filters

| Category | Filters |
|----------|---------|
| **String** | `upper`, `lower`, `capitalize`, `title`, `trim`, `ltrim`, `rtrim`, `striptags`, `escape`, `safe`, `urlencode`, `wordcount`, `truncate`, `wordwrap`, `center`, `ljust`, `rjust`, `zfill`, `indent`, `nl2br`, `replace` |
| **Collection** | `join`, `first`, `last`, `length`, `count`, `list`, `dict`, `batch`, `slice`, `sort`, `unique`, `min`, `max`, `sum`, `map`, `select`, `reject`, `selectattr`, `rejectattr`, `attr`, `items`, `reverse`, `groupby` |
| **Numeric** | `abs`, `round`, `int`, `float`, `bool` |
| **Type** | `string`, `default`, `json`, `tojson`, `pprint` |
| **Format** | `filesizeformat`, `date`, `datetime`, `yesno` |
| **Other** | `random`, `format`, `xmlattr` |

## Built-in Tests

| Test | Description |
|------|-------------|
| `defined`, `undefined` | Check if variable is defined |
| `none` | Check if value is nil |
| `string`, `number`, `integer`, `float`, `boolean` | Type checks |
| `callable`, `iterable`, `mapping`, `sequence` | Structure checks |
| `even`, `odd` | Numeric parity |
| `divisibleby(n)` | Divisibility check |
| `lower`, `upper` | Case checks |
| `startswith(s)`, `endswith(s)` | String prefix/suffix |
| `in`, `equalto`, `sameas` | Comparison |
| `gt`, `gte`, `lt`, `lte` | Ordering |

## Template Syntax

### Variables
```jinja
{{ variable }}
{{ object.attribute }}
{{ dict['key'] }}
{{ array[0] }}
{{ array[1:3] }}  # Slice
```

### Filters
```jinja
{{ name | upper }}
{{ name | default('N/A') }}
{{ items | join(', ') | upper }}
{{ price | round(2) }}
```

### Conditionals
```jinja
{% if condition %}
    ...
{% elif other %}
    ...
{% else %}
    ...
{% endif %}

{% if x is defined %}
{% if x is not none %}
{% if x > 0 and y < 10 %}
```

### Loops
```jinja
{% for item in items %}
    {{ loop.index }}      # 1-based index
    {{ loop.index0 }}     # 0-based index
    {{ loop.first }}      # True on first iteration
    {{ loop.last }}       # True on last iteration
    {{ loop.cycle('a', 'b') }}
{% else %}
    No items
{% endfor %}

{% for i in range(5) %}  # 0, 1, 2, 3, 4
{% endfor %}
```

### Assignments
```jinja
{% set x = 42 %}
{% set greeting = 'Hello ' ~ name %}
```

### Macros
```jinja
{% macro input(name, value='') %}
    <input type="text" name="{{ name }}" value="{{ value }}">
{% endmacro %}

{{ input('username', 'admin') }}
```

### Includes
```jinja
{% include 'header.html' %}
```

### Template Inheritance
```jinja
{# base.html #}
{% block title %}Default Title{% endblock %}
{% block content %}{% endblock %}

{# child.html #}
{% extends 'base.html' %}
{% block title %}My Title{% endblock %}
{% block content %}
    {{ super() }}  # Render parent block
    Additional content
{% endblock %}
```

### Operators
```jinja
{{ 1 + 2 }}        # Addition
{{ 10 - 3 }}       # Subtraction
{{ 3 * 4 }}        # Multiplication
{{ 10 // 3 }}      # Integer division
{{ 2 ** 3 }}       # Power
{{ 10 % 3 }}       # Modulo
{{ 'a' ~ 'b' }}    # String concatenation
{{ 'yes' if x else 'no' }}  # Ternary
```

### Comments
```jinja
{# This is a comment #}
```

## Using with Structs

Go structs work seamlessly with field access:

```go
type Person struct {
    Name string
    Age  int
}

tmpl := "{{ p.Name }} is {{ p.Age }} years old"
result, _ := jinja.RenderString(tmpl, map[string]interface{}{
    "p": Person{Name: "Alice", Age: 30},
})
// Alice is 30 years old
```

JSON tags are also supported:

```go
type User struct {
    Name string `json:"username"`
}

tmpl := "{{ user.username }}"
result, _ := jinja.RenderString(tmpl, map[string]interface{}{
    "user": User{Name: "bob"},
})
// bob
```

## License

MIT License