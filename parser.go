package jinja

import (
	"fmt"
	"strconv"
)

// parserConfig holds parser configuration.
type parserConfig struct {
	env *Environment
}

// parser builds an AST from a token stream.
type parser struct {
	tokens []Token
	pos    int
	env    *Environment
}

func newParser(tokens []Token, cfg parserConfig) *parser {
	return &parser{
		tokens: tokens,
		env:    cfg.env,
	}
}

// parse parses the full template into a list of nodes.
func (p *parser) parse() ([]Node, error) {
	return p.parseBody()
}

// parseBody parses a sequence of nodes until EOF or an end tag.
func (p *parser) parseBody() ([]Node, error) {
	var nodes []Node

	for !p.atEnd() {
		tok := p.peek()

		if tok.Type == tokEOF {
			break
		}

		if tok.Type == tokData {
			p.advance()
			nodes = append(nodes, &OutputNode{Text: tok.Value, Pos: tokenPos(tok)})
			continue
		}

		if tok.Type == tokVariableBegin {
			n, err := p.parseExprTag()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, n)
			continue
		}

		if tok.Type == tokBlockBegin {
			branch := p.peekBlockName()
			if isEndTag(branch) {
				break
			}
			n, err := p.parseBlockTag()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, n)
			continue
		}

		// Skip unexpected tokens
		p.advance()
	}

	return nodes, nil
}

// parseExprTag parses {{ expr }}.
func (p *parser) parseExprTag() (Node, error) {
	p.expect(tokVariableBegin)
	expr, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "expression: %s", err)
	}
	p.expect(tokVariableEnd)
	return &ExprNode{Expr: expr, Pos: tokenPos(p.peek())}, nil
}

// parseBlockTag parses {% ... %}.
func (p *parser) parseBlockTag() (Node, error) {
	_ = p.expect(tokBlockBegin)
	nameTok := p.expect(tokName)
	name := nameTok.Value

	switch name {
	case "if":
		return p.parseIf()
	case "for":
		return p.parseFor()
	case "block":
		return p.parseBlock()
	case "extends":
		return p.parseExtends()
	case "include":
		return p.parseInclude()
	case "macro":
		return p.parseMacro()
	case "set":
		return p.parseSet()
	case "raw":
		// raw blocks are handled by the lexer, skip to endraw
		return p.skipToEnd("raw")
	case "call":
		return p.parseCallBlock()
	case "with":
		return p.parseWith()
	case "do":
		return p.parseDo()
	case "import":
		return p.parseImport()
	case "from":
		return p.parseFromImport()
	default:
		return nil, p.errorf(nameTok, "unknown block tag %q", name)
	}
}

// parseIf parses {% if ... %}...{% elif ... %}...{% else %}...{% endif %}.
func (p *parser) parseIf() (Node, error) {
	startTok := p.tokens[p.pos-2]
	cond, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "if condition: %s", err)
	}
	p.expect(tokVariableEnd) // %}

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}

	node := &IfNode{
		Cond: cond,
		body: body,
		Pos:  tokenPos(startTok),
	}

	// Parse elif/else/endif
	for !p.atEnd() {
		tok := p.peek()
		if tok.Type != tokBlockBegin {
			break
		}

		// Look ahead to see if this is elif/else/endif
		branch := p.peekBlockName()

		switch branch {
		case "elif":
			p.advance() // skip {%
			p.advance() // skip "elif"
			elifCond, err := p.parseExpression()
			if err != nil {
				return nil, p.errorf(p.peek(), "elif condition: %s", err)
			}
			p.expect(tokVariableEnd)
			elifBody, err := p.parseBody()
			if err != nil {
				return nil, err
			}
			node.elifBranches = append(node.elifBranches, ElifBranch{
				Cond: elifCond,
				body: elifBody,
			})

		case "else":
			p.advance() // skip {%
			p.advance() // skip "else"
			p.expect(tokVariableEnd)
			elseBody, err := p.parseBody()
			if err != nil {
				return nil, err
			}
			node.elseBody = elseBody

		case "endif":
			p.advance() // skip {%
			p.advance() // skip "endif"
			p.expect(tokVariableEnd)
			return node, nil

		default:
			break
		}
	}

	return nil, p.errorf(p.peek(), "unclosed if block")
}

// parseFor parses {% for x in expr %}...{% endfor %}.
func (p *parser) parseFor() (Node, error) {
	startTok := p.tokens[p.pos-2]

	// Parse variable names
	varTok := p.expect(tokName)
	varName := varTok.Value
	keyName := ""

	// Check for tuple unpacking: {% for key, value in ... %}
	if p.peek().Type == tokComma {
		p.advance() // skip ,
		keyTok := p.expect(tokName)
		keyName = varName
		varName = keyTok.Value
	}

	// Expect "in"
	if !p.isOperator("in") {
		return nil, p.errorf(p.peek(), "expected 'in' in for loop, got %v", p.peek())
	}
	p.advance() // skip "in"

	// Check for "recursive"
	recursive := false
	if p.peek().Type == tokName && p.peek().Value == "recursive" {
		p.advance()
		recursive = true
	}

	iter, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "for iterable: %s", err)
	}

	// Check for "if" filter in for loop: {% for x in expr if condition %}
	var forFilter Expr
	if p.isOperator("if") {
		p.advance()
		forFilter, err = p.parseExpression()
		if err != nil {
			return nil, p.errorf(p.peek(), "for if filter: %s", err)
		}
	}

	p.expect(tokVariableEnd)

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}

	node := &ForNode{
		Var:       varName,
		Key:       keyName,
		Iter:      iter,
		body:      body,
		Recursive: recursive,
		Pos:       tokenPos(startTok),
	}

	// Parse else/ endfor
	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "else" {
			p.advance() // skip {%
			p.advance() // skip "else"
			p.expect(tokVariableEnd)
			elseBody, err := p.parseBody()
			if err != nil {
				return nil, err
			}
			node.elseBody = elseBody
		}
	}

	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "endfor" {
			p.advance() // skip {%
			p.advance() // skip "endfor"
			p.expect(tokVariableEnd)
			return node, nil
		}
	}

	// Apply for filter if specified
	if forFilter != nil {
		node.Iter = &FilterExpr{
			Node: node.Iter,
			Name: "select",
			Args: []Expr{
				&CallExpr{
					Func: &NameExpr{Name: "lambda"},
					Args: []Expr{
						&BinOpExpr{Left: &NameExpr{Name: "_"}, Op: "==", Right: forFilter},
					},
				},
			},
		}
	}

	return nil, p.errorf(p.peek(), "unclosed for block")
}

// parseBlock parses {% block name %}...{% endblock %}.
func (p *parser) parseBlock() (Node, error) {
	startTok := p.tokens[p.pos-2]
	nameTok := p.expect(tokName)
	blockName := nameTok.Value

	// Skip optional block name after (for endblock matching)
	p.expect(tokVariableEnd)

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}

	// Parse endblock [name]
	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "endblock" {
			p.advance() // skip {%
			p.advance() // skip "endblock"
			// Optional name after endblock
			if p.peek().Type == tokName {
				p.advance()
			}
			p.expect(tokVariableEnd)
		}
	}

	return &BlockNode{
		name: blockName,
		body: body,
		Pos:  tokenPos(startTok),
	}, nil
}

// parseExtends parses {% extends "template" %}.
func (p *parser) parseExtends() (Node, error) {
	startTok := p.tokens[p.pos-2]
	nameExpr, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "extends: %s", err)
	}
	p.expect(tokVariableEnd)

	var templateName string
	switch e := nameExpr.(type) {
	case *LiteralExpr:
		templateName, _ = e.Value.(string)
	default:
		return nil, p.errorf(p.peek(), "extends requires a string literal")
	}

	return &ExtendsNode{
		templateName: templateName,
		Pos:          tokenPos(startTok),
	}, nil
}

// parseInclude parses {% include "template" [ignore missing] [with context|without context] %}.
func (p *parser) parseInclude() (Node, error) {
	startTok := p.tokens[p.pos-2]
	nameExpr, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "include: %s", err)
	}

	var templateName string
	switch e := nameExpr.(type) {
	case *LiteralExpr:
		templateName, _ = e.Value.(string)
	default:
		return nil, p.errorf(p.peek(), "include requires a string literal")
	}

	node := &IncludeNode{
		templateName: templateName,
		Pos:          tokenPos(startTok),
	}

	// Parse optional flags
	for p.peek().Type == tokName {
		switch p.peek().Value {
		case "ignore", "missing":
			p.advance()
			if p.peek().Type == tokName && p.peek().Value == "missing" {
				p.advance()
			}
			node.IgnoreMissing = true
		case "with":
			p.advance()
			if p.peek().Type == tokName && p.peek().Value == "context" {
				p.advance()
				node.WithContext = true
			}
		case "without":
			p.advance()
			if p.peek().Type == tokName && p.peek().Value == "context" {
				p.advance()
				node.WithContext = false
			}
		default:
			break
		}
	}

	p.expect(tokVariableEnd)
	return node, nil
}

// parseMacro parses {% macro name(args) %}...{% endmacro %}.
func (p *parser) parseMacro() (Node, error) {
	startTok := p.tokens[p.pos-2]
	nameTok := p.expect(tokName)
	macroName := nameTok.Value

	p.expect(tokLParen)

	var args []MacroArg
	var defaults []Expr
	catchVarargs := false
	catchKwargs := false

	for p.peek().Type != tokRParen {
		if p.peek().Type == tokOperator && p.peek().Value == "*" {
			p.advance()
			p.expect(tokName)
			catchVarargs = true
			continue
		}
		if p.peek().Type == tokOperator && p.peek().Value == "**" {
			p.advance()
			p.expect(tokName)
			catchKwargs = true
			continue
		}

		argTok := p.expect(tokName)
		arg := MacroArg{Name: argTok.Value}

		if p.peek().Type == tokAssign {
			p.advance() // skip =
			def, err := p.parseExpression()
			if err != nil {
				return nil, p.errorf(p.peek(), "macro default: %s", err)
			}
			arg.Default = def
			defaults = append(defaults, def)
		}

		args = append(args, arg)

		if p.peek().Type == tokComma {
			p.advance()
		}
	}

	p.expect(tokRParen)

	// Check for "caller"
	if p.peek().Type == tokName && p.peek().Value == "caller" {
		// This is handled differently in Jinja2
	}

	p.expect(tokVariableEnd)

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}

	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "endmacro" {
			p.advance() // skip {%
			p.advance() // skip "endmacro"
			p.expect(tokVariableEnd)
		}
	}

	return &MacroNode{
		Name:         macroName,
		Args:         args,
		Defaults:     defaults,
		CatchVarargs: catchVarargs,
		CatchKwargs:  catchKwargs,
		body:         body,
		Pos:          tokenPos(startTok),
	}, nil
}

// parseSet parses {% set name = expr %} or {% set name %}...{% endset %}.
func (p *parser) parseSet() (Node, error) {
	startTok := p.tokens[p.pos-2]
	nameTok := p.expect(tokName)

	if p.peek().Type == tokAssign {
		p.advance() // skip =
		value, err := p.parseExpression()
		if err != nil {
			return nil, p.errorf(p.peek(), "set value: %s", err)
		}
		p.expect(tokVariableEnd)
		return &SetNode{
			Name:  nameTok.Value,
			Value: value,
			Pos:   tokenPos(startTok),
		}, nil
	}

	// Block set: {% set name %}...{% endset %}
	p.expect(tokVariableEnd)
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "endset" {
			p.advance() // skip {%
			p.advance() // skip "endset"
			p.expect(tokVariableEnd)
		}
	}
	return &SetBlockNode{
		Name: nameTok.Value,
		body: body,
		Pos:  tokenPos(startTok),
	}, nil
}

// parseCallBlock parses {% call func(args) %}...{% endcall %}.
func (p *parser) parseCallBlock() (Node, error) {
	startTok := p.tokens[p.pos-2]
	callExpr, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "call: %s", err)
	}
	p.expect(tokVariableEnd)

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "endcall" {
			p.advance()
			p.advance()
			p.expect(tokVariableEnd)
		}
	}
	return &CallBlockNode{
		CallExpr: callExpr,
		body:     body,
		Pos:      tokenPos(startTok),
	}, nil
}

// parseWith parses {% with x=y %}...{% endwith %}.
func (p *parser) parseWith() (Node, error) {
	startTok := p.tokens[p.pos-2]

	var assignments []SetNode
	for {
		nameTok := p.expect(tokName)
		p.expect(tokAssign)
		value, err := p.parseExpression()
		if err != nil {
			return nil, p.errorf(p.peek(), "with value: %s", err)
		}
		assignments = append(assignments, SetNode{Name: nameTok.Value, Value: value})

		if p.peek().Type == tokComma {
			p.advance()
			continue
		}
		break
	}
	p.expect(tokVariableEnd)

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() && p.peek().Type == tokBlockBegin {
		branch := p.peekBlockName()
		if branch == "endwith" {
			p.advance()
			p.advance()
			p.expect(tokVariableEnd)
		}
	}
	return &WithNode{
		assignments: assignments,
		body:        body,
		Pos:         tokenPos(startTok),
	}, nil
}

// parseDo parses {% do expr %}.
func (p *parser) parseDo() (Node, error) {
	startTok := p.tokens[p.pos-2]
	expr, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "do: %s", err)
	}
	p.expect(tokVariableEnd)
	return &DoNode{
		Expr: expr,
		Pos:  tokenPos(startTok),
	}, nil
}

// parseImport parses {% import "module" as name %}.
func (p *parser) parseImport() (Node, error) {
	startTok := p.tokens[p.pos-2]
	// For now, just parse and store
	_ = startTok
	expr, err := p.parseExpression()
	_ = expr
	if err != nil {
		return nil, p.errorf(p.peek(), "import: %s", err)
	}

	// Expect "as"
	if p.peek().Type == tokName && p.peek().Value == "as" {
		p.advance()
		aliasTok := p.expect(tokName)
		p.expect(tokVariableEnd)
		return &ImportNode{
			templateName: stringify(expr),
			Alias:        aliasTok.Value,
			Pos:          tokenPos(startTok),
		}, nil
	}

	p.expect(tokVariableEnd)
	return nil, p.errorf(p.peek(), "import requires 'as alias'")
}

// parseFromImport parses {% from "module" import name1, name2 as alias %}.
func (p *parser) parseFromImport() (Node, error) {
	startTok := p.tokens[p.pos-2]
	// Parse module name
	moduleExpr, err := p.parseExpression()
	if err != nil {
		return nil, p.errorf(p.peek(), "from import: %s", err)
	}

	// Expect "import"
	if !(p.peek().Type == tokName && p.peek().Value == "import") {
		return nil, p.errorf(p.peek(), "expected 'import'")
	}
	p.advance()

	var names []string
	var aliases []string

	for {
		nameTok := p.expect(tokName)
		names = append(names, nameTok.Value)
		alias := nameTok.Value

		if p.peek().Type == tokName && p.peek().Value == "as" {
			p.advance()
			aliasTok := p.expect(tokName)
			alias = aliasTok.Value
		}
		aliases = append(aliases, alias)

		if p.peek().Type == tokComma {
			p.advance()
			continue
		}
		break
	}

	p.expect(tokVariableEnd)

	return &FromImportNode{
		templateName: stringify(moduleExpr),
		Names:        names,
		Aliases:      aliases,
		Pos:          tokenPos(startTok),
	}, nil
}

// skipToEnd skips to the matching end tag.
func (p *parser) skipToEnd(tag string) (Node, error) {
	endTag := "end" + tag
	for !p.atEnd() {
		if p.peek().Type == tokBlockBegin {
			branch := p.peekBlockName()
			if branch == endTag {
				p.advance() // {%
				p.advance() // endtag
				p.expect(tokVariableEnd)
				return &CommentNode{Text: "skipped", Pos: tokenPos(p.peek())}, nil
			}
		}
		p.advance()
	}
	return nil, fmt.Errorf("unclosed %s block", tag)
}

// ==================== Expression Parsing ====================

// parseExpression parses a full expression.
func (p *parser) parseExpression() (Expr, error) {
	return p.parseTernary()
}

// parseTernary parses: expr1 if cond else expr2
func (p *parser) parseTernary() (Expr, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.peek().Type == tokName && p.peek().Value == "if" {
		p.advance() // skip "if"
		cond, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().Type == tokName && p.peek().Value == "else" {
			p.advance() // skip "else"
			falseExpr, err := p.parseTernary()
			if err != nil {
				return nil, err
			}
			return &CondExpr{True: expr, Cond: cond, False: falseExpr}, nil
		}
		// No else: falseExpr defaults to empty string
		return &CondExpr{True: expr, Cond: cond, False: &LiteralExpr{Value: ""}}, nil
	}

	return expr, nil
}

// parseOr parses: expr or expr
func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.isOperator("or") {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinOpExpr{Left: left, Op: "or", Right: right}
	}

	return left, nil
}

// parseAnd parses: expr and expr
func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.isOperator("and") {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinOpExpr{Left: left, Op: "and", Right: right}
	}

	return left, nil
}

// parseNot parses: not expr
func (p *parser) parseNot() (Expr, error) {
	if p.isOperator("not") {
		p.advance()
		expr, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "not", Node: expr}, nil
	}
	return p.parseCompare()
}

// parseCompare parses: expr op expr (chained comparisons, in/not in, is/is not)
func (p *parser) parseCompare() (Expr, error) {
	left, err := p.parseAdd()
	if err != nil {
		return nil, err
	}

	var ops []CompareOp
	for {
		op := p.peek()

		// Check for comparison operators
		if op.Type == tokOperator && isCompareOp(op.Value) {
			p.advance()
			right, err := p.parseAdd()
			if err != nil {
				return nil, err
			}
			ops = append(ops, CompareOp{Op: op.Value, Right: right})
			continue
		}

		// Check for "in" / "not in"
		if p.isOperator("in") {
			p.advance()
			right, err := p.parseAdd()
			if err != nil {
				return nil, err
			}
			ops = append(ops, CompareOp{Op: "in", Right: right})
			continue
		}
		if p.isOperator("not") && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == tokName && p.tokens[p.pos+1].Value == "in" {
			p.advance() // not
			p.advance() // in
			right, err := p.parseAdd()
			if err != nil {
				return nil, err
			}
			ops = append(ops, CompareOp{Op: "not in", Right: right})
			continue
		}

		// Check for "is" / "is not"
		if p.peek().Type == tokOperator && p.peek().Value == "is" {
			p.advance()
			negated := false
			if p.isOperator("not") {
				p.advance()
				negated = true
			}
			testTok := p.peek()
			if testTok.Type != tokName {
				p.advance()
			} else {
				testTok = p.advance()
			}
			testName := testTok.Value
			if testName == "" {
				// none, true, false
				switch testTok.Type {
				case tokNone:
					testName = "none"
				case tokBool:
					testName = testTok.Value
				}
			}
			var args []Expr
			if p.peek().Type == tokLParen {
				p.advance()
				for p.peek().Type != tokRParen {
					arg, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peek().Type == tokComma {
						p.advance()
					}
				}
				p.expect(tokRParen)
			}
			left = &TestExpr{Node: left, Name: testName, Args: args, Neg: negated}
			continue
		}

		break
	}

	if len(ops) == 0 {
		return left, nil
	}
	if len(ops) == 1 {
		return &BinOpExpr{Left: left, Op: ops[0].Op, Right: ops[0].Right}, nil
	}
	return &CompareExpr{Expr: left, Ops: ops}, nil
}

// parseAdd parses: expr + expr / expr - expr / expr ~ expr (concat)
func (p *parser) parseAdd() (Expr, error) {
	left, err := p.parseMul()
	if err != nil {
		return nil, err
	}

	for {
		op := p.peek()
		if op.Type == tokOperator && (op.Value == "+" || op.Value == "-" || op.Value == "~") {
			p.advance()
			right, err := p.parseMul()
			if err != nil {
				return nil, err
			}
			if op.Value == "~" {
				left = &ConcatExpr{Nodes: []Expr{left, right}}
			} else {
				left = &BinOpExpr{Left: left, Op: op.Value, Right: right}
			}
			continue
		}
		break
	}

	return left, nil
}

// parseMul parses: expr * expr / expr / expr / expr // expr / expr % expr / expr ** expr
func (p *parser) parseMul() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		op := p.peek()
		if op.Type == tokOperator && (op.Value == "*" || op.Value == "/" || op.Value == "//" || op.Value == "%" || op.Value == "**") {
			p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &BinOpExpr{Left: left, Op: op.Value, Right: right}
			continue
		}
		break
	}

	return left, nil
}

// parseUnary parses: -expr / +expr
func (p *parser) parseUnary() (Expr, error) {
	op := p.peek()
	if op.Type == tokOperator && (op.Value == "-" || op.Value == "+") {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op.Value, Node: expr}, nil
	}
	return p.parsePostfix()
}

// parsePostfix parses: expr.filter | expr.attr | expr[0] | expr(args) | expr.key
func (p *parser) parsePostfix() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		switch p.peek().Type {
		case tokPipe:
			// Filter: expr | filter(args)
			p.advance()
			filterName := p.expect(tokName)
			var args []Expr
			for p.peek().Type == tokColon {
				p.advance()
				arg, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
			// Also handle filter(arg) syntax (used without colon)
			if p.peek().Type == tokLParen {
				p.advance()
				for p.peek().Type != tokRParen {
					arg, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peek().Type == tokComma {
						p.advance()
					}
				}
				p.expect(tokRParen)
			}
			expr = &FilterExpr{Node: expr, Name: filterName.Value, Args: args}

		case tokDot:
			// Attribute access: expr.attr
			p.advance()
			attrTok := p.peek()
			if attrTok.Type == tokName {
				p.advance()
				expr = &GetAttrExpr{Node: expr, Attr: attrTok.Value}
			} else if attrTok.Type == tokInteger || attrTok.Type == tokName {
				p.advance()
				expr = &GetAttrExpr{Node: expr, Attr: attrTok.Value}
			} else {
				// Special case: .0, .1 etc (numeric keys)
				p.advance()
				expr = &GetAttrExpr{Node: expr, Attr: attrTok.Value}
			}

		case tokLBrack:
			// Subscript: expr[key]
			p.advance()
			idx, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			// Check for slice: [start:stop:step]
			if p.peek().Type == tokColon {
				p.advance()
				from := idx
				var to Expr
				var step Expr
				if p.peek().Type != tokRBrack && p.peek().Type != tokColon {
					to, err = p.parseExpression()
					if err != nil {
						return nil, err
					}
				}
				if p.peek().Type == tokColon {
					p.advance()
					if p.peek().Type != tokRBrack {
						step, err = p.parseExpression()
						if err != nil {
							return nil, err
						}
					}
				}
				p.expect(tokRBrack)
				expr = &SliceExpr{Node: expr, From: from, To: to, Step: step}
			} else {
				p.expect(tokRBrack)
				expr = &GetItemExpr{Node: expr, Idx: idx}
			}

		case tokLParen:
			// Function call: expr(args)
			p.advance()
			var args []Expr
			var kwargs []CallKwarg
			for p.peek().Type != tokRParen {
				// Check for keyword argument
				if p.peek().Type == tokName && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == tokAssign {
					key := p.expect(tokName)
					p.expect(tokAssign)
					value, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					kwargs = append(kwargs, CallKwarg{Key: key.Value, Value: value})
				} else {
					arg, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
				}
				if p.peek().Type == tokComma {
					p.advance()
				}
			}
			p.expect(tokRParen)
			expr = &CallExpr{Func: expr, Args: args, kwargs: kwargs}

		default:
			return expr, nil
		}
	}
}

// parsePrimary parses atoms: names, literals, lists, dicts, tuples, parenthesized expressions.
func (p *parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	switch tok.Type {
	case tokName:
		p.advance()
		return &NameExpr{Name: tok.Value}, nil

	case tokString:
		p.advance()
		return &LiteralExpr{Value: tok.Value}, nil

	case tokInteger:
		p.advance()
		n, err := strconv.ParseInt(tok.Value, 10, 64)
		if err != nil {
			return nil, p.errorf(tok, "invalid integer %q", tok.Value)
		}
		return &LiteralExpr{Value: n}, nil

	case tokFloat:
		p.advance()
		f, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, p.errorf(tok, "invalid float %q", tok.Value)
		}
		return &LiteralExpr{Value: f}, nil

	case tokBool:
		p.advance()
		return &LiteralExpr{Value: tok.Value == "true"}, nil

	case tokNone:
		p.advance()
		return &LiteralExpr{Value: nil}, nil

	case tokLBrack:
		return p.parseList()

	case tokLBrace:
		return p.parseDict()

	case tokLParen:
		return p.parseTupleOrGroup()

	default:
		return nil, p.errorf(tok, "unexpected token %v", tok)
	}
}

// parseList parses: [item, item, ...]
func (p *parser) parseList() (Expr, error) {
	p.expect(tokLBrack)
	var items []Expr
	for p.peek().Type != tokRBrack {
		item, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		if p.peek().Type == tokComma {
			p.advance()
		}
	}
	p.expect(tokRBrack)
	return &ListExpr{Items: items}, nil
}

// parseDict parses: {key: value, ...}
func (p *parser) parseDict() (Expr, error) {
	p.expect(tokLBrace)
	var pairs []DictPair
	for p.peek().Type != tokRBrace {
		key, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.expect(tokColon)
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, DictPair{Key: key, Value: value})
		if p.peek().Type == tokComma {
			p.advance()
		}
	}
	p.expect(tokRBrace)
	return &DictExpr{Pairs: pairs}, nil
}

// parseTupleOrGroup parses: (expr, expr, ...) or (expr).
func (p *parser) parseTupleOrGroup() (Expr, error) {
	p.expect(tokLParen)

	// Empty tuple
	if p.peek().Type == tokRParen {
		p.advance()
		return &TupleExpr{Items: nil}, nil
	}

	first, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.peek().Type == tokComma {
		// Tuple
		items := []Expr{first}
		for p.peek().Type == tokComma {
			p.advance()
			if p.peek().Type == tokRParen {
				break
			}
			item, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		p.expect(tokRParen)
		return &TupleExpr{Items: items}, nil
	}

	p.expect(tokRParen)
	return first, nil
}

// ==================== Helper Methods ====================

func (p *parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parser) expect(t tokenType) Token {
	tok := p.peek()
	if tok.Type != t {
		panic(p.errorf(tok, "expected %d, got %v", t, tok).Error())
	}
	return p.advance()
}

func (p *parser) atEnd() bool {
	return p.pos >= len(p.tokens) || p.peek().Type == tokEOF
}

func (p *parser) isOperator(op string) bool {
	tok := p.peek()
	return tok.Type == tokOperator && tok.Value == op
}

// peekBlockName looks ahead to find the name after a block begin tag.
// Does not consume tokens.
func (p *parser) peekBlockName() string {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != tokBlockBegin {
		return ""
	}
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == tokName {
		return p.tokens[p.pos+1].Value
	}
	return ""
}

func (p *parser) errorf(tok Token, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("line %d, col %d: %s", tok.Line, tok.Col, msg)
}

func tokenPos(tok Token) Position {
	return Position{Line: tok.Line, Column: tok.Col}
}

func isCompareOp(op string) bool {
	switch op {
	case "==", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

var endTags = map[string]bool{
	"endif": true, "endfor": true, "endblock": true,
	"endmacro": true, "endwith": true, "endcall": true,
	"endset": true, "endraw": true,
	"else": true, "elif": true,
}

func isEndTag(name string) bool {
	return endTags[name]
}
