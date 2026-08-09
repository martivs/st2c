// Тесты sema: вход-программа → ожидаемые подстроки ошибок (с позициями)
// либо их отсутствие.
package sema_test

import (
	"os"
	"path/filepath"
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
			// С этапа 6 пользовательские типы резолвятся в объявленные ФБ;
			// нерезолвившийся тип — ошибка (прежний молчаливый пропуск ушёл).
			// Диапазон INT к неизвестному типу по-прежнему не навязывается —
			// на литерал 100000 второй ошибки нет.
			name: "неизвестный пользовательский тип — ошибка (этап 6)",
			src:  "PROGRAM P\nVAR m : MyType := 100000; END_VAR\nEND_PROGRAM",
			want: []string{`line 2:5: unknown type "MyType"`},
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

// TestFunctions — этап 4: глобальная таблица POU, области видимости функций,
// проверки вызовов, запрет рекурсии. ФБ-специфика (экземпляры, члены,
// вызов-оператор) — этап 6, TestFunctionBlocks.
func TestFunctions(t *testing.T) {
	// Общая пара функций для кейсов вызова.
	const addSrc = `FUNCTION Add : INT
VAR_INPUT x : INT; y : INT; END_VAR
Add := x + y;
END_FUNCTION
`
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "корректная программа с функциями: позиционные, именованные, вложенные, возврат через имя",
			src: addSrc + `PROGRAM P
VAR a : INT; END_VAR
a := Add(2, 3);
a := Add(Add(1, 2), y := 4);
a := add(x := 1, y := 2);
END_PROGRAM`,
		},
		{
			name: "дубликат имени POU в разном регистре",
			src:  "FUNCTION F : INT\nEND_FUNCTION\nFUNCTION f : INT\nEND_FUNCTION",
			want: []string{`line 3:1: duplicate POU "f" (first declared as "F" at line 1)`},
		},
		{
			name: "дубликат имени между видами POU",
			src:  "FUNCTION_BLOCK X\nEND_FUNCTION_BLOCK\nPROGRAM X\nEND_PROGRAM",
			want: []string{`line 3:1: duplicate POU "X" (first declared as "X" at line 1)`},
		},
		{
			name: "вызов необъявленной функции",
			src:  "PROGRAM P\nVAR a : INT; END_VAR\na := Foo(1);\nEND_PROGRAM",
			want: []string{`line 3:6: undeclared function "Foo"`},
		},
		{
			name: "вызов не-функции (имя PROGRAM)",
			src:  "PROGRAM P\nVAR a : INT; END_VAR\na := P(1);\nEND_PROGRAM",
			want: []string{`line 3:6: "P" is not a function`},
		},
		{
			name: "вызов не-функции (имя FUNCTION_BLOCK)",
			src: `FUNCTION_BLOCK Counter
VAR_INPUT step : INT; END_VAR
END_FUNCTION_BLOCK
PROGRAM P
VAR a : INT; END_VAR
a := Counter(1);
END_PROGRAM`,
			want: []string{`line 6:6: "Counter" is not a function`},
		},
		{
			name: "неверное число аргументов: меньше и больше",
			src: addSrc + `PROGRAM P
VAR a : INT; END_VAR
a := Add(1);
a := Add(1, 2, 3);
END_PROGRAM`,
			want: []string{
				`line 7:9: function "Add" expects 2 argument(s), got 1`,
				`line 8:9: function "Add" expects 2 argument(s), got 3`,
			},
		},
		{
			name: "именованный аргумент с несуществующим входом",
			src:  addSrc + "PROGRAM P\nVAR a : INT; END_VAR\na := Add(x := 1, z := 2);\nEND_PROGRAM",
			want: []string{`line 7:9: function "Add" has no input "z"`},
		},
		{
			name: "повторная привязка входа: именованный поверх позиционного",
			src:  addSrc + "PROGRAM P\nVAR a : INT; END_VAR\na := Add(1, x := 2);\nEND_PROGRAM",
			want: []string{`line 7:9: input "x" of function "Add" bound more than once`},
		},
		{
			name: "позиционный аргумент после именованного",
			src:  addSrc + "PROGRAM P\nVAR a : INT; END_VAR\na := Add(x := 1, 2);\nEND_PROGRAM",
			want: []string{`line 7:9: positional argument after named argument in call of "Add"`},
		},
		{
			name: "привязка выхода => к функции неприменима",
			src:  addSrc + "PROGRAM P\nVAR a : INT; END_VAR\na := Add(1, y => a);\nEND_PROGRAM",
			want: []string{`line 7:9: output binding "y" => is not applicable to function "Add"`},
		},
		{
			name: "литерал вне диапазона INT в аргументе (целевой тип — тип входа)",
			src:  addSrc + "PROGRAM P\nVAR a : INT; END_VAR\na := Add(70000, y := 1);\nEND_PROGRAM",
			want: []string{`line 7:10: literal 70000 out of range for INT`},
		},
		{
			name: "необъявленная переменная в теле функции",
			src:  "FUNCTION F : INT\nF := q;\nEND_FUNCTION",
			want: []string{`line 2:6: undeclared variable "q"`},
		},
		{
			name: "параметр с именем функции — дубликат переменной возврата",
			src:  "FUNCTION F : INT\nVAR_INPUT F : INT; END_VAR\nEND_FUNCTION",
			want: []string{`line 2:11: duplicate declaration of "F" (first declared as "F" at line 1)`},
		},
		{
			name: "прямая рекурсия",
			src:  "FUNCTION F : INT\nVAR_INPUT x : INT; END_VAR\nF := F(x);\nEND_FUNCTION",
			want: []string{`line 3:6: recursive call of function "F" (recursion is not allowed)`},
		},
		{
			name: "взаимная рекурсия",
			src: `FUNCTION F : INT
VAR_INPUT x : INT; END_VAR
F := G(x);
END_FUNCTION
FUNCTION G : INT
VAR_INPUT x : INT; END_VAR
G := F(x);
END_FUNCTION`,
			want: []string{`line 7:6: recursive call of function "F" (recursion is not allowed)`},
		},
		{
			name: "цепочка вызовов без цикла — не рекурсия",
			src: `FUNCTION G : INT
VAR_INPUT x : INT; END_VAR
G := x * 2;
END_FUNCTION
FUNCTION F : INT
VAR_INPUT x : INT; END_VAR
F := G(x) + G(1);
END_FUNCTION`,
		},
		{
			name: "тело ФБ проверяется: необъявленная переменная",
			src:  "FUNCTION_BLOCK B\nVAR x : INT; END_VAR\nx := y;\nEND_FUNCTION_BLOCK",
			want: []string{`line 3:6: undeclared variable "y"`},
		},
		{
			// Экземпляры ФБ, вызов-оператор и члены с этапа 6 проверяются;
			// эта программа корректна по всем правилам (запись во вход,
			// чтение выхода, привязка `=>`) — ложных ошибок быть не должно.
			name: "экземпляры ФБ, вызов-оператор и члены: корректная программа чиста",
			src: `FUNCTION_BLOCK Counter
VAR_INPUT step : INT; END_VAR
VAR_OUTPUT count : INT; END_VAR
count := count + step;
END_FUNCTION_BLOCK
PROGRAM P
VAR c : Counter; total : INT; END_VAR
c(step := 1, count => total);
c.step := 2;
total := total + c.count;
END_PROGRAM`,
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

// TestFunctionBlocks — этап 6: резолв пользовательских типов в объявленные
// ФБ, доступ к членам (вход — только запись, выход — только чтение), вызов
// ФБ как оператор (только именованные аргументы; список необязателен и может
// быть неполным), запрет циклической вложенности экземпляров.
func TestFunctionBlocks(t *testing.T) {
	// Общий ФБ для кейсов: вход step, выход count, внутренняя calls.
	// Занимает строки 1–6, программа за ним начинается со строки 7.
	const fbSrc = `FUNCTION_BLOCK Counter
VAR_INPUT step : INT; END_VAR
VAR_OUTPUT count : INT; END_VAR
VAR calls : INT; END_VAR
count := count + step;
END_FUNCTION_BLOCK
`
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			// Аргументы ФБ необязательны: slow(step := 1) без привязки выхода
			// и slow() совсем без аргументов законны — непереданный вход
			// хранит значение с прошлого вызова, в этом смысл состояния.
			name: "корректная программа: частичные аргументы, члены, два экземпляра",
			src: fbSrc + `PROGRAM P
VAR fast : Counter; slow : Counter; out : INT; total : INT; END_VAR
fast(step := 10, count => out);
slow(step := 1);
slow();
fast.step := 2;
total := out + slow.count;
END_PROGRAM`,
		},
		{
			name: "привязка выхода в член-вход другого экземпляра — законна",
			src: fbSrc + `PROGRAM P
VAR a : Counter; b : Counter; END_VAR
a(count => b.step);
END_PROGRAM`,
		},
		{
			name: "неизвестный тип объявления",
			src:  "PROGRAM P\nVAR c : Widget; END_VAR\nEND_PROGRAM",
			want: []string{`line 2:5: unknown type "Widget"`},
		},
		{
			name: "имя PROGRAM в позиции типа",
			src:  "PROGRAM Q\nEND_PROGRAM\nPROGRAM P\nVAR c : Q; END_VAR\nEND_PROGRAM",
			want: []string{`line 4:5: "Q" is not a type`},
		},
		{
			name: "имя FUNCTION в позиции типа",
			src:  "FUNCTION F : INT\nF := 0;\nEND_FUNCTION\nPROGRAM P\nVAR c : F; END_VAR\nEND_PROGRAM",
			want: []string{`line 5:5: "F" is not a type`},
		},
		{
			name: "инициализатор у экземпляра ФБ",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter := 5; END_VAR\nEND_PROGRAM",
			want: []string{`line 8:5: function block instance cannot have an initializer`},
		},
		{
			name: "член у не-экземпляра",
			src:  "PROGRAM P\nVAR x : INT; y : INT; END_VAR\ny := x.foo;\nEND_PROGRAM",
			want: []string{`line 3:7: "x" is not a function block instance`},
		},
		{
			name: "несуществующий член",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; x : INT; END_VAR\nx := c.missing;\nEND_PROGRAM",
			want: []string{`line 9:7: function block "Counter" has no input or output "missing"`},
		},
		{
			name: "внутренняя VAR снаружи недоступна",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; x : INT; END_VAR\nx := c.calls;\nEND_PROGRAM",
			want: []string{`line 9:7: function block "Counter" has no input or output "calls"`},
		},
		{
			name: "чтение входа извне",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; x : INT; END_VAR\nx := c.step;\nEND_PROGRAM",
			want: []string{`line 9:7: cannot read input "step" of instance "c"`},
		},
		{
			name: "запись в выход извне",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc.count := 1;\nEND_PROGRAM",
			want: []string{`line 9:2: cannot assign to output "count" of instance "c"`},
		},
		{
			name: "диапазон литерала через тип члена",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc.step := 40000;\nEND_PROGRAM",
			want: []string{`line 9:11: literal 40000 out of range for INT`},
		},
		{
			name: "вызов-оператор у скалярной переменной",
			src:  "PROGRAM P\nVAR x : INT; END_VAR\nx();\nEND_PROGRAM",
			want: []string{`line 3:1: "x" is not a function block instance`},
		},
		{
			name: "вызов-оператор у имени ФБ-типа (без экземпляра)",
			src:  fbSrc + "PROGRAM P\nCounter(step := 1);\nEND_PROGRAM",
			want: []string{`line 8:1: "Counter" is not a function block instance`},
		},
		{
			name: "вызов-оператор необъявленного имени",
			src:  "PROGRAM P\nfoo();\nEND_PROGRAM",
			want: []string{`line 2:1: "foo" is not a function block instance`},
		},
		{
			name: "позиционный аргумент в вызове ФБ",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc(1);\nEND_PROGRAM",
			want: []string{`line 9:2: function block call arguments must be named`},
		},
		{
			name: "привязка := к выходу",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc(count := 1);\nEND_PROGRAM",
			want: []string{`line 9:2: "count" is an output of "Counter": bind it with =>, not :=`},
		},
		{
			name: "привязка => ко входу",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; x : INT; END_VAR\nc(step => x);\nEND_PROGRAM",
			want: []string{`line 9:2: "step" is an input of "Counter": pass it with :=, not =>`},
		},
		{
			name: "несуществующее имя аргумента",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc(bogus := 1);\nEND_PROGRAM",
			want: []string{`line 9:2: function block "Counter" has no input or output "bogus"`},
		},
		{
			name: "повторная привязка входа",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc(step := 1, step := 2);\nEND_PROGRAM",
			want: []string{`line 9:2: "step" bound more than once in call of instance "c"`},
		},
		{
			name: "экземпляр как значение в выражении",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; x : INT; END_VAR\nx := c;\nEND_PROGRAM",
			want: []string{`line 9:6: function block instance "c" cannot be used as a value`},
		},
		{
			name: "присваивание экземпляру целиком",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; END_VAR\nc := 1;\nEND_PROGRAM",
			want: []string{`line 9:1: cannot assign to function block instance "c"`},
		},
		{
			name: "вызов экземпляра в выражении",
			src:  fbSrc + "PROGRAM P\nVAR c : Counter; x : INT; END_VAR\nx := c(step := 1);\nEND_PROGRAM",
			want: []string{`line 9:6: function block instance "c" cannot be called in an expression`},
		},
		{
			name: "прямая циклическая вложенность",
			src:  "FUNCTION_BLOCK C\nVAR me : C; END_VAR\nEND_FUNCTION_BLOCK",
			want: []string{`line 2:5: instance "me" creates cyclic nesting of function blocks`},
		},
		{
			// Одна ошибка на цикл — на объявлении, замыкающем его при DFS
			// в порядке файла.
			name: "взаимная циклическая вложенность",
			src: `FUNCTION_BLOCK A
VAR b : B; END_VAR
END_FUNCTION_BLOCK
FUNCTION_BLOCK B
VAR a : A; END_VAR
END_FUNCTION_BLOCK`,
			want: []string{`line 5:5: instance "a" creates cyclic nesting of function blocks`},
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

// TestExamplesClean — весь корпус examples/ проходит sema без ошибок:
// регрессия на ложные срабатывания (особенно на ФБ-примерах: резолв типов,
// члены и вызов-оператор с этапа 6 проверяются по-настоящему).
func TestExamplesClean(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.st"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("examples/*.st не найдены")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if errs := check(t, string(src)); len(errs) > 0 {
				t.Errorf("неожиданные ошибки sema: %v", errs)
			}
		})
	}
}
