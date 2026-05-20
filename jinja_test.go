package jinja

import (
	"strings"
	"testing"
)

func TestBasicVariable(t *testing.T) {
	tmpl, err := New().TemplateFromString("test", "Hello {{ name }}!")
	if err != nil {
		t.Fatal(err)
	}
	result, err := tmpl.Render(map[string]interface{}{"name": "World"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello World!" {
		t.Errorf("got %q, want %q", result, "Hello World!")
	}
}

func TestBasicVariableMap(t *testing.T) {
	tmpl, err := New().TemplateFromString("test", "Hello {{ user.name }}!")
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]interface{}{
		"user": map[string]interface{}{"name": "Alice"},
	}
	result, err := tmpl.Render(data)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello Alice!" {
		t.Errorf("got %q, want %q", result, "Hello Alice!")
	}
}

func TestFilters(t *testing.T) {
	tests := []struct {
		tmpl   string
		data   interface{}
		expect string
	}{
		{"{{ x | upper }}", map[string]interface{}{"x": "hello"}, "HELLO"},
		{"{{ x | lower }}", map[string]interface{}{"x": "HELLO"}, "hello"},
		{"{{ x | trim }}", map[string]interface{}{"x": "  hi  "}, "hi"},
		{"{{ x | length }}", map[string]interface{}{"x": "hello"}, "5"},
		{"{{ x | default('N/A') }}", map[string]interface{}{}, "N/A"},
		{"{{ x | replace('a', 'b') }}", map[string]interface{}{"x": "abc"}, "bbc"},
		{"{{ x | join(', ') }}", map[string]interface{}{"x": []interface{}{"a", "b", "c"}}, "a, b, c"},
		{"{{ x | capitalize }}", map[string]interface{}{"x": "hello world"}, "Hello world"},
		{"{{ items | first }}", map[string]interface{}{"items": []interface{}{1, 2, 3}}, "1"},
		{"{{ items | last }}", map[string]interface{}{"items": []interface{}{1, 2, 3}}, "3"},
		{"{{ x | wordcount }}", map[string]interface{}{"x": "hello world foo"}, "3"},
		{"{{ x | abs }}", map[string]interface{}{"x": -5}, "5"},
		{"{{ x | round(1) }}", map[string]interface{}{"x": 3.14159}, "3.1"},
		{"{{ x | int }}", map[string]interface{}{"x": "42"}, "42"},
		{"{{ x | float }}", map[string]interface{}{"x": "3.14"}, "3.14"},
		{"{{ x | string }}", map[string]interface{}{"x": 42}, "42"},
		{"{{ x | title }}", map[string]interface{}{"x": "hello world"}, "Hello World"},
	}

	for _, tt := range tests {
		tmpl, err := New().TemplateFromString("test", tt.tmpl)
		if err != nil {
			t.Errorf("parse error for %q: %v", tt.tmpl, err)
			continue
		}
		result, err := tmpl.Render(tt.data)
		if err != nil {
			t.Errorf("execute error for %q: %v", tt.tmpl, err)
			continue
		}
		if result != tt.expect {
			t.Errorf("for %q: got %q, want %q", tt.tmpl, result, tt.expect)
		}
	}
}

func TestIfElse(t *testing.T) {
	tests := []struct {
		tmpl   string
		data   interface{}
		expect string
	}{
		{"{% if x %}yes{% endif %}", map[string]interface{}{"x": true}, "yes"},
		{"{% if x %}yes{% endif %}", map[string]interface{}{"x": false}, ""},
		{"{% if x %}yes{% else %}no{% endif %}", map[string]interface{}{"x": false}, "no"},
		{"{% if x %}a{% elif y %}b{% else %}c{% endif %}", map[string]interface{}{"x": false, "y": true}, "b"},
		{"{% if not x %}no{% endif %}", map[string]interface{}{"x": false}, "no"},
		{"{% if x and y %}both{% endif %}", map[string]interface{}{"x": true, "y": true}, "both"},
		{"{% if x or y %}one{% endif %}", map[string]interface{}{"x": false, "y": true}, "one"},
	}

	for _, tt := range tests {
		result, err := RenderString(tt.tmpl, tt.data)
		if err != nil {
			t.Errorf("error for %q: %v", tt.tmpl, err)
			continue
		}
		if result != tt.expect {
			t.Errorf("for %q: got %q, want %q", tt.tmpl, result, tt.expect)
		}
	}
}

func TestForLoop(t *testing.T) {
	tmpl := "{% for item in items %}{{ item }}{% endfor %}"
	result, err := RenderString(tmpl, map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "abc" {
		t.Errorf("got %q", result)
	}
}

func TestForLoopVar(t *testing.T) {
	tmpl := "{% for item in items %}{{ loop.index }}:{{ item }} {% endfor %}"
	result, err := RenderString(tmpl, map[string]interface{}{
		"items": []interface{}{"a", "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "1:a 2:b " {
		t.Errorf("got %q", result)
	}
}

func TestForLoopFirstLast(t *testing.T) {
	tmpl := "{% for item in items %}{% if loop.first %}[{% endif %}{{ item }}{% if loop.last %}]{% endif %}{% endfor %}"
	result, err := RenderString(tmpl, map[string]interface{}{
		"items": []interface{}{1, 2, 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "[123]" {
		t.Errorf("got %q", result)
	}
}

func TestSet(t *testing.T) {
	tmpl := "{% set x = 42 %}{{ x }}"
	result, err := RenderString(tmpl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "42" {
		t.Errorf("got %q", result)
	}
}

func TestOperators(t *testing.T) {
	tests := []struct {
		tmpl   string
		expect string
	}{
		{"{{ 1 + 2 }}", "3"},
		{"{{ 10 - 3 }}", "7"},
		{"{{ 3 * 4 }}", "12"},
		{"{{ 10 // 3 }}", "3"},
		{"{{ 2 ** 3 }}", "8"},
		{"{{ 10 % 3 }}", "1"},
		{"{{ 3 == 3 }}", "True"},
		{"{{ 3 != 4 }}", "True"},
		{"{{ 3 > 2 }}", "True"},
		{"{{ 3 < 2 }}", "False"},
	}

	for _, tt := range tests {
		result, err := RenderString(tt.tmpl, nil)
		if err != nil {
			t.Errorf("error for %q: %v", tt.tmpl, err)
			continue
		}
		if result != tt.expect {
			t.Errorf("for %q: got %q, want %q", tt.tmpl, result, tt.expect)
		}
	}
}

func TestTernary(t *testing.T) {
	result, err := RenderString("{{ 'yes' if x else 'no' }}", map[string]interface{}{"x": true})
	if err != nil {
		t.Fatal(err)
	}
	if result != "yes" {
		t.Errorf("got %q", result)
	}
}

func TestInOperator(t *testing.T) {
	result, err := RenderString("{% if 'a' in items %}yes{% endif %}", map[string]interface{}{
		"items": []interface{}{"a", "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "yes" {
		t.Errorf("got %q", result)
	}
}

func TestRange(t *testing.T) {
	result, err := RenderString("{% for i in range(5) %}{{ i }}{% endfor %}", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "01234" {
		t.Errorf("got %q", result)
	}
}

func TestStringConcat(t *testing.T) {
	result, err := RenderString("{{ 'hello' ~ ' ' ~ 'world' }}", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello world" {
		t.Errorf("got %q", result)
	}
}

func TestMacro(t *testing.T) {
	tmpl := "{% macro greet(name) %}Hello {{ name }}!{% endmacro %}{{ greet('World') }}"
	result, err := RenderString(tmpl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello World!" {
		t.Errorf("got %q", result)
	}
}

func TestComment(t *testing.T) {
	tmpl := "Hello{# this is a comment #} World"
	result, err := RenderString(tmpl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello World" {
		t.Errorf("got %q", result)
	}
}

func TestTests(t *testing.T) {
	tests := []struct {
		tmpl   string
		data   interface{}
		expect string
	}{
		{"{% if x is defined %}yes{% endif %}", map[string]interface{}{"x": 1}, "yes"},
		{"{% if x is undefined %}yes{% endif %}", map[string]interface{}{}, "yes"},
		{"{% if x is none %}yes{% endif %}", map[string]interface{}{"x": nil}, "yes"},
		{"{% if x is string %}yes{% endif %}", map[string]interface{}{"x": "hi"}, "yes"},
		{"{% if x is number %}yes{% endif %}", map[string]interface{}{"x": 42}, "yes"},
		{"{% if 4 is even %}yes{% endif %}", nil, "yes"},
		{"{% if 3 is odd %}yes{% endif %}", nil, "yes"},
		{"{% if x is divisibleby(3) %}yes{% endif %}", map[string]interface{}{"x": 9}, "yes"},
	}

	for _, tt := range tests {
		result, err := RenderString(tt.tmpl, tt.data)
		if err != nil {
			t.Errorf("error for %q: %v", tt.tmpl, err)
			continue
		}
		if result != tt.expect {
			t.Errorf("for %q: got %q, want %q", tt.tmpl, result, tt.expect)
		}
	}
}

func TestIsNot(t *testing.T) {
	result, err := RenderString("{% if x is not none %}yes{% endif %}", map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
	if result != "yes" {
		t.Errorf("got %q", result)
	}
}

func TestStructData(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	result, err := RenderString("{{ p.Name }} is {{ p.Age }}", map[string]interface{}{
		"p": Person{Name: "Alice", Age: 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Alice is 30" {
		t.Errorf("got %q", result)
	}
}

func TestAutoEscape(t *testing.T) {
	env := New(WithAutoEscape())
	tmpl, err := env.TemplateFromString("test", "{{ x }}")
	if err != nil {
		t.Fatal(err)
	}
	result, err := tmpl.Render(map[string]interface{}{"x": "<b>bold</b>"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "&lt;b&gt;bold&lt;/b&gt;" {
		t.Errorf("got %q", result)
	}
}

func TestSafeFilter(t *testing.T) {
	env := New(WithAutoEscape())
	tmpl, err := env.TemplateFromString("test", "{{ x | safe }}")
	if err != nil {
		t.Fatal(err)
	}
	result, err := tmpl.Render(map[string]interface{}{"x": "<b>bold</b>"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "<b>bold</b>" {
		t.Errorf("got %q", result)
	}
}

func TestTemplateInheritance(t *testing.T) {
	loader := NewMemoryLoader(map[string]string{
		"base.html":  "Header|{% block content %}default{% endblock %}|Footer",
		"child.html": "{% extends 'base.html' %}{% block content %}child content{% endblock %}",
	})
	env := New(WithLoader(loader))

	tmpl, err := env.TemplateFromFile("child.html")
	if err != nil {
		t.Fatal(err)
	}
	result, err := tmpl.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Header|child content|Footer" {
		t.Errorf("got %q", result)
	}
}

func TestInclude(t *testing.T) {
	loader := NewMemoryLoader(map[string]string{
		"header.html": "HEADER",
		"main.html":   "{% include 'header.html' %}-BODY",
	})
	env := New(WithLoader(loader))

	tmpl, err := env.TemplateFromFile("main.html")
	if err != nil {
		t.Fatal(err)
	}
	result, err := tmpl.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "HEADER-BODY" {
		t.Errorf("got %q", result)
	}
}

func TestLoopCycle(t *testing.T) {
	tmpl := "{% for i in range(4) %}{{ loop.cycle('a', 'b') }}{% endfor %}"
	result, err := RenderString(tmpl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "abab" {
		t.Errorf("got %q", result)
	}
}

func TestJSONFilter(t *testing.T) {
	result, err := RenderString(
		"{{ data | tojson }}",
		map[string]interface{}{
			"data": map[string]interface{}{"key": "value"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != `{"key":"value"}` {
		t.Errorf("got %q", result)
	}
}

func TestStringSlice(t *testing.T) {
	result, err := RenderString("{{ x[1:3] }}", map[string]interface{}{"x": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "el" {
		t.Errorf("got %q", result)
	}
}

func TestBatchFilter(t *testing.T) {
	result, err := RenderString(
		"{% for batch in items | batch(2) %}[{{ batch | join(',') }}]{% endfor %}",
		map[string]interface{}{
			"items": []interface{}{1, 2, 3, 4, 5},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "[1,2]") {
		t.Errorf("got %q", result)
	}
}
