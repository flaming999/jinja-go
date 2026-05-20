package jinja

import (
	"fmt"
	"sort"
	"strings"
)

func registerBuiltinGlobals(env *Environment) {
	env.globalFuncs["range"] = globalRange
	env.globalFuncs["lipsum"] = globalLipsum
	env.globalFuncs["dict"] = globalDict
	env.globalFuncs["namespace"] = globalNamespace
	env.globalFuncs["cycler"] = globalCycler
	env.globalFuncs["joiner"] = globalJoiner
	env.globalFuncs["super"] = globalSuper
}

func globalRange(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	var start, stop, step int

	switch len(args) {
	case 1:
		stop = toInt(args[0])
		start = 0
		step = 1
	case 2:
		start = toInt(args[0])
		stop = toInt(args[1])
		step = 1
	case 3:
		start = toInt(args[0])
		stop = toInt(args[1])
		step = toInt(args[2])
	default:
		return nil, fmt.Errorf("range expects 1-3 arguments, got %d", len(args))
	}

	if step == 0 {
		return nil, fmt.Errorf("range step cannot be 0")
	}

	var result []interface{}
	if step > 0 {
		for i := start; i < stop; i += step {
			result = append(result, i)
		}
	} else {
		for i := start; i > stop; i += step {
			result = append(result, i)
		}
	}

	return result, nil
}

func globalLipsum(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	n := 1
	html := true
	min_ := 20
	max_ := 100
	_ = max_
	if len(args) > 0 {
		n = toInt(args[0])
	}
	if len(args) > 1 {
		html = isTruthy(args[1])
	}
	if len(args) > 2 {
		min_ = toInt(args[2])
	}
	if len(args) > 3 {
		max_ = toInt(args[3])
	}

	// Generate lorem ipsum paragraphs
	lorem := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

	var paras []string
	for i := 0; i < n; i++ {
		words := strings.Fields(lorem)
		para := make([]string, min_+i*5)
		for j := range para {
			para[j] = words[j%len(words)]
		}
		paras = append(paras, strings.Join(para, " "))
	}

	result := strings.Join(paras, "\n\n")
	if html {
		return SafeString("<p>" + strings.ReplaceAll(result, "\n\n", "</p>\n<p>") + "</p>"), nil
	}
	return result, nil
}

func globalDict(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	m := make(map[string]interface{})
	for i := 0; i+1 < len(args); i += 2 {
		key := stringify(args[i])
		m[key] = args[i+1]
	}
	return m, nil
}

func globalNamespace(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	return &Namespace{}, nil
}

// Namespace is a simple namespace object for templates.
type Namespace struct {
	attrs map[string]interface{}
}

func (n *Namespace) GetAttr(key string) (interface{}, bool) {
	if n.attrs != nil {
		v, ok := n.attrs[key]
		return v, ok
	}
	return nil, false
}

func (n *Namespace) Set(key string, value interface{}) {
	if n.attrs == nil {
		n.attrs = make(map[string]interface{})
	}
	n.attrs[key] = value
}

// Cycler cycles through a set of values.
type Cycler struct {
	items []interface{}
	idx   int
}

func globalCycler(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	return &Cycler{items: args}, nil
}

func (c *Cycler) Current() interface{} {
	if len(c.items) == 0 {
		return nil
	}
	return c.items[c.idx%len(c.items)]
}

func (c *Cycler) Next() interface{} {
	if len(c.items) == 0 {
		return nil
	}
	c.idx++
	return c.items[c.idx%len(c.items)]
}

func (c *Cycler) Reset() {
	c.idx = 0
}

// Joiner joins strings with a separator, skipping the first.
type Joiner struct {
	sep   string
	first bool
	buf   strings.Builder
}

func globalJoiner(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	sep := ""
	if len(args) > 0 {
		sep = stringify(args[0])
	}
	return &Joiner{sep: sep, first: true}, nil
}

func (j *Joiner) Join(s interface{}) string {
	if !j.first {
		j.buf.WriteString(j.sep)
	}
	j.first = false
	result := stringify(s)
	j.buf.WriteString(result)
	return result
}

func (j *Joiner) String() string {
	return j.buf.String()
}

func globalSuper(ctx *ExecutionContext, args ...interface{}) (interface{}, error) {
	// super() is handled differently in block resolution
	return "", nil
}

// min is a utility function.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sortKeys returns sorted keys of a map.
func sortKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
