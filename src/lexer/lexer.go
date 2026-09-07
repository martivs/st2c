package lexer

import (
	"strings"
	"unicode"
)

type TokenType int

const (
	EOF     TokenType = iota
	ILLEGAL           // неизвестный символ или незакрытый комментарий

	IDENT    // идентификатор
	INT_LIT  // целочисленный литерал
	REAL_LIT // вещественный литерал: 3.14, 1.0E3, 1e-3

	ASSIGN    // :=
	PLUS      // +
	MINUS     // -
	STAR      // *
	SLASH     // /
	GT        // >
	LT        // <
	EQ        // =
	LE        // <=
	GE        // >=
	NE        // <>
	COLON     // :
	SEMICOLON // ;
	COMMA     // ,
	LPAREN    // (
	RPAREN    // )
	DOT       // .  (доступ к члену: inst.Out)
	ARROW     // => (привязка выхода: out => y)

	PROGRAM
	END_PROGRAM
	VAR
	VAR_INPUT
	VAR_OUTPUT
	VAR_IN_OUT
	VAR_TEMP
	END_VAR
	CONSTANT
	RETAIN
	INT
	REAL
	IF
	THEN
	ELSE
	END_IF
	FOR
	TO
	BY
	DO
	END_FOR
	FUNCTION
	END_FUNCTION
	FUNCTION_BLOCK
	END_FUNCTION_BLOCK
)

// tokenNames — имена типов токенов для печати. Ключевые слова сюда не
// вписываются вручную — они добавляются из keywords в init(), чтобы каждое
// новое слово правилось в одном месте. Всё остальное (литералы, операторы)
// вписывается сюда руками, иначе String() вернёт UNKNOWN в сообщениях парсера.
var tokenNames = map[TokenType]string{
	EOF: "EOF", ILLEGAL: "ILLEGAL",
	IDENT: "IDENT", INT_LIT: "INT_LIT", REAL_LIT: "REAL_LIT",
	ASSIGN: ":=", PLUS: "+", MINUS: "-", STAR: "*", SLASH: "/",
	GT: ">", LT: "<", EQ: "=", LE: "<=", GE: ">=", NE: "<>",
	COLON: ":", SEMICOLON: ";", COMMA: ",",
	LPAREN: "(", RPAREN: ")",
	DOT: ".", ARROW: "=>",
}

func init() {
	for lit, tt := range keywords {
		tokenNames[tt] = lit
	}
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

var keywords = map[string]TokenType{
	"PROGRAM": PROGRAM, "END_PROGRAM": END_PROGRAM,
	"VAR": VAR, "VAR_INPUT": VAR_INPUT, "VAR_OUTPUT": VAR_OUTPUT,
	"VAR_IN_OUT": VAR_IN_OUT, "VAR_TEMP": VAR_TEMP, "END_VAR": END_VAR,
	"CONSTANT": CONSTANT, "RETAIN": RETAIN, "INT": INT, "REAL": REAL,
	"IF": IF, "THEN": THEN, "ELSE": ELSE, "END_IF": END_IF,
	"FOR": FOR, "TO": TO, "BY": BY, "DO": DO, "END_FOR": END_FOR,
	"FUNCTION": FUNCTION, "END_FUNCTION": END_FUNCTION,
	"FUNCTION_BLOCK": FUNCTION_BLOCK, "END_FUNCTION_BLOCK": END_FUNCTION_BLOCK,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int // колонка начала токена, 1-based, в рунах
}

type Lexer struct {
	input []rune // NB: using []rune to handle Unicode characters properly
	// (4 bytes per rune!)
	pos       int
	line      int
	lineStart int // индекс в input, с которого начинается текущая строка (для колонок)
}

func New(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0, line: 1, lineStart: 0}
}

// col — колонка текущей позиции (1-based).
func (l *Lexer) col() int { return l.pos - l.lineStart + 1 }

func (l *Lexer) NextToken() Token {
	// Пробелы и комментарии могут чередоваться ((* a *) // b), поэтому цикл.
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			return Token{Type: EOF, Line: l.line, Col: l.col()}
		}
		ch := l.input[l.pos]
		if ch == '(' && l.peek() == '*' {
			if tok, ok := l.skipBlockComment(); !ok {
				return tok // незакрытый комментарий → ILLEGAL
			}
			continue
		}
		if ch == '/' && l.peek() == '/' {
			l.skipLineComment()
			continue
		}
		break
	}

	ch := l.input[l.pos]
	line, col := l.line, l.col()

	switch {
	case ch == ':' && l.peek() == '=':
		l.pos += 2
		return Token{Type: ASSIGN, Literal: ":=", Line: line, Col: col}
	case ch == ':':
		l.pos++
		return Token{Type: COLON, Literal: ":", Line: line, Col: col}
	case ch == ';':
		l.pos++
		return Token{Type: SEMICOLON, Literal: ";", Line: line, Col: col}
	case ch == ',':
		l.pos++
		return Token{Type: COMMA, Literal: ",", Line: line, Col: col}
	case ch == '(':
		l.pos++
		return Token{Type: LPAREN, Literal: "(", Line: line, Col: col}
	case ch == ')':
		l.pos++
		return Token{Type: RPAREN, Literal: ")", Line: line, Col: col}
	case ch == '+':
		l.pos++
		return Token{Type: PLUS, Literal: "+", Line: line, Col: col}
	case ch == '-':
		l.pos++
		return Token{Type: MINUS, Literal: "-", Line: line, Col: col}
	case ch == '*':
		l.pos++
		return Token{Type: STAR, Literal: "*", Line: line, Col: col}
	case ch == '/':
		l.pos++
		return Token{Type: SLASH, Literal: "/", Line: line, Col: col}
	case ch == '>' && l.peek() == '=':
		l.pos += 2
		return Token{Type: GE, Literal: ">=", Line: line, Col: col}
	case ch == '>':
		l.pos++
		return Token{Type: GT, Literal: ">", Line: line, Col: col}
	case ch == '<' && l.peek() == '=':
		l.pos += 2
		return Token{Type: LE, Literal: "<=", Line: line, Col: col}
	case ch == '<' && l.peek() == '>':
		l.pos += 2
		return Token{Type: NE, Literal: "<>", Line: line, Col: col}
	case ch == '<':
		l.pos++
		return Token{Type: LT, Literal: "<", Line: line, Col: col}
	// Ветка `=>` обязана стоять ДО ветки `=`: Go проверяет case по порядку,
	// при обратном порядке `=>` молча разберётся как `=` и `>` (ср. `<=`/`<>`
	// перед `<` выше).
	case ch == '=' && l.peek() == '>':
		l.pos += 2
		return Token{Type: ARROW, Literal: "=>", Line: line, Col: col}
	case ch == '=':
		l.pos++
		return Token{Type: EQ, Literal: "=", Line: line, Col: col}
	case ch == '.':
		l.pos++
		return Token{Type: DOT, Literal: ".", Line: line, Col: col}
	case unicode.IsLetter(ch) || ch == '_':
		return l.readIdent()
	case unicode.IsDigit(ch):
		return l.readNumber()
	default:
		l.pos++
		return Token{Type: ILLEGAL, Literal: string(ch), Line: line, Col: col}
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

// newline фиксирует переход на новую строку: инкремент номера и запоминание
// начала строки для подсчёта колонок. Вызывается на каждом '\n', где бы он
// ни встретился (пробелы, блочный комментарий).
func (l *Lexer) newline() {
	l.line++
	l.pos++
	l.lineStart = l.pos
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\n' {
			l.newline()
		} else if ch == ' ' || ch == '\t' || ch == '\r' {
			// Строка считается по '\n'; одиночный '\r' просто пропускается —
			// корректно и для LF, и для CRLF.
			l.pos++
		} else {
			break
		}
	}
}

// skipBlockComment пропускает блочный комментарий `(* ... *)` с поддержкой
// вложенности (допустима по 3-й редакции IEC 61131-3). Если комментарий не
// закрыт до конца файла, возвращает (ILLEGAL-токен с позицией его начала,
// false).
func (l *Lexer) skipBlockComment() (Token, bool) {
	line, col := l.line, l.col()
	l.pos += 2 // съесть "(*"
	depth := 1
	for l.pos < len(l.input) {
		switch {
		case l.input[l.pos] == '(' && l.peek() == '*':
			depth++
			l.pos += 2
		case l.input[l.pos] == '*' && l.peek() == ')':
			depth--
			l.pos += 2
			if depth == 0 {
				return Token{}, true
			}
		case l.input[l.pos] == '\n':
			l.newline()
		default:
			l.pos++
		}
	}
	return Token{Type: ILLEGAL, Literal: "unterminated comment (*", Line: line, Col: col}, false
}

// skipLineComment пропускает `// ...` до конца строки; сам '\n' не съедается —
// его учтёт skipWhitespace.
func (l *Lexer) skipLineComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) readIdent() Token {
	line, col := l.line, l.col()
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	literal := string(l.input[start:l.pos])
	upper := strings.ToUpper(literal)
	if tt, ok := keywords[upper]; ok {
		return Token{Type: tt, Literal: upper, Line: line, Col: col}
	}
	return Token{Type: IDENT, Literal: literal, Line: line, Col: col}
}

// readNumber — числовой литерал по грамматике IEC:
//
//	integer                      → INT_LIT
//	integer '.' integer [exp]    → REAL_LIT
//	integer exp                  → REAL_LIT,   exp = ('e'|'E') ['+'|'-'] integer
//
// Дробная часть и экспонента принимаются с откатом: `.` съедается, только
// если за ней идёт цифра (иначе `1..10` будущего ARRAY и `1.` остаются
// `INT_LIT DOT ...`), `e`/`E` — только если за ней идёт цифра или знак с
// цифрой (иначе `1EXIT` остаётся `INT_LIT IDENT`). `1.` без дробной части по
// IEC не REAL.
func (l *Lexer) readNumber() Token {
	line, col := l.line, l.col()
	start := l.pos
	l.skipDigits()
	isReal := false

	// Дробная часть: '.' digit+
	if l.cur() == '.' && unicode.IsDigit(l.peek()) {
		l.pos++ // '.'
		l.skipDigits()
		isReal = true
	}

	// Экспонента: ('e'|'E') ['+'|'-'] digit+
	if e := l.cur(); e == 'e' || e == 'E' {
		next := l.peek()
		if unicode.IsDigit(next) {
			l.pos++ // 'e'
			l.skipDigits()
			isReal = true
		} else if (next == '+' || next == '-') && l.pos+2 < len(l.input) && unicode.IsDigit(l.input[l.pos+2]) {
			l.pos += 2 // 'e' и знак
			l.skipDigits()
			isReal = true
		}
	}

	tt := INT_LIT
	if isReal {
		tt = REAL_LIT
	}
	return Token{Type: tt, Literal: string(l.input[start:l.pos]), Line: line, Col: col}
}

// cur — текущая руна либо 0 на конце входа.
func (l *Lexer) cur() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// skipDigits продвигает позицию за подряд идущие цифры.
func (l *Lexer) skipDigits() {
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.pos++
	}
}
