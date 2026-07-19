package lexer

import "testing"

// expTok — ожидаемая пара (тип, литерал) для table-driven тестов.
type expTok struct {
	typ TokenType
	lit string
}

// collect прогоняет вход через лексер и собирает токены до EOF включительно.
// Страховка от зацикливания — жёсткий предел итераций.
func collect(t *testing.T, input string) []Token {
	t.Helper()
	l := New(input)
	var toks []Token
	for i := 0; i < 10000; i++ {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == EOF {
			return toks
		}
	}
	t.Fatal("lexer did not reach EOF in 10000 tokens")
	return nil
}

// assertTokens сверяет поток токенов с ожидаемым списком (типы и литералы).
func assertTokens(t *testing.T, input string, want []expTok) {
	t.Helper()
	got := collect(t, input)
	if len(got) != len(want) {
		t.Fatalf("token count: got %d, want %d\ngot: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Type != want[i].typ || got[i].Literal != want[i].lit {
			t.Errorf("token %d: got (%s %q), want (%s %q)",
				i, got[i].Type, got[i].Literal, want[i].typ, want[i].lit)
		}
	}
}

// TestAllTokens покрывает все текущие типы токенов.
func TestAllTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []expTok
	}{
		{
			name:  "keywords",
			input: "PROGRAM END_PROGRAM VAR END_VAR INT IF THEN ELSE END_IF FOR TO BY DO END_FOR",
			want: []expTok{
				{PROGRAM, "PROGRAM"}, {END_PROGRAM, "END_PROGRAM"},
				{VAR, "VAR"}, {END_VAR, "END_VAR"}, {INT, "INT"},
				{IF, "IF"}, {THEN, "THEN"}, {ELSE, "ELSE"}, {END_IF, "END_IF"},
				{FOR, "FOR"}, {TO, "TO"}, {BY, "BY"}, {DO, "DO"}, {END_FOR, "END_FOR"},
				{EOF, ""},
			},
		},
		{
			name:  "operators and delimiters",
			input: ":= + - * / > < = <= >= <> : ; ( )",
			want: []expTok{
				{ASSIGN, ":="}, {PLUS, "+"}, {MINUS, "-"}, {STAR, "*"}, {SLASH, "/"},
				{GT, ">"}, {LT, "<"}, {EQ, "="},
				{LE, "<="}, {GE, ">="}, {NE, "<>"},
				{COLON, ":"}, {SEMICOLON, ";"},
				{LPAREN, "("}, {RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "two-char operators without spaces",
			input: "x<=3 y>=4 z<>5 a<b>c",
			want: []expTok{
				{IDENT, "x"}, {LE, "<="}, {INT_LIT, "3"},
				{IDENT, "y"}, {GE, ">="}, {INT_LIT, "4"},
				{IDENT, "z"}, {NE, "<>"}, {INT_LIT, "5"},
				{IDENT, "a"}, {LT, "<"}, {IDENT, "b"}, {GT, ">"}, {IDENT, "c"},
				{EOF, ""},
			},
		},
		{
			name:  "for with by",
			input: "FOR i := 1 TO 10 BY 2 DO",
			want: []expTok{
				{FOR, "FOR"}, {IDENT, "i"}, {ASSIGN, ":="}, {INT_LIT, "1"},
				{TO, "TO"}, {INT_LIT, "10"}, {BY, "BY"}, {INT_LIT, "2"},
				{DO, "DO"},
				{EOF, ""},
			},
		},
		{
			name:  "identifiers and literals",
			input: "x _tmp x2 count_1 42 007",
			want: []expTok{
				{IDENT, "x"}, {IDENT, "_tmp"}, {IDENT, "x2"}, {IDENT, "count_1"},
				{INT_LIT, "42"}, {INT_LIT, "007"},
				{EOF, ""},
			},
		},
		{
			name:  "assignment statement",
			input: "sum := sum + x;",
			want: []expTok{
				{IDENT, "sum"}, {ASSIGN, ":="}, {IDENT, "sum"}, {PLUS, "+"}, {IDENT, "x"},
				{SEMICOLON, ";"},
				{EOF, ""},
			},
		},
		{
			name:  "colon vs assign",
			input: "x : INT := 5",
			want: []expTok{
				{IDENT, "x"}, {COLON, ":"}, {INT, "INT"}, {ASSIGN, ":="}, {INT_LIT, "5"},
				{EOF, ""},
			},
		},
		{
			name:  "illegal character",
			input: "x ? y",
			want: []expTok{
				{IDENT, "x"}, {ILLEGAL, "?"}, {IDENT, "y"},
				{EOF, ""},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  []expTok{{EOF, ""}},
		},
		{
			name:  "whitespace only",
			input: " \t\r\n  \n",
			want:  []expTok{{EOF, ""}},
		},
		{
			name:  "block comment",
			input: "x := (* пояснение *) 1;",
			want: []expTok{
				{IDENT, "x"}, {ASSIGN, ":="}, {INT_LIT, "1"}, {SEMICOLON, ";"},
				{EOF, ""},
			},
		},
		{
			name:  "nested block comment",
			input: "a (* outer (* inner *) still outer *) b",
			want: []expTok{
				{IDENT, "a"}, {IDENT, "b"},
				{EOF, ""},
			},
		},
		{
			name:  "comment with stars and parens inside",
			input: "a (* * ( ) ** *) b",
			want: []expTok{
				{IDENT, "a"}, {IDENT, "b"},
				{EOF, ""},
			},
		},
		{
			name:  "line comment",
			input: "x := 1; // хвост строки := не токен\ny := 2;",
			want: []expTok{
				{IDENT, "x"}, {ASSIGN, ":="}, {INT_LIT, "1"}, {SEMICOLON, ";"},
				{IDENT, "y"}, {ASSIGN, ":="}, {INT_LIT, "2"}, {SEMICOLON, ";"},
				{EOF, ""},
			},
		},
		{
			name:  "line comment at eof without newline",
			input: "x // comment",
			want: []expTok{
				{IDENT, "x"},
				{EOF, ""},
			},
		},
		{
			name:  "adjacent comments",
			input: "(* a *) // b\n(* c *) x",
			want: []expTok{
				{IDENT, "x"},
				{EOF, ""},
			},
		},
		{
			name:  "single paren and slash are still tokens",
			input: "(a / b)",
			want: []expTok{
				{LPAREN, "("}, {IDENT, "a"}, {SLASH, "/"}, {IDENT, "b"}, {RPAREN, ")"},
				{EOF, ""},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertTokens(t, tc.input, tc.want)
		})
	}
}

// TestKeywordsCaseInsensitive: ключевые слова регистронезависимы, литерал
// приводится к верхнему регистру; идентификаторы сохраняют написание.
func TestKeywordsCaseInsensitive(t *testing.T) {
	assertTokens(t, "program If eNd_If foR Sum", []expTok{
		{PROGRAM, "PROGRAM"}, {IF, "IF"}, {END_IF, "END_IF"}, {FOR, "FOR"},
		{IDENT, "Sum"},
		{EOF, ""},
	})
}

// TestLineNumbers: номер строки растёт на \n (LF и CRLF).
func TestLineNumbers(t *testing.T) {
	input := "PROGRAM P\nx := 1;\r\n\ny := 2;"
	wantLines := map[string]int{
		"PROGRAM": 1, "P": 1,
		"x": 2, "1": 2,
		"y": 4, "2": 4,
	}
	for _, tok := range collect(t, input) {
		if want, ok := wantLines[tok.Literal]; ok && tok.Line != want {
			t.Errorf("token %q: line %d, want %d", tok.Literal, tok.Line, want)
		}
	}
}

// TestLineNumbersInsideBlockComment: переводы строк внутри (* ... *)
// учитываются в нумерации.
func TestLineNumbersInsideBlockComment(t *testing.T) {
	input := "a (* строка 1\nстрока 2\nстрока 3 *) b\nc"
	wantLines := map[string]int{"a": 1, "b": 3, "c": 4}
	for _, tok := range collect(t, input) {
		if want, ok := wantLines[tok.Literal]; ok && tok.Line != want {
			t.Errorf("token %q: line %d, want %d", tok.Literal, tok.Line, want)
		}
	}
}

// TestUnterminatedBlockComment: незакрытый (* даёт ILLEGAL с позицией начала
// комментария.
func TestUnterminatedBlockComment(t *testing.T) {
	for _, input := range []string{"x (* no end", "x (* outer (* inner *) no end"} {
		l := New(input)
		l.NextToken() // x
		tok := l.NextToken()
		if tok.Type != ILLEGAL {
			t.Errorf("input %q: got %s %q, want ILLEGAL", input, tok.Type, tok.Literal)
		}
		if tok.Line != 1 || tok.Col != 3 {
			t.Errorf("input %q: position %d:%d, want 1:3", input, tok.Line, tok.Col)
		}
	}
}

// TestColumns: колонка — 1-based позиция начала токена в строке, сбрасывается
// на каждом переводе строки.
func TestColumns(t *testing.T) {
	input := "x := 10;\n  sum := sum + 1;"
	want := []struct {
		lit  string
		line int
		col  int
	}{
		{"x", 1, 1}, {":=", 1, 3}, {"10", 1, 6}, {";", 1, 8},
		{"sum", 2, 3}, {":=", 2, 7}, {"sum", 2, 10}, {"+", 2, 14}, {"1", 2, 16}, {";", 2, 17},
	}
	toks := collect(t, input)
	if len(toks) != len(want)+1 { // +1 за EOF
		t.Fatalf("token count: got %d, want %d", len(toks), len(want)+1)
	}
	for i, w := range want {
		if toks[i].Literal != w.lit || toks[i].Line != w.line || toks[i].Col != w.col {
			t.Errorf("token %d: got %q at %d:%d, want %q at %d:%d",
				i, toks[i].Literal, toks[i].Line, toks[i].Col, w.lit, w.line, w.col)
		}
	}
}
