package jinja

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Token types for the Jinja2 lexer.
type tokenType int

const (
	tokText tokenType = iota
	tokVariableBegin // {{
	tokVariableEnd   // }}
	tokBlockBegin    // {%
	tokBlockEnd      // %}
	tokCommentBegin  // {#
	tokCommentEnd    // #}
	tokData          // raw text between tags
	tokName          // identifier
	tokString        // "..." or '...'
	tokInteger       // 123
	tokFloat         // 1.23
	tokBool          // true/false
	tokNone          // none
	tokOperator      // +, -, *, /, //, %, **, ==, !=, <, >, <=, >=, ~
	tokAssign        // =
	tokComma         // ,
	tokDot           // .
	tokColon         // :
	tokSemicolon     // ;
	tokPipe          // |
	tokLParen        // (
	tokRParen        // )
	tokLBrack        // [
	tokRBrack        // ]
	tokLBrace        // {
	tokRBrace        // }
	tokEOF
	tokError
)

// Token represents a single token from the template source.
type Token struct {
	Type  tokenType
	Value string
	Line  int
	Col   int
}

func (t Token) String() string {
	names := map[tokenType]string{
		tokText:         "Text",
		tokVariableBegin: "{{",
		tokVariableEnd:   "}}",
		tokBlockBegin:    "{%",
		tokBlockEnd:      "%}",
		tokCommentBegin:  "{#",
		tokCommentEnd:    "#}",
		tokData:          "Data",
		tokName:          "Name",
		tokString:        "String",
		tokInteger:       "Integer",
		tokFloat:         "Float",
		tokBool:          "Bool",
		tokNone:          "None",
		tokOperator:      "Op",
		tokAssign:        "=",
		tokComma:         ",",
		tokDot:           ".",
		tokColon:         ":",
		tokSemicolon:     ";",
		tokPipe:          "|",
		tokLParen:        "(",
		tokRParen:        ")",
		tokLBrack:        "[",
		tokRBrack:        "]",
		tokLBrace:        "{",
		tokRBrace:        "}",
		tokEOF:           "EOF",
		tokError:         "Error",
	}
	name := names[t.Type]
	if name == "" {
		name = fmt.Sprintf("Unknown(%d)", t.Type)
	}
	return fmt.Sprintf("%s(%q)@%d:%d", name, t.Value, t.Line, t.Col)
}

// lexerConfig holds lexer configuration.
type lexerConfig struct {
	leftDelim    string
	rightDelim   string
	blockDelim   string
	commentDelim string
	trimBlocks   bool
	lstripBlocks bool
}

// lexer tokenizes a Jinja2 template string.
type lexer struct {
	input    string
	pos      int
	line     int
	col      int
	startPos int
	startLine int
	startCol  int
	config   lexerConfig
}

func newLexer(input string, cfg lexerConfig) *lexer {
	return &lexer{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
		config: cfg,
	}
}

// tokenize performs the full tokenization of the input.
// Returns a list of tokens. The tokens are simplified for the parser:
// - Raw text becomes tokData tokens
// - Tag content becomes the individual tokens (names, strings, operators, etc.)
func (l *lexer) tokenize() ([]Token, error) {
	var tokens []Token

	for l.pos < len(l.input) {
		l.startPos = l.pos
		l.startLine = l.line
		l.startCol = l.col

		// Try to find the next tag opening
		nextTag := l.findNextTag()

		if nextTag == -1 {
			// No more tags, emit remaining text
			text := l.input[l.pos:]
			if text != "" {
				tokens = append(tokens, Token{Type: tokData, Value: text, Line: l.line, Col: l.col})
			}
			break
		}

		// Emit text before the tag
		text := l.input[l.pos:nextTag]
		if text != "" {
			tokens = append(tokens, Token{Type: tokData, Value: text, Line: l.line, Col: l.col})
		}

		// Advance to the tag
		l.pos = nextTag
		l.line = l.startLine + strings.Count(l.input[l.startPos:nextTag], "\n")
		lastNL := strings.LastIndex(l.input[l.startPos:nextTag], "\n")
		if lastNL >= 0 {
			l.col = nextTag - l.startPos - lastNL
		} else {
			l.col = l.startCol + (nextTag - l.startPos)
		}

		// Determine tag type
		var tagType tokenType
		var endDelim string
		rest := l.input[l.pos:]

		if strings.HasPrefix(rest, l.config.commentDelim) {
			tagType = tokCommentBegin
			endDelim = strings.TrimPrefix(l.config.commentDelim, "{") + "}"
			// Actually the end for comment is the reverse of the open: {# ... #}
			endDelim = "#" + l.config.commentDelim[len(l.config.commentDelim)-1:]
			if endDelim == "#}" {
				// default
			}
			// find #}
			endDelim = "#}"
		} else if strings.HasPrefix(rest, l.config.blockDelim) {
			tagType = tokBlockBegin
			suffix := l.config.blockDelim[len(l.config.blockDelim)-1:]
			endDelim = suffix + "}"
		} else if strings.HasPrefix(rest, l.config.leftDelim) {
			tagType = tokVariableBegin
			endDelim = "}}"
		} else {
			// Shouldn't happen, but just skip
			l.advance(1)
			continue
		}

		// Special handling for comment: skip to closing delimiter
		if tagType == tokCommentBegin {
			l.advance(len(l.config.commentDelim))
			endIdx := strings.Index(l.input[l.pos:], endDelim)
			if endIdx == -1 {
				return nil, fmt.Errorf("unclosed comment tag at line %d", l.line)
			}
			// Skip comment content (don't emit tokens for comments)
			l.advance(endIdx + len(endDelim))

			// Handle trimBlocks
			if l.config.trimBlocks {
				l.trimLeadingNewline()
			}
			continue
		}

		// Special handling for raw blocks
		if tagType == tokBlockBegin {
			afterDelim := l.pos + len(l.config.blockDelim)
			if afterDelim < len(l.input) {
				remaining := l.input[afterDelim:]
				trimmed := strings.TrimSpace(remaining)
				if strings.HasPrefix(trimmed, "raw") && (len(trimmed) == 3 || !isIdentRune(rune(trimmed[3]))) {
					l.advance(len(l.config.blockDelim))
					return l.tokenizeRaw(tokens)
				}
			}
		}

		// Emit the opening tag
		tokens = append(tokens, Token{Type: tagType, Value: l.config.leftDelim, Line: l.line, Col: l.col})
		l.advance(len(l.config.leftDelim))

		// Determine the correct end delimiter based on tag type
		if tagType == tokBlockBegin {
			endDelim = l.config.blockDelim[len(l.config.blockDelim)-1:] + "}"
		} else {
			endDelim = "}}"
		}

		// Tokenize the content inside the tag
		for l.pos < len(l.input) && !strings.HasPrefix(l.input[l.pos:], endDelim) {
			tok, err := l.nextToken()
			if err != nil {
				return nil, err
			}
			if tok.Type != tokEOF {
				tokens = append(tokens, tok)
			}
		}

		if l.pos >= len(l.input) {
			return nil, fmt.Errorf("unclosed tag at line %d", l.line)
		}

		// Emit the closing tag
		tokens = append(tokens, Token{Type: tokVariableEnd, Value: endDelim, Line: l.line, Col: l.col})
		l.advance(len(endDelim))

		// Handle trimBlocks: remove first newline after block tag
		if tagType == tokBlockBegin && l.config.trimBlocks {
			l.trimLeadingNewline()
		}
	}

	tokens = append(tokens, Token{Type: tokEOF, Value: "", Line: l.line, Col: l.col})
	return tokens, nil
}

// tokenizeRaw handles the {% raw %}...{% endraw %} construct.
func (l *lexer) tokenizeRaw(tokens []Token) ([]Token, error) {
	// Find {% endraw %}
	// Simplified: find {% endraw %}
	rawEnd := strings.Index(l.input[l.pos:], l.config.blockDelim)
	for rawEnd >= 0 {
		after := l.pos + rawEnd + len(l.config.blockDelim)
		if after < len(l.input) {
			inner := strings.TrimSpace(l.input[after:])
			if strings.HasPrefix(inner, "endraw") {
				// Found endraw
				closeEnd := strings.Index(l.input[after:], l.config.blockDelim[len(l.config.blockDelim)-1:]+"}")
				if closeEnd >= 0 {
					// Everything between raw and endraw is raw text
					rawText := l.input[l.pos : l.pos+rawEnd]
					if rawText != "" {
						tokens = append(tokens, Token{Type: tokData, Value: rawText, Line: l.line, Col: l.col})
					}
					// Skip past {% endraw %}
					l.pos = after + closeEnd + 1
					if l.pos <= len(l.input) {
						l.pos++
					}
					// Continue normal tokenization
					rest, err := l.tokenize()
					if err != nil {
						return nil, err
					}
					tokens = append(tokens, rest...)
					return tokens, nil
				}
			}
		}
		nextEnd := strings.Index(l.input[l.pos+rawEnd+1:], l.config.blockDelim)
		if nextEnd < 0 {
			break
		}
		rawEnd = rawEnd + 1 + nextEnd
	}

	// If no endraw found, treat rest as raw
	rawText := l.input[l.pos:]
	tokens = append(tokens, Token{Type: tokData, Value: rawText, Line: l.line, Col: l.col})
	l.pos = len(l.input)
	return tokens, nil
}

// findNextTag finds the position of the next template tag opening.
func (l *lexer) findNextTag() int {
	for {
		idx := -1
		candidates := []string{l.config.commentDelim, l.config.blockDelim, l.config.leftDelim}
		for _, delim := range candidates {
			i := strings.Index(l.input[l.pos:], delim)
			if i >= 0 && (idx < 0 || i < idx) {
				idx = i
			}
		}
		if idx < 0 {
			return -1
		}
		return l.pos + idx
	}
}

// nextToken reads the next token from inside a template tag.
func (l *lexer) nextToken() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: tokEOF, Line: l.line, Col: l.col}, nil
	}

	ch := l.peek()

	// Don't consume } if it's the start of the closing delimiter
	if ch == '}' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '}' {
		return Token{Type: tokEOF, Line: l.line, Col: l.col}, nil
	}
	if ch == '%' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '}' {
		return Token{Type: tokEOF, Line: l.line, Col: l.col}, nil
	}
	if ch == '#' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '}' {
		return Token{Type: tokEOF, Line: l.line, Col: l.col}, nil
	}

	// String literals
	if ch == '"' || ch == '\'' {
		return l.readString()
	}

	// Numbers
	if ch >= '0' && ch <= '9' {
		return l.readNumber()
	}

	// Identifiers and keywords
	if isIdentStart(ch) {
		return l.readName()
	}

	// Two-character operators
	if l.pos+1 < len(l.input) {
		two := l.input[l.pos : l.pos+2]
		switch two {
		case "==", "!=", "<=", ">=", "//", "**":
			l.advance(2)
			return Token{Type: tokOperator, Value: two, Line: l.line, Col: l.col}, nil
		}
	}

	// Single-character tokens
	switch ch {
	case '+', '-', '*', '/', '%':
		l.advance(1)
		return Token{Type: tokOperator, Value: string(ch), Line: l.line, Col: l.col}, nil
	case '<', '>':
		l.advance(1)
		return Token{Type: tokOperator, Value: string(ch), Line: l.line, Col: l.col}, nil
	case '=':
		l.advance(1)
		return Token{Type: tokAssign, Value: "=", Line: l.line, Col: l.col}, nil
	case ',':
		l.advance(1)
		return Token{Type: tokComma, Value: ",", Line: l.line, Col: l.col}, nil
	case '.':
		l.advance(1)
		return Token{Type: tokDot, Value: ".", Line: l.line, Col: l.col}, nil
	case ':':
		l.advance(1)
		return Token{Type: tokColon, Value: ":", Line: l.line, Col: l.col}, nil
	case ';':
		l.advance(1)
		return Token{Type: tokSemicolon, Value: ";", Line: l.line, Col: l.col}, nil
	case '|':
		l.advance(1)
		return Token{Type: tokPipe, Value: "|", Line: l.line, Col: l.col}, nil
	case '(':
		l.advance(1)
		return Token{Type: tokLParen, Value: "(", Line: l.line, Col: l.col}, nil
	case ')':
		l.advance(1)
		return Token{Type: tokRParen, Value: ")", Line: l.line, Col: l.col}, nil
	case '[':
		l.advance(1)
		return Token{Type: tokLBrack, Value: "[", Line: l.line, Col: l.col}, nil
	case ']':
		l.advance(1)
		return Token{Type: tokRBrack, Value: "]", Line: l.line, Col: l.col}, nil
	case '{':
		l.advance(1)
		return Token{Type: tokLBrace, Value: "{", Line: l.line, Col: l.col}, nil
	case '}':
		l.advance(1)
		return Token{Type: tokRBrace, Value: "}", Line: l.line, Col: l.col}, nil
	case '~':
		l.advance(1)
		return Token{Type: tokOperator, Value: "~", Line: l.line, Col: l.col}, nil
	}

	return Token{Type: tokError, Value: string(ch), Line: l.line, Col: l.col},
		fmt.Errorf("unexpected character %q at line %d, col %d", ch, l.line, l.col)
}

func (l *lexer) readString() (Token, error) {
	quote := l.peek()
	l.advance(1)
	line := l.line
	col := l.col
	var sb strings.Builder

	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '\\' && l.pos+1 < len(l.input) {
			l.advance(1)
			next := l.peek()
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(byte(next))
			}
			l.advance(1)
			continue
		}
		if ch == quote {
			l.advance(1)
			return Token{Type: tokString, Value: sb.String(), Line: line, Col: col}, nil
		}
		if ch == '\n' {
			l.line++
			l.col = 1
		}
		sb.WriteByte(byte(ch))
		l.advance(1)
	}

	return Token{Type: tokError, Line: line, Col: col},
		fmt.Errorf("unterminated string starting at line %d", line)
}

func (l *lexer) readNumber() (Token, error) {
	line := l.line
	col := l.col
	start := l.pos
	isFloat := false

	for l.pos < len(l.input) {
		ch := l.peek()
		if ch >= '0' && ch <= '9' {
			l.advance(1)
			continue
		}
		if ch == '.' && !isFloat {
			// Check if next char is a digit (to distinguish from attribute access)
			if l.pos+1 < len(l.input) && l.input[l.pos+1] >= '0' && l.input[l.pos+1] <= '9' {
				isFloat = true
				l.advance(1)
				continue
			}
		}
		break
	}

	value := l.input[start:l.pos]
	if isFloat {
		return Token{Type: tokFloat, Value: value, Line: line, Col: col}, nil
	}
	return Token{Type: tokInteger, Value: value, Line: line, Col: col}, nil
}

func (l *lexer) readName() (Token, error) {
	line := l.line
	col := l.col
	start := l.pos

	for l.pos < len(l.input) && isIdentRune(l.peek()) {
		l.advance(1)
	}

	value := l.input[start:l.pos]

	// Check for keywords
	switch value {
	case "true", "True", "TRUE":
		return Token{Type: tokBool, Value: "true", Line: line, Col: col}, nil
	case "false", "False", "FALSE":
		return Token{Type: tokBool, Value: "false", Line: line, Col: col}, nil
	case "none", "None", "NONE", "null":
		return Token{Type: tokNone, Value: "none", Line: line, Col: col}, nil
	case "not", "and", "or", "in", "not_in", "is":
		return Token{Type: tokOperator, Value: value, Line: line, Col: col}, nil
	}

	return Token{Type: tokName, Value: value, Line: line, Col: col}, nil
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if ch == '\n' {
				l.line++
				l.col = 1
			}
			l.advance(1)
		} else {
			break
		}
	}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos])
}

func (l *lexer) advance(n int) {
	for i := 0; i < n && l.pos < len(l.input); i++ {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *lexer) trimLeadingNewline() {
	if l.pos < len(l.input) && l.input[l.pos] == '\n' {
		l.pos++
		l.line++
		l.col = 1
	} else if l.pos+1 < len(l.input) && l.input[l.pos] == '\r' && l.input[l.pos+1] == '\n' {
		l.pos += 2
		l.line++
		l.col = 1
	}
}

func isIdentStart(ch rune) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isIdentRune(ch rune) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
