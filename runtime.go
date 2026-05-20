package jinja

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type mapEntry struct {
	key, value interface{}
}

type loopVar struct {
	index     int
	index1    int
	first     bool
	last      bool
	length    int
	revindex  int
	revindex1 int
	items     []interface{}
	depth     int
}

func newLoopVar(i, length int, items []interface{}) *loopVar {
	return &loopVar{
		index:     i + 1,
		index1:    i + 1,
		first:     i == 0,
		last:      i == length-1,
		length:    length,
		revindex:  length - i,
		revindex1: length - i - 1,
		items:     items,
		depth:     1,
	}
}

func (l *loopVar) Get(key string) (interface{}, bool) {
	switch key {
	case "index":
		return l.index, true
	case "index0":
		return l.index - 1, true
	case "index1":
		return l.index1, true
	case "first":
		return l.first, true
	case "last":
		return l.last, true
	case "length":
		return l.length, true
	case "revindex":
		return l.revindex, true
	case "revindex0":
		return l.revindex, true
	case "revindex1":
		return l.revindex1, true
	case "depth":
		return l.depth, true
	case "depth0":
		return l.depth - 1, true
	case "depth1":
		return l.depth, true
	case "cycle":
		return l.cycleFunc(), true
	case "changed":
		return l.changedFunc, true
	}
	return nil, false
}

func (l *loopVar) cycleFunc() func(args ...interface{}) interface{} {
	return func(args ...interface{}) interface{} {
		if len(args) == 0 {
			return nil
		}
		idx := (l.index - 1) % len(args)
		return args[idx]
	}
}

func (l *loopVar) changedFunc(val ...interface{}) bool {
	if l.index == 0 {
		return true
	}
	if len(val) == 0 {
		return false
	}
	return fmt.Sprintf("%v", val[0]) != fmt.Sprintf("%v", l.items[l.index-1])
}

// ExecutionContext holds the state during template execution.
type ExecutionContext struct {
	env    *Environment
	vars   map[string]interface{}
	blocks map[string]*BlockNode
	parent *ExecutionContext
	caller *CallerInfo
}

// CallerInfo holds information about a call block.
type CallerInfo struct {
	body []Node
	ctx  *ExecutionContext
}

// NewExecutionContext creates a new execution context.
func NewExecutionContext(env *Environment, data interface{}) *ExecutionContext {
	ctx := &ExecutionContext{
		env:    env,
		vars:   make(map[string]interface{}),
		blocks: make(map[string]*BlockNode),
	}
	if data != nil {
		flattenData(data, ctx.vars)
	}
	if env != nil {
		env.mu.RLock()
		for k, v := range env.globalFuncs {
			ctx.vars[k] = v
		}
		env.mu.RUnlock()
	}
	return ctx
}

func (ctx *ExecutionContext) child() *ExecutionContext {
	child := &ExecutionContext{
		env:    ctx.env,
		vars:   make(map[string]interface{}),
		blocks: ctx.blocks,
		parent: ctx,
	}
	return child
}

func (ctx *ExecutionContext) get(name string) interface{} {
	if v, ok := ctx.vars[name]; ok {
		return v
	}
	if ctx.parent != nil {
		return ctx.parent.get(name)
	}
	if ctx.env != nil {
		if v, ok := ctx.env.GetGlobal(name); ok {
			return v
		}
	}
	return &UndefinedValue{Name: name, Msg: fmt.Sprintf("'%s' is undefined", name)}
}

func (ctx *ExecutionContext) set(name string, value interface{}) {
	ctx.vars[name] = value
}

func flattenData(data interface{}, vars map[string]interface{}) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key().Interface())
			vars[key] = iter.Value().Interface()
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := field.Name
			tag := field.Tag.Get("jinja")
			if tag != "" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" {
					name = parts[0]
				}
			} else if tag = field.Tag.Get("json"); tag != "" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" {
					name = parts[0]
				}
			}
			vars[name] = v.Field(i).Interface()
		}
	}
}

// executeNodes executes a list of nodes.
func executeNodes(ctx *ExecutionContext, buf *stringsWrapper, nodes []Node) error {
	for _, node := range nodes {
		if err := executeNode(ctx, buf, node); err != nil {
			return err
		}
	}
	return nil
}

func executeNode(ctx *ExecutionContext, buf *stringsWrapper, node Node) error {
	switch n := node.(type) {
	case *OutputNode:
		buf.WriteString(n.Text)
	case *ExprNode:
		val, err := evalExpr(ctx, n.Expr)
		if err != nil {
			return err
		}
		buf.WriteString(outputString(ctx.env, val))
	case *IfNode:
		return executeIf(ctx, buf, n)
	case *ForNode:
		return executeFor(ctx, buf, n)
	case *BlockNode:
		return executeBlock(ctx, buf, n)
	case *ExtendsNode:
		return nil
	case *IncludeNode:
		return executeInclude(ctx, buf, n)
	case *MacroNode:
		ctx.set(n.Name, createMacro(ctx, n))
		return nil
	case *CallBlockNode:
		return executeCallBlock(ctx, buf, n)
	case *SetNode:
		val, err := evalExpr(ctx, n.Value)
		if err != nil {
			return err
		}
		ctx.set(n.Name, val)
		return nil
	case *SetBlockNode:
		var inner stringsWrapper
		if err := executeNodes(ctx, &inner, n.body); err != nil {
			return err
		}
		ctx.set(n.Name, inner.String())
		return nil
	case *WithNode:
		return executeWith(ctx, buf, n)
	case *DoNode:
		_, err := evalExpr(ctx, n.Expr)
		return err
	case *ImportNode:
		return executeImport(ctx, n)
	case *FromImportNode:
		return executeFromImport(ctx, n)
	case *RawNode:
		buf.WriteString(n.Text)
	case *CommentNode:
		return nil
	default:
		return fmt.Errorf("unknown node type: %T", node)
	}
	return nil
}

func executeIf(ctx *ExecutionContext, buf *stringsWrapper, n *IfNode) error {
	cond, err := evalExpr(ctx, n.Cond)
	if err != nil {
		return err
	}
	if isTruthy(cond) {
		return executeNodes(ctx, buf, n.body)
	}
	for _, elif := range n.elifBranches {
		cond, err := evalExpr(ctx, elif.Cond)
		if err != nil {
			return err
		}
		if isTruthy(cond) {
			return executeNodes(ctx, buf, elif.body)
		}
	}
	if n.elseBody != nil {
		return executeNodes(ctx, buf, n.elseBody)
	}
	return nil
}

func executeFor(ctx *ExecutionContext, buf *stringsWrapper, n *ForNode) error {
	iterVal, err := evalExpr(ctx, n.Iter)
	if err != nil {
		return err
	}
	items := toSlice(iterVal)
	if len(items) == 0 {
		if n.elseBody != nil {
			return executeNodes(ctx, buf, n.elseBody)
		}
		return nil
	}
	for i, item := range items {
		loopCtx := ctx.child()
		if n.Key != "" {
			switch v := item.(type) {
			case [2]interface{}:
				loopCtx.set(n.Key, v[0])
				loopCtx.set(n.Var, v[1])
			case mapEntry:
				loopCtx.set(n.Key, v.key)
				loopCtx.set(n.Var, v.value)
			default:
				rv := reflect.ValueOf(item)
				if rv.Kind() == reflect.Map {
					iter := rv.MapRange()
					if iter.Next() {
						loopCtx.set(n.Key, iter.Key().Interface())
						loopCtx.set(n.Var, iter.Value().Interface())
					}
				}
			}
		} else {
			loopCtx.set(n.Var, item)
		}
		loop := newLoopVar(i, len(items), items)
		loopCtx.set("loop", loop)
		if err := executeNodes(loopCtx, buf, n.body); err != nil {
			return err
		}
	}
	return nil
}

func executeBlock(ctx *ExecutionContext, buf *stringsWrapper, n *BlockNode) error {
	if ctx.blocks != nil {
		ctx.blocks[n.name] = n
	}
	return executeNodes(ctx, buf, n.body)
}

func executeInclude(ctx *ExecutionContext, buf *stringsWrapper, n *IncludeNode) error {
	if ctx.env == nil || ctx.env.loader == nil {
		if n.IgnoreMissing {
			return nil
		}
		return fmt.Errorf("no template loader configured")
	}
	src, err := ctx.env.loader.Load(n.templateName)
	if err != nil {
		if n.IgnoreMissing {
			return nil
		}
		return fmt.Errorf("cannot include template %q: %w", n.templateName, err)
	}
	tmpl, err := ctx.env.TemplateFromString(n.templateName, src)
	if err != nil {
		return err
	}
	includeCtx := ctx
	if !n.WithContext {
		includeCtx = NewExecutionContext(ctx.env, nil)
	}
	return tmpl.treeExecute(includeCtx, buf)
}

func executeWith(ctx *ExecutionContext, buf *stringsWrapper, n *WithNode) error {
	child := ctx.child()
	for _, a := range n.assignments {
		val, err := evalExpr(ctx, a.Value)
		if err != nil {
			return err
		}
		child.set(a.Name, val)
	}
	return executeNodes(child, buf, n.body)
}

func executeImport(ctx *ExecutionContext, n *ImportNode) error {
	if ctx.env == nil || ctx.env.loader == nil {
		return fmt.Errorf("no template loader configured")
	}
	src, err := ctx.env.loader.Load(n.templateName)
	if err != nil {
		return fmt.Errorf("cannot import %q: %w", n.templateName, err)
	}
	tmpl, err := ctx.env.TemplateFromString(n.templateName, src)
	if err != nil {
		return err
	}
	macroCtx := NewExecutionContext(ctx.env, nil)
	var _ stringsWrapper
	_ = tmpl.treeExecute(macroCtx, new(stringsWrapper))
	ctx.set(n.Alias, macroCtx)
	return nil
}

func executeFromImport(ctx *ExecutionContext, n *FromImportNode) error {
	if ctx.env == nil || ctx.env.loader == nil {
		return fmt.Errorf("no template loader configured")
	}
	src, err := ctx.env.loader.Load(n.templateName)
	if err != nil {
		return fmt.Errorf("cannot import %q: %w", n.templateName, err)
	}
	tmpl, err := ctx.env.TemplateFromString(n.templateName, src)
	if err != nil {
		return err
	}
	macroCtx := NewExecutionContext(ctx.env, nil)
	_ = tmpl.treeExecute(macroCtx, new(stringsWrapper))
	for i, name := range n.Names {
		alias := name
		if i < len(n.Aliases) {
			alias = n.Aliases[i]
		}
		if v, ok := macroCtx.vars[name]; ok {
			ctx.set(alias, v)
		}
	}
	return nil
}

func executeCallBlock(ctx *ExecutionContext, buf *stringsWrapper, n *CallBlockNode) error {
	caller := &CallerInfo{body: n.body, ctx: ctx}
	oldCaller := ctx.caller
	ctx.caller = caller
	defer func() { ctx.caller = oldCaller }()
	val, err := evalExpr(ctx, n.CallExpr)
	if err != nil {
		return err
	}
	buf.WriteString(outputString(ctx.env, val))
	return nil
}

// ==================== Expression Evaluation ====================

func evalExpr(ctx *ExecutionContext, expr Expr) (interface{}, error) {
	switch e := expr.(type) {
	case *LiteralExpr:
		return e.Value, nil
	case *NameExpr:
		return ctx.get(e.Name), nil
	case *GetAttrExpr:
		return evalGetAttr(ctx, e)
	case *GetItemExpr:
		return evalGetItem(ctx, e)
	case *CallExpr:
		return evalCall(ctx, e)
	case *FilterExpr:
		return evalFilter(ctx, e)
	case *TestExpr:
		return evalTest(ctx, e)
	case *BinOpExpr:
		return evalBinOp(ctx, e)
	case *UnaryExpr:
		return evalUnary(ctx, e)
	case *CondExpr:
		return evalCondExpr(ctx, e)
	case *ListExpr:
		return evalList(ctx, e)
	case *DictExpr:
		return evalDict(ctx, e)
	case *TupleExpr:
		return evalTuple(ctx, e)
	case *SliceExpr:
		return evalSlice(ctx, e)
	case *CompareExpr:
		return evalCompare(ctx, e)
	case *ConcatExpr:
		return evalConcat(ctx, e)
	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

func evalGetAttr(ctx *ExecutionContext, e *GetAttrExpr) (interface{}, error) {
	obj, err := evalExpr(ctx, e.Node)
	if err != nil {
		return nil, err
	}
	if lv, ok := obj.(*loopVar); ok {
		if v, ok := lv.Get(e.Attr); ok {
			return v, nil
		}
	}
	if m, ok := obj.(map[string]interface{}); ok {
		if v, exists := m[e.Attr]; exists {
			return v, nil
		}
	}
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		fv := rv.FieldByName(e.Attr)
		if fv.IsValid() {
			return fv.Interface(), nil
		}
		t := rv.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Anonymous {
				continue
			}
			tag := field.Tag.Get("jinja")
			if tag == "" {
				tag = field.Tag.Get("json")
			}
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] == e.Attr {
				return rv.Field(i).Interface(), nil
			}
		}
	}
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Struct {
		mv := reflect.ValueOf(obj)
		if mv.Kind() != reflect.Ptr {
			mv = reflect.New(rv.Type())
			mv.Elem().Set(rv)
		}
		method := mv.MethodByName(e.Attr)
		if method.IsValid() {
			return &GoFunc{Name: e.Attr, Method: method}, nil
		}
	}
	rv = reflect.ValueOf(obj)
	if rv.Kind() == reflect.Map {
		key := reflect.ValueOf(e.Attr)
		if key.Type().ConvertibleTo(rv.Type().Key()) {
			v := rv.MapIndex(key.Convert(rv.Type().Key()))
			if v.IsValid() {
				return v.Interface(), nil
			}
		}
	}
	return &UndefinedValue{Name: e.Attr, Msg: fmt.Sprintf("'%s' is undefined", e.Attr)}, nil
}

func evalGetItem(ctx *ExecutionContext, e *GetItemExpr) (interface{}, error) {
	obj, err := evalExpr(ctx, e.Node)
	if err != nil {
		return nil, err
	}
	idx, err := evalExpr(ctx, e.Idx)
	if err != nil {
		return nil, err
	}
	rv := reflect.ValueOf(obj)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		i := toInt(idx)
		if i < 0 {
			i = rv.Len() + i
		}
		if i < 0 || i >= rv.Len() {
			return &UndefinedValue{Name: fmt.Sprintf("index %d", i), Msg: fmt.Sprintf("index %d out of range", i)}, nil
		}
		return rv.Index(i).Interface(), nil
	case reflect.Map:
		key := reflect.ValueOf(idx)
		if key.Type().ConvertibleTo(rv.Type().Key()) {
			v := rv.MapIndex(key.Convert(rv.Type().Key()))
			if v.IsValid() {
				return v.Interface(), nil
			}
		}
		return nil, nil
	case reflect.String:
		i := toInt(idx)
		if i < 0 {
			i = len(obj.(string)) + i
		}
		s := obj.(string)
		if i >= 0 && i < len(s) {
			return string(s[i]), nil
		}
		return "", nil
	}
	return nil, fmt.Errorf("cannot index into %T", obj)
}

type GoFunc struct {
	Name   string
	Method reflect.Value
}

func evalCall(ctx *ExecutionContext, e *CallExpr) (interface{}, error) {
	fnVal, err := evalExpr(ctx, e.Func)
	if err != nil {
		return nil, err
	}
	args := make([]interface{}, len(e.Args))
	for i, a := range e.Args {
		args[i], err = evalExpr(ctx, a)
		if err != nil {
			return nil, err
		}
	}
	kwargs := make(map[string]interface{})
	for _, kw := range e.kwargs {
		val, err := evalExpr(ctx, kw.Value)
		if err != nil {
			return nil, err
		}
		kwargs[kw.Key] = val
	}

	switch fn := fnVal.(type) {
	case func(*ExecutionContext, ...interface{}) (interface{}, error):
		return fn(ctx, args...)
	case func(...interface{}) (interface{}, error):
		return fn(args...)
	case func(*ExecutionContext, ...interface{}) interface{}:
		return fn(ctx, args...), nil
	case func(...interface{}) interface{}:
		return fn(args...), nil
	case *Macro:
		return callMacro(ctx, fn, args, kwargs)
	case *GoFunc:
		return callGoFunc(fn, args)
	}
	// Try reflect-based call
	rv := reflect.ValueOf(fnVal)
	if rv.Kind() == reflect.Func {
		return callReflectFunc(rv, args)
	}
	return nil, fmt.Errorf("'%v' is not callable", fnVal)
}

func callMacro(ctx *ExecutionContext, m *Macro, args []interface{}, kwargs map[string]interface{}) (interface{}, error) {
	macroCtx := ctx.child()
	for i, arg := range m.Args {
		if i < len(args) {
			macroCtx.set(arg.Name, args[i])
		} else if kwarg, ok := kwargs[arg.Name]; ok {
			macroCtx.set(arg.Name, kwarg)
		} else if arg.Default != nil {
			val, err := evalExpr(ctx, arg.Default)
			if err != nil {
				return nil, err
			}
			macroCtx.set(arg.Name, val)
		}
	}
	if m.CatchVarargs && len(args) > len(m.Args) {
		macroCtx.set("varargs", args[len(m.Args):])
	}
	if m.CatchKwargs {
		macroCtx.set("kwargs", kwargs)
	}
	var buf stringsWrapper
	if err := executeNodes(macroCtx, &buf, m.Node.body); err != nil {
		return nil, err
	}
	return buf.String(), nil
}

func callGoFunc(fn *GoFunc, args []interface{}) (interface{}, error) {
	mv := fn.Method
	mt := mv.Type()
	in := make([]reflect.Value, mt.NumIn())
	for i := 0; i < mt.NumIn() && i < len(args); i++ {
		argVal := reflect.ValueOf(args[i])
		if argVal.Type().ConvertibleTo(mt.In(i)) {
			in[i] = argVal.Convert(mt.In(i))
		} else {
			in[i] = argVal
		}
	}
	results := mv.Call(in)
	if len(results) == 0 {
		return nil, nil
	}
	if len(results) == 1 {
		return results[0].Interface(), nil
	}
	if err, ok := results[len(results)-1].Interface().(error); ok {
		return results[0].Interface(), err
	}
	slice := make([]interface{}, len(results))
	for i, r := range results {
		slice[i] = r.Interface()
	}
	return slice, nil
}

func callReflectFunc(fn reflect.Value, args []interface{}) (interface{}, error) {
	ft := fn.Type()
	in := make([]reflect.Value, ft.NumIn())
	for i := 0; i < ft.NumIn() && i < len(args); i++ {
		in[i] = reflect.ValueOf(args[i])
	}
	results := fn.Call(in)
	if len(results) == 0 {
		return nil, nil
	}
	return results[0].Interface(), nil
}

func createMacro(ctx *ExecutionContext, n *MacroNode) *Macro {
	return &Macro{
		Name:         n.Name,
		Args:         n.Args,
		Node:         n,
		Env:          ctx.env,
		Ctx:          ctx,
		CatchVarargs: n.CatchVarargs,
		CatchKwargs:  n.CatchKwargs,
	}
}

type Macro struct {
	Name         string
	Args         []MacroArg
	Node         *MacroNode
	Env          *Environment
	Ctx          *ExecutionContext
	CatchVarargs bool
	CatchKwargs  bool
}

func evalFilter(ctx *ExecutionContext, e *FilterExpr) (interface{}, error) {
	val, err := evalExpr(ctx, e.Node)
	if err != nil {
		return nil, err
	}
	args := []interface{}{val}
	for _, a := range e.Args {
		arg, err := evalExpr(ctx, a)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	if ctx.env != nil {
		if fn, ok := ctx.env.GetFilter(e.Name); ok {
		return fn(ctx, args[0], args[1:]...)
		}
	}
	return nil, fmt.Errorf("no filter named %q", e.Name)
}

func evalTest(ctx *ExecutionContext, e *TestExpr) (interface{}, error) {
	val, err := evalExpr(ctx, e.Node)
	if err != nil {
		return nil, err
	}
	args := make([]interface{}, len(e.Args))
	for i, a := range e.Args {
		args[i], err = evalExpr(ctx, a)
		if err != nil {
			return nil, err
		}
	}
	if ctx.env != nil {
		if fn, ok := ctx.env.GetTest(e.Name); ok {
			result, err := fn(ctx, val, args...)
			if err != nil {
				return nil, err
			}
			if e.Neg {
				return !result, nil
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("no test named %q", e.Name)
}

func evalBinOp(ctx *ExecutionContext, e *BinOpExpr) (interface{}, error) {
	left, err := evalExpr(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := evalExpr(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case "and":
		if !isTruthy(left) {
			return left, nil
		}
		return right, nil
	case "or":
		if isTruthy(left) {
			return left, nil
		}
		return right, nil
	}
	return binOp(e.Op, left, right)
}

func evalUnary(ctx *ExecutionContext, e *UnaryExpr) (interface{}, error) {
	val, err := evalExpr(ctx, e.Node)
	if err != nil {
		return nil, err
	}
	switch e.Op {
	case "not":
		return !isTruthy(val), nil
	case "-":
		switch v := val.(type) {
		case int:
			return -v, nil
		case int64:
			return -v, nil
		case float64:
			return -v, nil
		}
		return nil, fmt.Errorf("cannot negate %T", val)
	case "+":
		return val, nil
	}
	return nil, fmt.Errorf("unknown unary operator %q", e.Op)
}

func evalCondExpr(ctx *ExecutionContext, e *CondExpr) (interface{}, error) {
	cond, err := evalExpr(ctx, e.Cond)
	if err != nil {
		return nil, err
	}
	if isTruthy(cond) {
		return evalExpr(ctx, e.True)
	}
	return evalExpr(ctx, e.False)
}

func evalList(ctx *ExecutionContext, e *ListExpr) (interface{}, error) {
	items := make([]interface{}, len(e.Items))
	for i, item := range e.Items {
		val, err := evalExpr(ctx, item)
		if err != nil {
			return nil, err
		}
		items[i] = val
	}
	return items, nil
}

func evalDict(ctx *ExecutionContext, e *DictExpr) (interface{}, error) {
	m := make(map[string]interface{})
	for _, pair := range e.Pairs {
		key, err := evalExpr(ctx, pair.Key)
		if err != nil {
			return nil, err
		}
		value, err := evalExpr(ctx, pair.Value)
		if err != nil {
			return nil, err
		}
		m[fmt.Sprintf("%v", key)] = value
	}
	return m, nil
}

func evalTuple(ctx *ExecutionContext, e *TupleExpr) (interface{}, error) {
	items := make([]interface{}, len(e.Items))
	for i, item := range e.Items {
		val, err := evalExpr(ctx, item)
		if err != nil {
			return nil, err
		}
		items[i] = val
	}
	return items, nil
}

func evalSlice(ctx *ExecutionContext, e *SliceExpr) (interface{}, error) {
	obj, err := evalExpr(ctx, e.Node)
	if err != nil {
		return nil, err
	}
	items := toSlice(obj)
	length := len(items)
	from, to, step := 0, length, 1
	if e.From != nil {
		f, err := evalExpr(ctx, e.From)
		if err != nil {
			return nil, err
		}
		from = toInt(f)
	}
	if e.To != nil {
		t, err := evalExpr(ctx, e.To)
		if err != nil {
			return nil, err
		}
		to = toInt(t)
	}
	if e.Step != nil {
		s, err := evalExpr(ctx, e.Step)
		if err != nil {
			return nil, err
		}
		step = toInt(s)
	}
	if from < 0 {
		from = length + from
	}
	if to < 0 {
		to = length + to
	}
	result := make([]interface{}, 0)
	if step > 0 {
		for i := from; i < to && i < length; i += step {
			if i >= 0 {
				result = append(result, items[i])
			}
		}
	} else if step < 0 {
		for i := from; i > to && i >= 0; i += step {
			result = append(result, items[i])
		}
	}
	switch obj.(type) {
	case string:
		strs := make([]string, len(result))
		for i, v := range result {
			strs[i] = fmt.Sprintf("%v", v)
		}
		return strings.Join(strs, ""), nil
	}
	return result, nil
}

func evalCompare(ctx *ExecutionContext, e *CompareExpr) (interface{}, error) {
	left, err := evalExpr(ctx, e.Expr)
	if err != nil {
		return nil, err
	}
	prev := left
	for _, op := range e.Ops {
		right, err := evalExpr(ctx, op.Right)
		if err != nil {
			return nil, err
		}
		result, err := compareOp(op.Op, prev, right)
		if err != nil {
			return nil, err
		}
		if !result {
			return false, nil
		}
		prev = right
	}
	return true, nil
}

func evalConcat(ctx *ExecutionContext, e *ConcatExpr) (interface{}, error) {
	var parts []string
	for _, n := range e.Nodes {
		val, err := evalExpr(ctx, n)
		if err != nil {
			return nil, err
		}
		parts = append(parts, stringify(val))
	}
	return strings.Join(parts, ""), nil
}

// ==================== Operator Helpers ====================

func binOp(op string, left, right interface{}) (interface{}, error) {
	if op == "+" {
		switch l := left.(type) {
		case string:
			return l + stringify(right), nil
		case SafeString:
			return SafeString(string(l) + stringify(right)), nil
		}
		switch right.(type) {
		case string:
			return stringify(left) + right.(string), nil
		}
	}

	lf := toFloat(left)
	rf := toFloat(right)
	li := toInt(left)
	ri := toInt(right)
	bothInt := isInt(left) && isInt(right)

	switch op {
	case "+":
		if bothInt {
			return li + ri, nil
		}
		return lf + rf, nil
	case "-":
		if bothInt {
			return li - ri, nil
		}
		return lf - rf, nil
	case "*":
		if bothInt {
			return li * ri, nil
		}
		return lf * rf, nil
	case "/":
		if ri == 0 || rf == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return lf / rf, nil
	case "//":
		if ri == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		if bothInt {
			return li / ri, nil
		}
		return int(lf / rf), nil
	case "%":
		if ri == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		if bothInt {
			return li % ri, nil
		}
		return int(math.Mod(lf, rf)), nil
	case "**":
		if bothInt && ri >= 0 {
			result := li
			for i := 1; i < ri; i++ {
				result *= li
			}
			return result, nil
		}
		return math.Pow(lf, rf), nil
	case "==":
		return compareEqual(left, right), nil
	case "!=":
		return !compareEqual(left, right), nil
	case "<":
		return compareValues(left, right) < 0, nil
	case ">":
		return compareValues(left, right) > 0, nil
	case "<=":
		return compareValues(left, right) <= 0, nil
	case ">=":
		return compareValues(left, right) >= 0, nil
	case "in":
		return contains(left, right), nil
	case "not in":
		return !contains(left, right), nil
	}
	return nil, fmt.Errorf("unknown operator %q", op)
}

func compareOp(op string, left, right interface{}) (bool, error) {
	switch op {
	case "==":
		return compareEqual(left, right), nil
	case "!=":
		return !compareEqual(left, right), nil
	case "<":
		return compareValues(left, right) < 0, nil
	case ">":
		return compareValues(left, right) > 0, nil
	case "<=":
		return compareValues(left, right) <= 0, nil
	case ">=":
		return compareValues(left, right) >= 0, nil
	case "in":
		return contains(left, right), nil
	case "not in":
		return !contains(left, right), nil
	}
	return false, fmt.Errorf("unknown comparison operator %q", op)
}

// ==================== Type Helpers ====================

func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != ""
	case SafeString:
		return val != ""
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	case *UndefinedValue:
		return false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() > 0
	case reflect.Ptr:
		return !rv.IsNil()
	}
	return true
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}

func isInt(v interface{}) bool {
	switch v.(type) {
	case int, int64, int32, int16, int8:
		return true
	}
	return false
}

func toSlice(v interface{}) []interface{} {
	switch val := v.(type) {
	case []interface{}:
		return val
	case string:
		items := make([]interface{}, len(val))
		for i, ch := range val {
			items[i] = string(ch)
		}
		return items
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		items := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			items[i] = rv.Index(i).Interface()
		}
		return items
	case reflect.Map:
		items := make([]interface{}, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			items = append(items, mapEntry{
				key:   iter.Key().Interface(),
				value: iter.Value().Interface(),
			})
		}
		return items
	}
	return nil
}

func compareEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b) {
		switch va := a.(type) {
		case string:
			return va == b.(string)
		case int:
			return va == b.(int)
		case int64:
			return va == b.(int64)
		case float64:
			return va == b.(float64)
		case bool:
			return va == b.(bool)
		}
	}
	if isNumeric(a) && isNumeric(b) {
		return toFloat(a) == toFloat(b)
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compareValues(a, b interface{}) int {
	if isNumeric(a) && isNumeric(b) {
		af := toFloat(a)
		bf := toFloat(b)
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	if as < bs {
		return -1
	}
	if as > bs {
		return 1
	}
	return 0
}

func isNumeric(v interface{}) bool {
	switch v.(type) {
	case int, int64, int32, int16, int8, float64, float32:
		return true
	}
	return false
}

func contains(needle, haystack interface{}) bool {
	switch h := haystack.(type) {
	case string:
		return strings.Contains(h, stringify(needle))
	case []interface{}:
		for _, item := range h {
			if compareEqual(needle, item) {
				return true
			}
		}
		return false
	case map[string]interface{}:
		key := stringify(needle)
		_, ok := h[key]
		return ok
	}
	rv := reflect.ValueOf(haystack)
	if rv.Kind() == reflect.Map {
		key := reflect.ValueOf(needle)
		if key.Type().ConvertibleTo(rv.Type().Key()) {
			return rv.MapIndex(key.Convert(rv.Type().Key())).IsValid()
		}
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			if compareEqual(needle, rv.Index(i).Interface()) {
				return true
			}
		}
	}
	return false
}
