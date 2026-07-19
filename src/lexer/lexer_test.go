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
			input: "PROGRAM END_PROGRAM VAR END_VAR INT IF THEN ELSE END_IF FOR TO DO END_FOR",
			want: []expTok{
				{PROGRAM, "PROGRAM"}, {END_PROGRAM, "END_PROGRAM"},
				{VAR, "VAR"}, {END_VAR, "END_VAR"}, {INT, "INT"},
				{IF, "IF"}, {THEN, "THEN"}, {ELSE, "ELSE"}, {END_IF, "END_IF"},
				{FOR, "FOR"}, {TO, "TO"}, {DO, "DO"}, {END_FOR, "END_FOR"},
				{EOF, ""},
			},
		},
		{
			name:  "operators and delimiters",
			input: ":= + - * / > < = : ; ( )",
			want: []expTok{
				{ASSIGN, ":="}, {PLUS, "+"}, {MINUS, "-"}, {STAR, "*"}, {SLASH, "/"},
				{GT, ">"}, {LT, "<"}, {EQ, "="}, {COLON, ":"}, {SEMICOLON, ";"},
				{LPAREN, "("}, {RPAREN, ")"},
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
