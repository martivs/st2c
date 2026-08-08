// Golden-тесты парсера: examples/<имя>.st разбирается, String() сравнивается
// с эталоном testdata/<имя>.ast. Корпус .st един и живёт в examples/
// (решение 9 плана 2026-08-07), множество тестов задают сами эталоны:
// glob по testdata/*.ast. Эталоны перегенерируются флагом -update:
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

// parseOneProgram — хелпер: разбирает исходник через ParseSourceFile и
// возвращает единственную PROGRAM из корня.
func parseOneProgram(t *testing.T, src string) *ast.Program {
	t.Helper()
	p := parser.New(lexer.New(src))
	sf, err := p.ParseSourceFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(sf.POUs) != 1 {
		t.Fatalf("POU: got %d, want 1", len(sf.POUs))
	}
	prog, ok := sf.POUs[0].(*ast.Program)
	if !ok {
		t.Fatalf("ожидался *ast.Program, получен %T", sf.POUs[0])
	}
	return prog
}

func TestGolden(t *testing.T) {
	astFiles, err := filepath.Glob(filepath.Join("testdata", "*.ast"))
	if err != nil {
		t.Fatal(err)
	}
	if len(astFiles) == 0 {
		t.Fatal("нет эталонов testdata/*.ast")
	}
	for _, goldenPath := range astFiles {
		name := strings.TrimSuffix(filepath.Base(goldenPath), ".ast")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("..", "..", "examples", name+".st"))
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(src)))
			sf, perr := p.ParseSourceFile()
			if perr != nil {
				t.Fatalf("parse error: %v", perr)
			}
			got := sf.String()

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
			prog := parseOneProgram(t, src)
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

// TestPostfix: этап 3b — постфиксы `.` и `(` в выражениях (MemberExpr,
// CallExpr) и вызов ФБ как оператор (CallStatement). Форма дерева для
// вызовов и членов сверяется по String().
func TestPostfix(t *testing.T) {
	t.Run("член в выражении", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\nx := inst.Out + 1;\nEND_PROGRAM")
		assign := prog.Body[0].(*ast.AssignStatement)
		want := `Binary(+)
  Member(Out)
    Ident(inst)
  Int(1)`
		if got := strings.TrimRight(assign.Value.String(), "\n"); got != want {
			t.Errorf("--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	})
	t.Run("член как цель присваивания", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\ninst.In := 5;\nEND_PROGRAM")
		assign := prog.Body[0].(*ast.AssignStatement)
		if _, ok := assign.Target.(*ast.MemberExpr); !ok {
			t.Fatalf("цель: ожидался *ast.MemberExpr, получен %T", assign.Target)
		}
	})
	t.Run("вызов с позиционными и вложенным вызовом", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\nx := Add(1, Add(2, 3));\nEND_PROGRAM")
		assign := prog.Body[0].(*ast.AssignStatement)
		want := `Call
  Callee:
    Ident(Add)
  Arg
    Int(1)
  Arg
    Call
      Callee:
        Ident(Add)
      Arg
        Int(2)
      Arg
        Int(3)`
		if got := strings.TrimRight(assign.Value.String(), "\n"); got != want {
			t.Errorf("--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	})
	t.Run("унарный минус слабее вызова", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\nx := -F(1);\nEND_PROGRAM")
		assign := prog.Body[0].(*ast.AssignStatement)
		un, ok := assign.Value.(*ast.UnaryExpr)
		if !ok {
			t.Fatalf("ожидался *ast.UnaryExpr, получен %T", assign.Value)
		}
		if _, ok := un.Operand.(*ast.CallExpr); !ok {
			t.Fatalf("операнд: ожидался *ast.CallExpr, получен %T", un.Operand)
		}
	})
	t.Run("вызов ФБ как оператор", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\nfast(step := 10, count => out, 7);\nEND_PROGRAM")
		call, ok := prog.Body[0].(*ast.CallStatement)
		if !ok {
			t.Fatalf("ожидался *ast.CallStatement, получен %T", prog.Body[0])
		}
		want := `CallStmt
  Call
    Callee:
      Ident(fast)
    Arg(step :=)
      Int(10)
    Arg(count =>)
      Ident(out)
    Arg
      Int(7)`
		if got := strings.TrimRight(call.String(), "\n"); got != want {
			t.Errorf("--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	})
	t.Run("вызов без аргументов", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\ninst();\nEND_PROGRAM")
		call, ok := prog.Body[0].(*ast.CallStatement)
		if !ok {
			t.Fatalf("ожидался *ast.CallStatement, получен %T", prog.Body[0])
		}
		if len(call.Call.Args) != 0 {
			t.Errorf("аргументов: got %d, want 0", len(call.Call.Args))
		}
	})
	t.Run("привязка выхода в член", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\nfb(Out => other.In);\nEND_PROGRAM")
		call := prog.Body[0].(*ast.CallStatement)
		arg := call.Call.Args[0]
		if !arg.Output {
			t.Error("ожидался Output-аргумент")
		}
		if _, ok := arg.Value.(*ast.MemberExpr); !ok {
			t.Errorf("цель =>: ожидался *ast.MemberExpr, получен %T", arg.Value)
		}
	})
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
			prog := parseOneProgram(t, src)
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

// TestVarBlocks: фаза 4 — виды VAR-блоков, списки имён, инициализаторы,
// пользовательские имена типов, квалификаторы CONSTANT/RETAIN.
func TestVarBlocks(t *testing.T) {
	tests := []struct {
		name string
		vars string // блоки объявлений между PROGRAM P и телом
		want []string // String() каждого блока, без завершающего \n
	}{
		{
			name: "несколько блоков разных видов",
			vars: "VAR x : INT; END_VAR\nVAR_INPUT a : INT; END_VAR\nVAR_OUTPUT b : INT; END_VAR\nVAR_IN_OUT c : INT; END_VAR\nVAR_TEMP t : INT; END_VAR",
			want: []string{
				"VarBlock(VAR)\n  VarDecl(x : INT)",
				"VarBlock(VAR_INPUT)\n  VarDecl(a : INT)",
				"VarBlock(VAR_OUTPUT)\n  VarDecl(b : INT)",
				"VarBlock(VAR_IN_OUT)\n  VarDecl(c : INT)",
				"VarBlock(VAR_TEMP)\n  VarDecl(t : INT)",
			},
		},
		{
			name: "список имён в одном объявлении",
			vars: "VAR a, b, c : INT; END_VAR",
			want: []string{"VarBlock(VAR)\n  VarDecl(a, b, c : INT)"},
		},
		{
			name: "инициализатор",
			vars: "VAR x : INT := 5; y : INT := -1 + 2; END_VAR",
			want: []string{`VarBlock(VAR)
  VarDecl(x : INT)
    Init:
      Int(5)
  VarDecl(y : INT)
    Init:
      Binary(+)
        Unary(-)
          Int(1)
        Int(2)`},
		},
		{
			name: "пользовательское имя типа",
			vars: "VAR m : MyType; END_VAR",
			want: []string{"VarBlock(VAR)\n  VarDecl(m : MyType)"},
		},
		{
			name: "квалификаторы CONSTANT и RETAIN",
			vars: "VAR CONSTANT k : INT := 7; END_VAR\nVAR RETAIN r : INT; END_VAR",
			want: []string{
				"VarBlock(VAR CONSTANT)\n  VarDecl(k : INT)\n    Init:\n      Int(7)",
				"VarBlock(VAR RETAIN)\n  VarDecl(r : INT)",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "PROGRAM P\n" + tc.vars + "\nEND_PROGRAM"
			prog := parseOneProgram(t, src)
			if len(prog.VarBlocks) != len(tc.want) {
				t.Fatalf("блоков: got %d, want %d", len(prog.VarBlocks), len(tc.want))
			}
			for i, want := range tc.want {
				got := strings.TrimRight(prog.VarBlocks[i].String(), "\n")
				if got != want {
					t.Errorf("блок %d:\n--- got ---\n%s\n--- want ---\n%s", i, got, want)
				}
			}
		})
	}
}

// TestSourceFile: фаза 5 — корень дерева SourceFile, цикл по POU до EOF.
func TestSourceFile(t *testing.T) {
	t.Run("один PROGRAM", func(t *testing.T) {
		prog := parseOneProgram(t, "PROGRAM P\nEND_PROGRAM")
		if prog.Name != "P" {
			t.Errorf("имя программы: got %q, want %q", prog.Name, "P")
		}
	})
	t.Run("два PROGRAM в одном файле", func(t *testing.T) {
		src := "PROGRAM A\nEND_PROGRAM\nPROGRAM B\nEND_PROGRAM"
		p := parser.New(lexer.New(src))
		sf, err := p.ParseSourceFile()
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if len(sf.POUs) != 2 {
			t.Fatalf("POU: got %d, want 2", len(sf.POUs))
		}
		for i, wantName := range []string{"A", "B"} {
			prog, ok := sf.POUs[i].(*ast.Program)
			if !ok {
				t.Fatalf("POU %d: ожидался *ast.Program, получен %T", i, sf.POUs[i])
			}
			if prog.Name != wantName {
				t.Errorf("POU %d: имя %q, want %q", i, prog.Name, wantName)
			}
		}
	})
	t.Run("SourceFile.String — конкатенация POU", func(t *testing.T) {
		src := "PROGRAM A\nEND_PROGRAM\nPROGRAM B\nEND_PROGRAM"
		p := parser.New(lexer.New(src))
		sf, err := p.ParseSourceFile()
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if got, want := sf.String(), "Program(A)\nProgram(B)\n"; got != want {
			t.Errorf("String(): got %q, want %q", got, want)
		}
	})
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
			name:       "литерал вместо имени типа в объявлении",
			input:      "PROGRAM P\nVAR\nx : 5;\nEND_VAR\nEND_PROGRAM",
			wantSubstr: `line 3: expected type name, got INT_LIT "5"`,
		},
		{
			name:       "пропущена запятая или двоеточие в списке имён",
			input:      "PROGRAM P\nVAR\na b : INT;\nEND_VAR\nEND_PROGRAM",
			wantSubstr: `line 3: expected :`,
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
			name:       "мусор на верхнем уровне после END_PROGRAM",
			input:      "PROGRAM P\nEND_PROGRAM\nx := 1;\n",
			wantSubstr: `line 3: expected PROGRAM, FUNCTION or FUNCTION_BLOCK at top level, got IDENT "x"`,
		},
		{
			name:       "BY без выражения шага",
			input:      "PROGRAM P\nFOR i := 1 TO 5 BY DO\ni := 1;\nEND_FOR\nEND_PROGRAM",
			wantSubstr: `line 2: expected expression, got DO`,
		},
		{
			name:       "вызов без закрывающей скобки",
			input:      "PROGRAM P\nx := Add(1, 2;\nEND_PROGRAM",
			wantSubstr: `line 2: expected )`,
		},
		{
			name:       "=> с не-lvalue",
			input:      "PROGRAM P\ninst(Out => 5);\nEND_PROGRAM",
			wantSubstr: `line 2: output binding Out => requires a variable`,
		},
		{
			name:       "END_FUNCTION вместо END_FUNCTION_BLOCK",
			input:      "FUNCTION_BLOCK FB\nx := 1;\nEND_FUNCTION",
			wantSubstr: `line 3: expected END_FUNCTION_BLOCK, got END_FUNCTION`,
		},
		{
			name:       "вызов-оператор без точки с запятой",
			input:      "PROGRAM P\ninst(step := 1)\nEND_PROGRAM",
			wantSubstr: `line 3: expected ;`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.New(lexer.New(tc.input))
			_, err := p.ParseSourceFile()
			if err == nil {
				t.Fatalf("ожидалась ошибка с подстрокой %q, разбор прошёл без ошибок", tc.wantSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("ошибка %q не содержит %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}
