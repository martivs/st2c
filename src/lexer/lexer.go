package lexer

import (
	"strings"
	"unicode"
)

type TokenType int

const (
	EOF     TokenType = iota
	ILLEGAL           // неизвестный символ

	IDENT   // идентификатор
	INT_LIT // целочисленный литерал

	ASSIGN    // :=
	PLUS      // +
	MINUS     // -
	STAR      // *
	SLASH     // /
	GT        // >
	LT        // <
	EQ        // =
	COLON     // :
	SEMICOLON // ;
	LPAREN    // (
	RPAREN    // )

	PROGRAM
	END_PROGRAM
	VAR
	END_VAR
	INT
	IF
	THEN
	ELSE
	END_IF
	FOR
	TO
	DO
	END_FOR
)

var tokenNames = map[TokenType]string{
	EOF: "EOF", ILLEGAL: "ILLEGAL",
	IDENT: "IDENT", INT_LIT: "INT_LIT",
	ASSIGN: ":=", PLUS: "+", MINUS: "-", STAR: "*", SLASH: "/",
	GT: ">", LT: "<", EQ: "=", COLON: ":", SEMICOLON: ";",
	LPAREN: "(", RPAREN: ")",
	PROGRAM: "PROGRAM", END_PROGRAM: "END_PROGRAM",
	VAR: "VAR", END_VAR: "END_VAR", INT: "INT",
	IF: "IF", THEN: "THEN", ELSE: "ELSE", END_IF: "END_IF",
	FOR: "FOR", TO: "TO", DO: "DO", END_FOR: "END_FOR",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

var keywords = map[string]TokenType{
	"PROGRAM": PROGRAM, "END_PROGRAM": END_PROGRAM,
	"VAR": VAR, "END_VAR": END_VAR, "INT": INT,
	"IF": IF, "THEN": THEN, "ELSE": ELSE, "END_IF": END_IF,
	"FOR": FOR, "TO": TO, "DO": DO, "END_FOR": END_FOR,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
}

type Lexer struct {
	input []rune // NB: using []rune to handle Unicode characters properly
	// (4 bytes per rune!)
	pos  int
	line int
}

func New(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0, line: 1}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: EOF, Line: l.line}
	}

	ch := l.input[l.pos]

	switch {
	case ch == ':' && l.peek() == '=':
		l.pos += 2
		return Token{Type: ASSIGN, Literal: ":=", Line: l.line} // TODO: Literal duplicates tokenNames; unify when adding new token types
	case ch == ':':
		l.pos++
		return Token{Type: COLON, Literal: ":", Line: l.line}
	case ch == ';':
		l.pos++
		return Token{Type: SEMICOLON, Literal: ";", Line: l.line}
	case ch == '(':
		l.pos++
		return Token{Type: LPAREN, Literal: "(", Line: l.line}
	case ch == ')':
		l.pos++
		return Token{Type: RPAREN, Literal: ")", Line: l.line}
	case ch == '+':
		l.pos++
		return Token{Type: PLUS, Literal: "+", Line: l.line}
	case ch == '-':
		l.pos++
		return Token{Type: MINUS, Literal: "-", Line: l.line}
	case ch == '*':
		l.pos++
		return Token{Type: STAR, Literal: "*", Line: l.line}
	case ch == '/':
		l.pos++
		return Token{Type: SLASH, Literal: "/", Line: l.line}
	case ch == '>':
		l.pos++
		return Token{Type: GT, Literal: ">", Line: l.line}
	case ch == '<':
		l.pos++
		return Token{Type: LT, Literal: "<", Line: l.line}
	case ch == '=':
		l.pos++
		return Token{Type: EQ, Literal: "=", Line: l.line}
	case unicode.IsLetter(ch) || ch == '_':
		return l.readIdent()
	case unicode.IsDigit(ch):
		return l.readInt()
	default:
		l.pos++
		return Token{Type: ILLEGAL, Literal: string(ch), Line: l.line}
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\n' { // FIXME
			l.line++
			l.pos++
		} else if ch == ' ' || ch == '\t' || ch == '\r' {
			l.pos++
		} else {
			break
		}
	}
}

func (l *Lexer) readIdent() Token {
	line := l.line
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	literal := string(l.input[start:l.pos])
	upper := strings.ToUpper(literal)
	if tt, ok := keywords[upper]; ok {
		return Token{Type: tt, Literal: upper, Line: line}
	}
	return Token{Type: IDENT, Literal: literal, Line: line}
}

func (l *Lexer) readInt() Token {
	line := l.line
	start := l.pos
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.pos++
	}
	return Token{Type: INT_LIT, Literal: string(l.input[start:l.pos]), Line: line}
}
