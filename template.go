package jinja

import (
	"fmt"
)

// Template is a parsed Jinja2 template ready for rendering.
type Template struct {
	env       *Environment
	name      string
	source    string
	tree      []Node
	parent    *Template
	resolved  bool
}

// Render renders the template with the given data and returns the result.
func (t *Template) Render(data interface{}) (string, error) {
	ctx := NewExecutionContext(t.env, data)
	return t.renderWithContext(ctx)
}

// renderWithContext renders the template using an existing context (for internal use).
func (t *Template) renderWithContext(ctx *ExecutionContext) (string, error) {
	var buf stringsWrapper
	err := t.treeExecute(ctx, &buf)
	if err != nil {
		return "", fmt.Errorf("jinja: render error in %s: %w", t.name, err)
	}
	return buf.String(), nil
}

// treeExecute executes all nodes in the tree.
func (t *Template) treeExecute(ctx *ExecutionContext, buf *stringsWrapper) error {
	return executeNodes(ctx, buf, t.tree)
}

// stringsWrapper is a simple string builder.
type stringsWrapper struct {
	buf []byte
}

func (w *stringsWrapper) WriteString(s string) {
	w.buf = append(w.buf, s...)
}

func (w *stringsWrapper) String() string {
	return string(w.buf)
}


