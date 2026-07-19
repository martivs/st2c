// Golden-тесты парсера: testdata/*.st разбирается, prog.String() сравнивается
// с эталоном testdata/*.ast. Эталоны перегенерируются флагом -update:
//
//	go test ./src/parser -update
//
// Diff перегенерированных эталонов просматривается вручную перед коммитом.
package parser_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"st2c/src/ast"
	"st2c/src/lexer"
	"st2c/src/parser"
)

var update = flag.Bool("update", false, "перегенерировать golden-эталоны testdata/*.ast")

func TestGolden(t *testing.T) {
	stFiles, err := filepath.Glob(filepath.Join("testdata", "*.st"))
	if err != nil {
		t.Fatal(err)
	}
	if len(stFiles) == 0 {
		t.Fatal("нет файлов testdata/*.st")
	}
	for _, stPath := range stFiles {
		name := strings.TrimSuffix(filepath.Base(stPath), ".st")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(stPath)
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(src)))
			prog, perr := p.ParseProgram()
			if perr != nil {
				t.Fatalf("parse error: %v", perr)
			}
			got := prog.String()

			goldenPath := filepath.Join("testdata", name+".ast")
			if *update {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("нет эталона %s (сгенерировать: go test ./src/parser -update): %v", goldenPath, err)
			}
			if got != string(want) {
				t.Errorf("AST не совпал с эталоном %s\n--- got ---\n%s--- want ---\n%s", goldenPath, got, want)
			}
		})
	}
}

// TestExpressionStructure: приоритеты и ассоциативность Pratt-парсера.
// Выражение подставляется в присваивание, сравнивается String() значения.
func TestExpressionStructure(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string // String() значения присваивания, без завершающего \n
	}{
		{
			name: "левая ассоциативность вычитания",
			expr: "a - b - c",
			want: `Binary(-)
  Binary(-)
    Ident(a)
    Ident(b)
  Ident(c)`,
		},
		{
			name: "умножение сильнее сложения",
			expr: "a + b * c",
			want: `Binary(+)
  Ident(a)
  Binary(*)
    Ident(b)
    Ident(c)`,
		},
		{
			name: "унарный минус",
			expr: "-x + 1",
			want: `Binary(+)
  Unary(-)
    Ident(x)
  Int(1)`,
		},
		{
			name: "двойной унарный минус",
			expr: "--x",
			want: `Unary(-)
  Unary(-)
    Ident(x)`,
		},
		{
			name: "нестрогое сравнение",
			expr: "a <= b",
			want: `Binary(<=)
  Ident(a)
  Ident(b)`,
		},
		{
			name: "неравенство",
			expr: "a <> b",
			want: `Binary(<>)
  Ident(a)
  Ident(b)`,
		},
		{
			name: "скобки перебивают приоритет",
			expr: "(i + j) > 4",
			want: `Binary(>)
  Binary(+)
    Ident(i)
    Ident(j)
  Int(4)`,
		},
		{
			// P8 ревью: цепочка сравнений разбирается левоассоциативно без
			// предупреждения — семантическая проверка BOOL/INT придёт с sema.
			name: "цепочка сравнений левоассоциативна",
			expr: "a < b < c",
			want: `Binary(<)
  Binary(<)
    Ident(a)
    Ident(b)
  Ident(c)`,
		},
		{
			name: "сравнение = слабее отношения <",
			expr: "a = b < c",
			want: `Binary(=)
  Ident(a)
  Binary(<)
    Ident(b)
    Ident(c)`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "PROGRAM P\nres := " + tc.expr + ";\nEND_PROGRAM"
			p := parser.New(lexer.New(src))
			prog, err := p.ParseProgram()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			assign, ok := prog.Body[0].(*ast.AssignStatement)
			if !ok {
				t.Fatalf("ожидался AssignStatement, получен %T", prog.Body[0])
			}
			got := strings.TrimRight(assign.Value.String(), "\n")
			if got != tc.want {
				t.Errorf("дерево выражения %q:\n--- got ---\n%s\n--- want ---\n%s", tc.expr, got, tc.want)
			}
		})
	}
}

// TestForBy: необязательный шаг `BY` в FOR — узел Step в дереве; без BY
// секции Step нет (nil → шаг 1).
func TestForBy(t *testing.T) {
	tests := []struct {
		name string
		stmt string
		want string // String() оператора FOR, без завершающего \n
	}{
		{
			name: "с шагом BY",
			stmt: "FOR i := 1 TO 10 BY 2 DO sum := sum + i; END_FOR",
			want: `For(i)
  Start:
    Int(1)
  End:
    Int(10)
  Step:
    Int(2)
  Do:
    Assign
      Target:
        Ident(sum)
      Value:
        Binary(+)
          Ident(sum)
          Ident(i)`,
		},
		{
			name: "шаг-выражение со знаком",
			stmt: "FOR i := 10 TO 1 BY -1 DO sum := i; END_FOR",
			want: `For(i)
  Start:
    Int(10)
  End:
    Int(1)
  Step:
    Unary(-)
      Int(1)
  Do:
    Assign
      Target:
        Ident(sum)
      Value:
        Ident(i)`,
		},
		{
			name: "без BY секции Step нет",
			stmt: "FOR i := 1 TO 3 DO sum := i; END_FOR",
			want: `For(i)
  Start:
    Int(1)
  End:
    Int(3)
  Do:
    Assign
      Target:
        Ident(sum)
      Value:
        Ident(i)`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "PROGRAM P\n" + tc.stmt + "\nEND_PROGRAM"
			p := parser.New(lexer.New(src))
			prog, err := p.ParseProgram()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			forStmt, ok := prog.Body[0].(*ast.ForStatement)
			if !ok {
				t.Fatalf("ожидался ForStatement, получен %T", prog.Body[0])
			}
			got := strings.TrimRight(forStmt.String(), "\n")
			if got != tc.want {
				t.Errorf("дерево FOR:\n--- got ---\n%s\n--- want ---\n%s", got, tc.want)
			}
		})
	}
}

// TestParseErrors: вход → подстрока ожидаемого сообщения об ошибке
// (включая номер строки).
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSubstr string
	}{
		{
			name:       "равно вместо присваивания",
			input:      "PROGRAM P\nx = 1;\nEND_PROGRAM",
			wantSubstr: `line 2: expected :=`,
		},
		{
			name:       "литерал в начале оператора",
			input:      "PROGRAM P\n123;\nEND_PROGRAM",
			wantSubstr: `line 2: unexpected token INT_LIT "123" at statement start`,
		},
		{
			name:       "пропущен THEN",
			input:      "PROGRAM P\nIF x > 1 x := 2; END_IF\nEND_PROGRAM",
			wantSubstr: `line 2: expected THEN`,
		},
		{
			name:       "пропущена точка с запятой",
			input:      "PROGRAM P\nx := 1\ny := 2;\nEND_PROGRAM",
			wantSubstr: `line 3: expected ;`,
		},
		{
			name:       "нет END_PROGRAM",
			input:      "PROGRAM P\nx := 1;\n",
			wantSubstr: `expected END_PROGRAM`,
		},
		{
			name:       "не-INT тип в объявлении",
			input:      "PROGRAM P\nVAR\nx : REAL;\nEND_VAR\nEND_PROGRAM",
			wantSubstr: `line 3: expected INT`,
		},
		{
			name:       "нет выражения после оператора",
			input:      "PROGRAM P\nx := ;\nEND_PROGRAM",
			wantSubstr: `line 2: expected expression`,
		},
		{
			name:       "незакрытая скобка",
			input:      "PROGRAM P\nx := (1 + 2;\nEND_PROGRAM",
			wantSubstr: `line 2: expected )`,
		},
		{
			name:       "литерал слева от присваивания",
			input:      "PROGRAM P\n5 := x;\nEND_PROGRAM",
			wantSubstr: `line 2: unexpected token INT_LIT "5" at statement start`,
		},
		{
			name:       "оператор вместо := после цели присваивания",
			input:      "PROGRAM P\nx + 1;\nEND_PROGRAM",
			wantSubstr: `line 2: expected :=, got + "+"`,
		},
		{
			name:       "BY без выражения шага",
			input:      "PROGRAM P\nFOR i := 1 TO 5 BY DO\ni := 1;\nEND_FOR\nEND_PROGRAM",
			wantSubstr: `line 2: expected expression, got DO`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.New(lexer.New(tc.input))
			_, err := p.ParseProgram()
			if err == nil {
				t.Fatalf("ожидалась ошибка с подстрокой %q, разбор прошёл без ошибок", tc.wantSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("ошибка %q не содержит %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}
