// Тесты sema: вход-программа → ожидаемые подстроки ошибок (с позициями)
// либо их отсутствие.
package sema_test

import (
	"strings"
	"testing"

	"st2c/src/lexer"
	"st2c/src/parser"
	"st2c/src/sema"
)

// check разбирает исходник (разбор обязан пройти) и возвращает ошибки sema.
func check(t *testing.T, src string) []error {
	t.Helper()
	p := parser.New(lexer.New(src))
	sf, err := p.ParseSourceFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return sema.Check(sf)
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string // подстроки ошибок в порядке следования; пусто — ошибок нет
	}{
		{
			name: "корректная программа",
			src: `PROGRAM P
VAR x : INT := 5; sum : INT; END_VAR
sum := 0;
FOR x := 1 TO 10 BY 2 DO
    sum := sum + x;
END_FOR
IF sum > 10 THEN sum := -32768; END_IF
END_PROGRAM`,
		},
		{
			name: "необъявленная переменная в выражении",
			src:  "PROGRAM P\nVAR y : INT; END_VAR\ny := x + 1;\nEND_PROGRAM",
			want: []string{`line 3:6: undeclared variable "x"`},
		},
		{
			name: "необъявленная цель присваивания",
			src:  "PROGRAM P\nx := 1;\nEND_PROGRAM",
			want: []string{`line 2:1: undeclared variable "x"`},
		},
		{
			name: "необъявленная переменная цикла FOR",
			src:  "PROGRAM P\nFOR i := 1 TO 3 DO\nEND_FOR\nEND_PROGRAM",
			want: []string{`line 2:5: undeclared variable "i"`},
		},
		{
			name: "регистронезависимость: Sum и sum — одна переменная",
			src:  "PROGRAM P\nVAR Sum : INT; END_VAR\nsum := 1;\nSUM := sUm + 1;\nEND_PROGRAM",
		},
		{
			name: "дубликат объявления в разном регистре",
			src:  "PROGRAM P\nVAR Sum : INT;\nsum : INT;\nEND_VAR\nEND_PROGRAM",
			want: []string{`line 3:1: duplicate declaration of "sum" (first declared as "Sum" at line 2)`},
		},
		{
			name: "дубликат в списке имён одного объявления",
			src:  "PROGRAM P\nVAR a, a : INT; END_VAR\nEND_PROGRAM",
			want: []string{`line 2:8: duplicate declaration of "a"`},
		},
		{
			// Позиция ошибки — колонка самого дубликата, а не первого имени
			// объявления (Names хранит []*Identifier с токеном каждого имени).
			name: "позиция дубликата в списке имён — само имя, не первое в списке",
			src:  "PROGRAM P\nVAR alpha : INT; END_VAR\nVAR aaa, bbb, alpha : INT; END_VAR\nEND_PROGRAM",
			want: []string{`line 3:15: duplicate declaration of "alpha" (first declared as "alpha" at line 2)`},
		},
		{
			name: "дубликат между блоками разных видов",
			src:  "PROGRAM P\nVAR x : INT; END_VAR\nVAR_INPUT x : INT; END_VAR\nEND_PROGRAM",
			want: []string{`line 3:11: duplicate declaration of "x"`},
		},
		{
			name: "инициализатор вне диапазона INT",
			src:  "PROGRAM P\nVAR x : INT := 100000; END_VAR\nEND_PROGRAM",
			want: []string{`line 2:16: literal 100000 out of range for INT (-32768..32767)`},
		},
		{
			name: "присваивание вне диапазона INT",
			src:  "PROGRAM P\nVAR x : INT; END_VAR\nx := 40000;\nEND_PROGRAM",
			want: []string{`line 3:6: literal 40000 out of range for INT`},
		},
		{
			name: "минимум INT со знаком минус — в диапазоне",
			src:  "PROGRAM P\nVAR x : INT := -32768; END_VAR\nx := 32767;\nEND_PROGRAM",
		},
		{
			name: "за нижней границей",
			src:  "PROGRAM P\nVAR x : INT; END_VAR\nx := -32769;\nEND_PROGRAM",
			want: []string{`literal -32769 out of range for INT`},
		},
		{
			name: "литерал в границах внутри выражения условия",
			src:  "PROGRAM P\nVAR x : INT; END_VAR\nIF x > 32768 THEN x := 1; END_IF\nEND_PROGRAM",
			want: []string{`literal 32768 out of range for INT`},
		},
		{
			name: "пользовательский тип: диапазон INT не навязывается",
			src:  "PROGRAM P\nVAR m : MyType := 100000; END_VAR\nEND_PROGRAM",
		},
		{
			name: "несколько ошибок за один проход",
			src:  "PROGRAM P\nVAR x : INT; END_VAR\ny := 1;\nx := z;\nEND_PROGRAM",
			want: []string{
				`line 3:1: undeclared variable "y"`,
				`line 4:6: undeclared variable "z"`,
			},
		},
		{
			name: "инициализатор ссылается на переменную из следующего блока",
			src:  "PROGRAM P\nVAR x : INT := y; END_VAR\nVAR y : INT; END_VAR\nEND_PROGRAM",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := check(t, tc.src)
			if len(errs) != len(tc.want) {
				t.Fatalf("ошибок: got %d, want %d\ngot: %v", len(errs), len(tc.want), errs)
			}
			for i, sub := range tc.want {
				if !strings.Contains(errs[i].Error(), sub) {
					t.Errorf("ошибка %d: %q не содержит %q", i, errs[i].Error(), sub)
				}
			}
		})
	}
}
