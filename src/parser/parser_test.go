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
