// Тесты codegen в два слоя (решение 7 плана):
//
//  1. TestGolden — examples/<name>.st прогоняется через parse → sema →
//     Generate, результат сверяется с эталоном testdata/<name>.c. Эталоны
//     перегенерируются флагом -update:
//
//     go test ./src/codegen -update
//
//     Diff перегенерированных эталонов просматривается вручную перед коммитом.
//
//  2. TestGCC — эталонный testdata/<name>.c собирается gcc, бинарь
//     запускается (с таймаутом: сломанный FOR не выдаёт неверное число, а
//     зависает), stdout сверяется с testdata/<name>.expected. Файлы .expected
//     написаны руками: ключевые значения выписаны в комментариях самих .st.
//     Без gcc в системе тест скипается, go test ./... остаётся зелёным.
//
// Корпус .st един и живёт в examples/ (решение 9); в testdata/ — только
// эталоны. Список goldenExamples пополняется по мере этапов плана.
package codegen_test

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"st2c/src/codegen"
	"st2c/src/lexer"
	"st2c/src/parser"
	"st2c/src/sema"
)

var update = flag.Bool("update", false, "перегенерировать golden-эталоны testdata/*.c")

// goldenExamples — имена файлов examples/*.st, для которых есть эталоны.
// Этап 1: объявления, присваивания, выражения, IF, FOR. Этап 2: for_edge
// (регрессия на границы INT — без широкого счётчика зависает, ловится
// таймаутом) и четыре старых примера. Этап 5: func_simple (FUNCTION, вызовы,
// раскладка именованных аргументов). Этап 7: fb_counter (два независимых
// экземпляра — на одном «состояние утекло в глобальную» не видна) и
// fb_nested (рекурсивный _init, порядок typedef).
var goldenExamples = []string{
	"vars_all",
	"expr_all",
	"name_clash",
	"example",
	"nested_if_in_for",
	"nested_for_in_if",
	"deeply_nested",
	"for_edge",
	"func_simple",
	"fb_counter",
	"fb_nested",
}

// exampleScans — сколько сканов зовёт драйвер эталона (по умолчанию 1).
// Смысл ФБ — состояние, переживающее скан, — виден только за несколько
// сканов; значения по сканам выписаны в комментариях самих .st.
var exampleScans = map[string]int{
	"fb_counter": 3,
	"fb_nested":  3,
}

// runTimeout — предел на запуск собранного бинаря: режим отказа сломанного
// цикла — зависание.
const runTimeout = 10 * time.Second

// generateExample — общий конвейер тестов: examples/<name>.st → C-текст
// в режиме самодостаточного файла (-main; число сканов — из exampleScans).
func generateExample(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "examples", name+".st"))
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(src)))
	sf, err := p.ParseSourceFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if errs := sema.Check(sf); len(errs) > 0 {
		t.Fatalf("sema errors: %v", errs)
	}
	scans := exampleScans[name]
	if scans == 0 {
		scans = 1
	}
	code, err := codegen.Generate(sf, codegen.Options{Main: true, Scans: scans})
	if err != nil {
		t.Fatalf("codegen error: %v", err)
	}
	return code
}

func TestGolden(t *testing.T) {
	for _, name := range goldenExamples {
		t.Run(name, func(t *testing.T) {
			got := generateExample(t, name)
			goldenPath := filepath.Join("testdata", name+".c")
			if *update {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("нет эталона %s (сгенерировать: go test ./src/codegen -update): %v", goldenPath, err)
			}
			if got != string(want) {
				t.Errorf("C не совпал с эталоном %s\n--- got ---\n%s--- want ---\n%s", goldenPath, got, want)
			}
		})
	}
}

func TestGCC(t *testing.T) {
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc не найден в PATH — сквозной тест пропущен")
	}
	for _, name := range goldenExamples {
		t.Run(name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), name)
			cmd := exec.Command(gcc, "-std=c99", "-Wall", "-Wextra", "-o", bin,
				filepath.Join("testdata", name+".c"))
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("gcc: %v\n%s", err, out)
			} else if len(out) > 0 {
				// Без -Werror (решение 7): предупреждения видны, но тест не валят.
				t.Logf("gcc warnings:\n%s", out)
			}

			ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
			defer cancel()
			got, err := exec.CommandContext(ctx, bin).Output()
			if ctx.Err() != nil {
				t.Fatalf("бинарь не завершился за %v — вероятно, бесконечный цикл", runTimeout)
			}
			if err != nil {
				t.Fatalf("запуск: %v", err)
			}
			want, err := os.ReadFile(filepath.Join("testdata", name+".expected"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Errorf("вывод не совпал с %s.expected\n--- got ---\n%s--- want ---\n%s", name, got, want)
			}
		})
	}
}

// TestNameErrors: отказы маппера имён (решение 8) — то, что фронтенд
// пропускает молча, codegen обязан отвергнуть внятной ошибкой с позицией.
func TestNameErrors(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantSubstr string
		skipSema   bool // кейсы, которые sema этапа 6 отвергла бы раньше codegen
	}{
		{
			name:       "не-ASCII имя переменной",
			src:        "PROGRAM P\nVAR\nпеременная : INT;\nEND_VAR\nEND_PROGRAM",
			wantSubstr: "line 3:1: codegen: identifier \"переменная\" is not representable in C",
		},
		{
			name:       "склейка после префиксации",
			src:        "PROGRAM P\nVAR\nst_switch : INT;\nswitch : INT;\nEND_VAR\nEND_PROGRAM",
			wantSubstr: `line 4:1: codegen: renamed "switch" collides with "st_switch"`,
		},
		{
			// С этапа 6 неизвестный тип ловит sema; проверка в cType остаётся
			// внутренней защитой (Generate можно позвать и без sema) — тест
			// зовёт генератор напрямую, минуя sema.
			name:       "неизвестный тип",
			src:        "PROGRAM P\nVAR\nm : MyType;\nEND_VAR\nEND_PROGRAM",
			wantSubstr: `line 3:1: codegen: no C mapping for type "MyType"`,
			skipSema:   true,
		},
		{
			name:       "не-ASCII имя параметра функции",
			src:        "FUNCTION F : INT\nVAR_INPUT\nпар : INT;\nEND_VAR\nF := 0;\nEND_FUNCTION\nPROGRAM P\nEND_PROGRAM",
			wantSubstr: "line 3:1: codegen: identifier \"пар\" is not representable in C",
		},
		{
			name:       "VAR_OUTPUT в FUNCTION",
			src:        "FUNCTION F : INT\nVAR_OUTPUT\no : INT;\nEND_VAR\nF := 0;\nEND_FUNCTION\nPROGRAM P\nEND_PROGRAM",
			wantSubstr: "line 2:1: codegen: VAR_OUTPUT block is not supported in FUNCTION",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.New(lexer.New(tc.src))
			sf, err := p.ParseSourceFile()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !tc.skipSema {
				if errs := sema.Check(sf); len(errs) > 0 {
					t.Fatalf("sema errors: %v", errs)
				}
			}
			_, err = codegen.Generate(sf, codegen.Options{})
			if err == nil {
				t.Fatalf("ожидалась ошибка с подстрокой %q, генерация прошла", tc.wantSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("ошибка %q не содержит %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}
