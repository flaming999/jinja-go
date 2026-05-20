package jinja

// AST node types for the Jinja2 template parse tree.

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeType() string
	Position() Position
}

// Position represents a source position in the template.
type Position struct {
	Line   int
	Column int
}

// Expr is the interface for all expression types.
type Expr interface {
	exprType() string
}

// --- Output ---

// OutputNode represents a literal text segment.
type OutputNode struct {
	Pos  Position
	Text string
}

func (n *OutputNode) nodeType() string { return "Output" }
func (n *OutputNode) Position() Position { return n.Pos }

// --- Expression ---

// ExprNode represents a {{ expr }} tag.
type ExprNode struct {
	Pos  Position
	Expr Expr
}

func (n *ExprNode) nodeType() string { return "Expr" }
func (n *ExprNode) Position() Position { return n.Pos }

// --- Control structures ---

// IfNode represents an {% if ... %} block.
type IfNode struct {
	Pos          Position
	Cond         Expr
	body         []Node
	elifBranches []ElifBranch
	elseBody     []Node
}

func (n *IfNode) nodeType() string { return "If" }
func (n *IfNode) Position() Position { return n.Pos }

// ElifBranch represents an elif clause.
type ElifBranch struct {
	Cond Expr
	body []Node
}

// ForNode represents a {% for ... in ... %} block.
type ForNode struct {
	Pos       Position
	Var       string
	Key       string // empty for simple loops, set for "for key, val in ..."
	Iter      Expr
	body      []Node
	elseBody  []Node
	Recursive bool
}

func (n *ForNode) nodeType() string { return "For" }
func (n *ForNode) Position() Position { return n.Pos }

// --- Template inheritance ---

// ExtendsNode represents {% extends "template" %}.
type ExtendsNode struct {
	Pos          Position
	templateName string
}

func (n *ExtendsNode) nodeType() string { return "Extends" }
func (n *ExtendsNode) Position() Position { return n.Pos }

// BlockNode represents a {% block name %}...{% endblock %}.
type BlockNode struct {
	Pos   Position
	name  string
	body  []Node
	super *BlockNode
}

func (n *BlockNode) nodeType() string { return "Block" }
func (n *BlockNode) Position() Position { return n.Pos }

// IncludeNode represents {% include "template" %}.
type IncludeNode struct {
	Pos           Position
	templateName  string
	IgnoreMissing bool
	WithContext   bool
}

func (n *IncludeNode) nodeType() string { return "Include" }
func (n *IncludeNode) Position() Position { return n.Pos }

// FromImportNode represents {% from "module" import name1, name2 %}.
type FromImportNode struct {
	Pos          Position
	templateName string
	Names        []string
	Aliases      []string
}

func (n *FromImportNode) nodeType() string { return "FromImport" }
func (n *FromImportNode) Position() Position { return n.Pos }

// ImportNode represents {% import "module" as name %}.
type ImportNode struct {
	Pos          Position
	templateName string
	Alias        string
}

func (n *ImportNode) nodeType() string { return "Import" }
func (n *ImportNode) Position() Position { return n.Pos }

// --- Macros ---

// MacroNode represents a {% macro name(args) %}...{% endmacro %} definition.
type MacroNode struct {
	Pos           Position
	Name          string
	Args          []MacroArg
	Defaults      []Expr
	CatchVarargs  bool
	CatchKwargs   bool
	body          []Node
	Caller        bool
}

func (n *MacroNode) nodeType() string { return "Macro" }
func (n *MacroNode) Position() Position { return n.Pos }

// MacroArg is a macro argument definition.
type MacroArg struct {
	Name    string
	Default Expr // nil if no default
}

// CallBlockNode represents {% call func() %}...{% endcall %}.
type CallBlockNode struct {
	Pos      Position
	CallExpr Expr
	body     []Node
}

func (n *CallBlockNode) nodeType() string { return "CallBlock" }
func (n *CallBlockNode) Position() Position { return n.Pos }

// --- Assignment ---

// SetNode represents {% set x = expr %}.
type SetNode struct {
	Pos   Position
	Name  string
	Value Expr
}

func (n *SetNode) nodeType() string { return "Set" }
func (n *SetNode) Position() Position { return n.Pos }

// SetBlockNode represents {% set x %}...{% endset %}.
type SetBlockNode struct {
	Pos  Position
	Name string
	body []Node
}

func (n *SetBlockNode) nodeType() string { return "SetBlock" }
func (n *SetBlockNode) Position() Position { return n.Pos }

// --- Misc ---

// CommentNode represents a {# comment #} (stripped by lexer).
type CommentNode struct {
	Pos  Position
	Text string
}

func (n *CommentNode) nodeType() string { return "Comment" }
func (n *CommentNode) Position() Position { return n.Pos }

// RawNode represents {% raw %}...{% endraw %}.
type RawNode struct {
	Pos  Position
	Text string
}

func (n *RawNode) nodeType() string { return "Raw" }
func (n *RawNode) Position() Position { return n.Pos }

// WithNode represents {% with x=y %}...{% endwith %}.
type WithNode struct {
	Pos         Position
	assignments []SetNode
	body        []Node
}

func (n *WithNode) nodeType() string { return "With" }
func (n *WithNode) Position() Position { return n.Pos }

// DoNode represents {% do expr %}.
type DoNode struct {
	Pos  Position
	Expr Expr
}

func (n *DoNode) nodeType() string { return "Do" }
func (n *DoNode) Position() Position { return n.Pos }

// ==================== Expression Types ====================

// LiteralExpr represents a literal value (string, number, bool, nil).
type LiteralExpr struct {
	Value interface{}
}

func (e *LiteralExpr) exprType() string { return "Literal" }

// NameExpr represents a variable reference.
type NameExpr struct {
	Name string
}

func (e *NameExpr) exprType() string { return "Name" }

// GetAttrExpr represents attribute access (obj.attr or obj["key"]).
type GetAttrExpr struct {
	Node Expr
	Attr string
	// Dynamic attribute/key for obj[expr]
	DynamicKey Expr
	IsCall     bool // true if this is obj.method() call position
}

func (e *GetAttrExpr) exprType() string { return "GetAttr" }

// GetItemExpr represents subscript access (obj[key]).
type GetItemExpr struct {
	Node Expr
	Idx  Expr
}

func (e *GetItemExpr) exprType() string { return "GetItem" }

// CallExpr represents a function call f(args, kwargs).
type CallExpr struct {
	Func   Expr
	Args   []Expr
	kwargs []CallKwarg
}

func (e *CallExpr) exprType() string { return "Call" }

// CallKwarg is a keyword argument in a call expression.
type CallKwarg struct {
	Key   string
	Value Expr
}

// FilterExpr represents expr | filter(args).
type FilterExpr struct {
	Node Expr
	Name string
	Args []Expr
}

func (e *FilterExpr) exprType() string { return "Filter" }

// TestExpr represents expr is test(args).
type TestExpr struct {
	Node Expr
	Name string
	Args []Expr
	Neg  bool // true for expr is not test
}

func (e *TestExpr) exprType() string { return "Test" }

// BinOpExpr represents a binary operation (a + b, a == b, etc.).
type BinOpExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (e *BinOpExpr) exprType() string { return "BinOp" }

// UnaryExpr represents a unary operation (not x, -x, +x).
type UnaryExpr struct {
	Op   string
	Node Expr
}

func (e *UnaryExpr) exprType() string { return "Unary" }

// CondExpr represents a ternary: expr1 if cond else expr2.
type CondExpr struct {
	True  Expr
	Cond  Expr
	False Expr
}

func (e *CondExpr) exprType() string { return "CondExpr" }

// ListExpr represents a list literal [a, b, c].
type ListExpr struct {
	Items []Expr
}

func (e *ListExpr) exprType() string { return "List" }

// DictExpr represents a dict literal {k: v, ...}.
type DictExpr struct {
	Pairs []DictPair
}

func (e *DictExpr) exprType() string { return "Dict" }

// DictPair is a key-value pair in a dict literal.
type DictPair struct {
	Key   Expr
	Value Expr
}

// TupleExpr represents a tuple literal (a, b, c).
type TupleExpr struct {
	Items []Expr
}

func (e *TupleExpr) exprType() string { return "Tuple" }

// SliceExpr represents slice notation obj[start:stop:step].
type SliceExpr struct {
	Node Expr
	From Expr
	To   Expr
	Step Expr
}

func (e *SliceExpr) exprType() string { return "Slice" }

// CompareExpr represents chained comparisons (a < b < c).
type CompareExpr struct {
	Expr Expr
	Ops  []CompareOp
}

func (e *CompareExpr) exprType() string { return "Compare" }

// CompareOp is a single comparison in a chain.
type CompareOp struct {
	Op    string
	Right Expr
}

// ConcatExpr represents the ~ (tilde/string concatenation) operator.
type ConcatExpr struct {
	Nodes []Expr
}

func (e *ConcatExpr) exprType() string { return "Concat" }
