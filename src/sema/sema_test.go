// Тесты sema: вход-программа → ожидаемые подстроки ошибок (с позициями)
// либо их отсутствие.
package sema_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"st2c/src/ast"
	"st2c/src/lexer"
	"st2c/src/parser"
	"st2c/src/sema"
)

// check разбирает исходник (разбор обязан пройти) и возвращает ошибки sema.
func check(t *testing.T, src string) []error {
	t.Helper()
	_, _, errs := checkInfo(t, src)
	return errs
}

// checkInfo — то же, плюс дерево и side-table типов для проверок Info.
func checkInfo(t *testing.T, src string) (*ast.SourceFile, *sema.Info, []error) {
	t.Helper()
	p := parser.New(lexer.New(src))
	sf, err := p.ParseSourceFile()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	info, errs := sema.Check(sf)
	return sf, info, errs
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

// TestTypes — этап 2 плана REAL: слой типов. Вывод типов снизу вверх,
// запрет неявного смешения INT и REAL, адаптивный целый литерал, BOOL
// только в условии IF, FOR только по INT, встроенные конверсии, резолв
// типа возврата функции, типы аргументов функций и входов/выходов ФБ.
func TestTypes(t *testing.T) {
	// Функция с REAL-параметром и REAL-возвратом; занимает строки 1–4.
	const halfSrc = `FUNCTION Half : REAL
VAR_INPUT x : REAL; END_VAR
Half := x / 2;
END_FUNCTION
`
	// ФБ с REAL-входом и REAL-выходом; занимает строки 1–5.
	const avgSrc = `FUNCTION_BLOCK Avg
VAR_INPUT v : REAL; END_VAR
VAR_OUTPUT m : REAL; END_VAR
m := (m + v) / 2;
END_FUNCTION_BLOCK
`
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			// Позитив: адаптивные литералы (`:= 0`, `1 / 2`, `r + 1`, `-1`),
			// сравнения в обе стороны, конверсии, целочисленный FOR с
			// накоплением в REAL.
			name: "корректная программа с REAL",
			src: `PROGRAM P
VAR r : REAL := 0; s : REAL := 2.5; i : INT := 3; END_VAR
r := 1 / 2;
r := s * 2.0 + r;
r := -r;
r := -1;
r := 100000;
r := INT_TO_REAL(i) * 0.5;
i := REAL_TO_INT(s);
IF r > 1 THEN r := 1.0; END_IF
IF 1 < r THEN r := 0.0; END_IF
IF 1.5 > 1 THEN r := 0.0; END_IF
IF r = s THEN i := 0; END_IF
FOR i := 1 TO 3 DO r := r + 1; END_FOR
END_PROGRAM`,
		},
		{
			name: "присваивание REAL в INT",
			src:  "PROGRAM P\nVAR i : INT; r : REAL; END_VAR\ni := r;\nEND_PROGRAM",
			want: []string{`line 3:1: cannot assign REAL to "i" of type INT (use INT_TO_REAL / REAL_TO_INT)`},
		},
		{
			name: "присваивание INT в REAL",
			src:  "PROGRAM P\nVAR i : INT; r : REAL; END_VAR\nr := i;\nEND_PROGRAM",
			want: []string{`line 3:1: cannot assign INT to "r" of type REAL (use INT_TO_REAL / REAL_TO_INT)`},
		},
		{
			// Закрывает наблюдение этапа 1: раньше RealLiteral проходил sema
			// и падал в codegen.
			name: "вещественный литерал в INT-переменную",
			src:  "PROGRAM P\nVAR i : INT; END_VAR\ni := 3.14;\nEND_PROGRAM",
			want: []string{`line 3:1: cannot assign REAL to "i" of type INT`},
		},
		{
			name: "инициализатор REAL у INT",
			src:  "PROGRAM P\nVAR i : INT := 1.5; END_VAR\nEND_PROGRAM",
			want: []string{`line 2:5: cannot assign REAL to "i" of type INT`},
		},
		{
			// Одна ошибка: результат смешения — Invalid, присваивание молчит.
			name: "смешение INT и REAL в арифметике",
			src:  "PROGRAM P\nVAR i : INT; r : REAL; END_VAR\nr := i + r;\nEND_PROGRAM",
			want: []string{`line 3:8: operands of "+" have different types: INT and REAL (use INT_TO_REAL / REAL_TO_INT)`},
		},
		{
			// Литерал берёт тип соседа-переменной, а не цели: `i + 1` — INT,
			// и ошибка именно на присваивании, а не «INT и REAL в +».
			name: "литерал адаптируется к операнду, а не к цели",
			src:  "PROGRAM P\nVAR i : INT; r : REAL; END_VAR\nr := i + 1;\nEND_PROGRAM",
			want: []string{`line 3:1: cannot assign INT to "r" of type REAL`},
		},
		{
			name: "сравнение INT с REAL",
			src:  "PROGRAM P\nVAR i : INT; r : REAL; END_VAR\nIF i > r THEN i := 0; END_IF\nEND_PROGRAM",
			want: []string{`line 3:6: operands of ">" have different types: INT and REAL`},
		},
		{
			name: "условие IF не BOOL",
			src: `PROGRAM P
VAR i : INT; r : REAL; END_VAR
IF r THEN r := 1.0; END_IF
IF 1 THEN i := 1; END_IF
IF i + 1 THEN i := 1; END_IF
END_PROGRAM`,
			want: []string{
				`line 3:4: IF condition must be BOOL (a comparison), got REAL`,
				`line 4:4: IF condition must be BOOL (a comparison), got INT`,
				`line 5:4: IF condition must be BOOL (a comparison), got INT`,
			},
		},
		{
			name: "BOOL в арифметике, присваивании, унарном минусе и сравнении",
			src: `PROGRAM P
VAR i : INT; r : REAL; END_VAR
i := (i > 1) + 1;
i := i > 1;
r := -(r > 1.0);
IF (i > 1) = (i < 3) THEN i := 0; END_IF
END_PROGRAM`,
			want: []string{
				`line 3:14: operator "+" is not applicable to BOOL`,
				`line 4:1: cannot assign BOOL to "i" of type INT`,
				`line 5:6: unary minus is not applicable to BOOL`,
				`line 6:12: cannot compare BOOL values`,
			},
		},
		{
			name: "FOR по REAL",
			src:  "PROGRAM P\nVAR r : REAL; END_VAR\nFOR r := 1.0 TO 2.0 DO r := r; END_FOR\nEND_PROGRAM",
			want: []string{
				`line 3:5: FOR variable "r" must be INT, got REAL`,
				`line 3:10: FOR start must be INT, got REAL`,
				`line 3:17: FOR end must be INT, got REAL`,
			},
		},
		{
			name: "FOR по INT с вещественным шагом",
			src:  "PROGRAM P\nVAR i : INT; END_VAR\nFOR i := 1 TO 3 BY 0.5 DO i := i; END_FOR\nEND_PROGRAM",
			want: []string{`line 3:20: FOR step must be INT, got REAL`},
		},
		{
			name: "REAL-литерал вне диапазона float",
			src:  "PROGRAM P\nVAR r : REAL; END_VAR\nr := 1e39;\nEND_PROGRAM",
			want: []string{`line 3:6: literal 1e39 out of range for REAL`},
		},
		{
			// В контексте INT вещественного литерала диапазон INT не касается:
			// ошибка одна — на присваивании.
			name: "диапазон INT к целому литералу в REAL-контексте не применяется",
			src:  "PROGRAM P\nVAR r : REAL; END_VAR\nr := -100000;\nr := 100000 * 2;\nEND_PROGRAM",
		},
		{
			name: "INT_TO_REAL: неверный тип аргумента",
			src:  "PROGRAM P\nVAR r : REAL; END_VAR\nr := INT_TO_REAL(r);\nEND_PROGRAM",
			want: []string{`line 3:17: argument of "INT_TO_REAL" must be INT, got REAL`},
		},
		{
			name: "REAL_TO_INT: неверный тип аргумента",
			src:  "PROGRAM P\nVAR i : INT; END_VAR\ni := REAL_TO_INT(i);\nEND_PROGRAM",
			want: []string{`line 3:17: argument of "REAL_TO_INT" must be REAL, got INT`},
		},
		{
			name: "результат конверсии несовместим с целью",
			src:  "PROGRAM P\nVAR i : INT; END_VAR\ni := INT_TO_REAL(i);\nEND_PROGRAM",
			want: []string{`line 3:1: cannot assign REAL to "i" of type INT`},
		},
		{
			name: "число аргументов конверсии",
			src:  "PROGRAM P\nVAR r : REAL; i : INT; END_VAR\nr := INT_TO_REAL();\nr := INT_TO_REAL(i, i);\nEND_PROGRAM",
			want: []string{
				`line 3:17: built-in function "INT_TO_REAL" expects 1 argument(s), got 0`,
				`line 4:17: built-in function "INT_TO_REAL" expects 1 argument(s), got 2`,
			},
		},
		{
			name: "именованный аргумент и => у конверсии",
			src:  "PROGRAM P\nVAR r : REAL; i : INT; END_VAR\nr := INT_TO_REAL(x := i);\nr := INT_TO_REAL(x => i);\nEND_PROGRAM",
			want: []string{
				`line 3:17: built-in function "INT_TO_REAL" takes positional arguments only`,
				`line 4:17: output binding "x" => is not applicable to built-in function "INT_TO_REAL"`,
			},
		},
		{
			name: "конверсия регистронезависима",
			src:  "PROGRAM P\nVAR r : REAL; i : INT; END_VAR\nr := int_to_real(i);\nEND_PROGRAM",
		},
		{
			name: "POU с именем встроенной функции",
			src:  "FUNCTION INT_TO_REAL : REAL\nINT_TO_REAL := 1.0;\nEND_FUNCTION",
			want: []string{`line 1:1: "INT_TO_REAL" is a reserved name (built-in function)`},
		},
		{
			// Закрывает долг «FUNCTION F : Bogus падает уже в codegen».
			// Тело с возвратной переменной Invalid каскада не даёт.
			name: "нерезолвящийся тип возврата функции",
			src:  "FUNCTION F : Bogus\nF := 100000;\nEND_FUNCTION",
			want: []string{`line 1:1: unknown type "Bogus"`},
		},
		{
			name: "тип возврата функции — ФБ",
			src:  "FUNCTION_BLOCK B\nEND_FUNCTION_BLOCK\nFUNCTION F : B\nEND_FUNCTION",
			want: []string{`line 3:1: function "F" cannot return a function block ("B")`},
		},
		{
			name: "тип возврата функции — PROGRAM",
			src:  "PROGRAM Q\nEND_PROGRAM\nFUNCTION F : Q\nEND_FUNCTION",
			want: []string{`line 3:1: "Q" is not a type`},
		},
		{
			// Позитивные строки 7–8: литерал адаптируется к REAL-входу, вызов
			// именованным аргументом; ошибки — строки 9 и 10.
			name: "функция с REAL: возврат, аргументы позиционные и именованные",
			src: halfSrc + `PROGRAM P
VAR r : REAL; i : INT; END_VAR
r := Half(1);
r := Half(x := r);
i := Half(1.0);
r := Half(i);
END_PROGRAM`,
			want: []string{
				`line 9:1: cannot assign REAL to "i" of type INT`,
				`line 10:10: argument for input "x" of function "Half" must be REAL, got INT`,
			},
		},
		{
			// Позитивная строка 8; ошибки — 9–12.
			name: "ФБ с REAL: входы, выходы, члены",
			src: avgSrc + `PROGRAM P
VAR a : Avg; r : REAL; i : INT; END_VAR
a(v := 1, m => r);
a(v := i);
a(m => i);
a.v := i;
i := a.m;
END_PROGRAM`,
			want: []string{
				`line 9:2: input "v" of "Avg" must be REAL, got INT`,
				`line 10:2: output "m" of "Avg" is REAL, cannot bind it to "i" of type INT`,
				`line 11:1: cannot assign INT to "a.v" of type REAL`,
				`line 12:1: cannot assign REAL to "i" of type INT`,
			},
		},
		{
			// Диапазон INT-литерала по-прежнему проверяется в INT-контексте,
			// в том числе через подсказку соседа-операнда.
			name: "диапазон INT сохранился: аргумент, операнд сравнения",
			src:  halfSrc + "PROGRAM P\nVAR i : INT; r : REAL; END_VAR\ni := i + 40000;\nIF 40000 > i THEN i := 0; END_IF\nEND_PROGRAM",
			want: []string{
				`line 7:10: literal 40000 out of range for INT`,
				`line 8:4: literal 40000 out of range for INT`,
			},
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

// TestReal — этап 4 плана REAL: дополняет TestTypes (не дублирует) кейсами
// в контекстах, которых там нет: тела FUNCTION и FUNCTION_BLOCK, члены
// экземпляров, вызовы функций в условии и в операндах, литералы под знаком и
// с экспонентой, адаптивный литерал в аргументах конверсий и во входах ФБ,
// отсутствие каскада у нерезолвящегося типа возврата.
func TestReal(t *testing.T) {
	// Функция с REAL-параметром (строки 1–4) и ФБ с REAL-входом и
	// REAL-выходом (строки 5–9); программа за ними начинается со строки 10.
	const pous = `FUNCTION Half : REAL
VAR_INPUT x : REAL; END_VAR
Half := x / 2;
END_FUNCTION
FUNCTION_BLOCK Integ
VAR_INPUT v : REAL; END_VAR
VAR_OUTPUT total : REAL; END_VAR
total := total + v;
END_FUNCTION_BLOCK
`
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			// Позитив: адаптивный литерал в инициализаторе (`:= 0`), в делении
			// (`1 / 2` — вещественное), под скобками, в REAL-локальной
			// функции, во входе ФБ (`g(v := 1)`, `g.v := 2`), в аргументе
			// конверсии (`REAL_TO_INT(3)`); минус над вызовом; цепочка
			// конверсий; экспонента в инициализаторе ФБ; условие по члену.
			// `i := 1 / 2` в INT-контексте остаётся целочисленным делением.
			name: "корректная программа: REAL во всех контекстах",
			src: `FUNCTION Half : REAL
VAR_INPUT x : REAL; END_VAR
VAR t : REAL := 3; END_VAR
Half := x / t;
END_FUNCTION
FUNCTION_BLOCK Integ
VAR_INPUT v : REAL; END_VAR
VAR_OUTPUT total : REAL; END_VAR
VAR dt : REAL := 2.5E-1; END_VAR
total := total + v * dt;
END_FUNCTION_BLOCK
PROGRAM P
VAR r : REAL := 0; s : REAL := -1.5; i : INT; g : Integ; END_VAR
r := 1 / 2;
r := (1 + 2) / 4;
r := -Half(r);
r := INT_TO_REAL(REAL_TO_INT(r));
i := REAL_TO_INT(3);
g(v := 1);
g.v := 2;
g(total => r);
IF g.total > 0 THEN r := g.total; END_IF
i := 1 / 2;
END_PROGRAM`,
		},
		{
			name: "INT-вход в REAL-возврат функции",
			src:  "FUNCTION F : REAL\nVAR_INPUT i : INT; END_VAR\nF := i;\nEND_FUNCTION",
			want: []string{`line 3:1: cannot assign INT to "F" of type REAL (use INT_TO_REAL / REAL_TO_INT)`},
		},
		{
			name: "смешение в теле ФБ",
			src:  "FUNCTION_BLOCK B\nVAR_INPUT v : REAL; END_VAR\nVAR n : INT; END_VAR\nn := n + v;\nEND_FUNCTION_BLOCK",
			want: []string{`line 4:8: operands of "+" have different types: INT and REAL (use INT_TO_REAL / REAL_TO_INT)`},
		},
		{
			// Тип члена и тип возврата функции участвуют в выводе как
			// операнды; `i + 1` берёт подсказку от i, а не от r слева.
			name: "смешение через член экземпляра, результат функции и подвыражение",
			src: pous + `PROGRAM P
VAR r : REAL; i : INT; g : Integ; END_VAR
r := g.total * i;
i := Half(r) + i;
IF r > i + 1 THEN i := 0; END_IF
END_PROGRAM`,
			want: []string{
				`line 12:14: operands of "*" have different types: REAL and INT`,
				`line 13:14: operands of "+" have different types: REAL and INT`,
				`line 14:6: operands of ">" have different types: REAL and INT`,
			},
		},
		{
			name: "условие IF не BOOL: член, вызов, вещественный литерал",
			src: pous + `PROGRAM P
VAR r : REAL; g : Integ; END_VAR
IF g.total THEN r := 1.0; END_IF
IF Half(r) THEN r := 1.0; END_IF
IF 2.5 THEN r := 1.0; END_IF
END_PROGRAM`,
			want: []string{
				`line 12:4: IF condition must be BOOL (a comparison), got REAL`,
				`line 13:4: IF condition must be BOOL (a comparison), got REAL`,
				`line 14:4: IF condition must be BOOL (a comparison), got REAL`,
			},
		},
		{
			name: "FOR по INT с REAL-границей и отрицательным вещественным шагом",
			src: `PROGRAM P
VAR i : INT; r : REAL; END_VAR
FOR i := 1 TO r DO i := i; END_FOR
FOR i := 1 TO 2 BY -0.5 DO i := i; END_FOR
END_PROGRAM`,
			want: []string{
				`line 3:15: FOR end must be INT, got REAL`,
				`line 4:20: FOR step must be INT, got REAL`,
			},
		},
		{
			// Ошибка одна — на литерале под минусом; граничные значения
			// float и экспонента законны.
			name: "диапазон REAL-литерала: под минусом, границы, экспонента",
			src: `PROGRAM P
VAR r : REAL; END_VAR
r := -1e39;
r := 3.4e38;
r := 1.5E3;
r := -3.4028235e38;
END_PROGRAM`,
			want: []string{`line 3:7: literal 1e39 out of range for REAL`},
		},
		{
			// Целый литерал в аргументе REAL_TO_INT адаптируется к REAL —
			// следствие решения об адаптивном литерале, не ошибка.
			name: "конверсии: позитив, адаптивный литерал в аргументе, цепочка",
			src: `PROGRAM P
VAR r : REAL; i : INT; END_VAR
r := INT_TO_REAL(3);
i := REAL_TO_INT(3);
i := REAL_TO_INT(2.5);
r := INT_TO_REAL(REAL_TO_INT(r));
r := INT_TO_REAL(i) / 2;
END_PROGRAM`,
		},
		{
			name: "конверсии: неверный тип выражения-аргумента",
			src: `PROGRAM P
VAR r : REAL; i : INT; END_VAR
r := INT_TO_REAL(r * 2);
i := REAL_TO_INT(i + 1);
END_PROGRAM`,
			want: []string{
				`line 3:17: argument of "INT_TO_REAL" must be INT, got REAL`,
				`line 4:17: argument of "REAL_TO_INT" must be REAL, got INT`,
			},
		},
		{
			name: "функция с REAL и INT входами: адаптивный литерал и REAL-литерал в INT-вход",
			src: `FUNCTION Scale : REAL
VAR_INPUT v : REAL; k : INT; END_VAR
Scale := v * INT_TO_REAL(k);
END_FUNCTION
PROGRAM P
VAR r : REAL; END_VAR
r := Scale(1, 2);
r := Scale(1.0, 2.0);
END_PROGRAM`,
			want: []string{`line 8:11: argument for input "k" of function "Scale" must be INT, got REAL (use INT_TO_REAL / REAL_TO_INT)`},
		},
		{
			name: "REAL-литерал во вход INT ФБ: вызов и член",
			src: `FUNCTION_BLOCK C
VAR_INPUT step : INT; END_VAR
END_FUNCTION_BLOCK
PROGRAM P
VAR c : C; END_VAR
c(step := 1.5);
c.step := 2.5;
END_PROGRAM`,
			want: []string{
				`line 6:2: input "step" of "C" must be INT, got REAL (use INT_TO_REAL / REAL_TO_INT)`,
				`line 7:1: cannot assign REAL to "c.step" of type INT`,
			},
		},
		{
			// Ошибка одна — на объявлении функции; ни `F := 1` в теле, ни
			// `F() + 1` в месте вызова каскада не дают.
			name: "нерезолвящийся тип возврата — без каскада в теле и в месте вызова",
			src: `FUNCTION F : Widget
F := 1;
END_FUNCTION
PROGRAM P
VAR r : REAL; END_VAR
r := F() + 1;
END_PROGRAM`,
			want: []string{`line 1:1: unknown type "Widget"`},
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

// TestInfo — side-table: адаптивные литералы получают тип контекста (в
// том числе под унарным минусом), сравнение — BOOL, вызванные конверсии
// попадают в UsedBuiltins.
func TestInfo(t *testing.T) {
	sf, info, errs := checkInfo(t, `PROGRAM P
VAR r : REAL; i : INT; END_VAR
r := 1 / 2;
r := -1;
i := REAL_TO_INT(r) + 1;
IF r > 1 THEN i := 0; END_IF
END_PROGRAM`)
	if len(errs) > 0 {
		t.Fatalf("неожиданные ошибки: %v", errs)
	}
	body := sf.POUs[0].(*ast.Program).Body
	assign := func(i int) *ast.AssignStatement { return body[i].(*ast.AssignStatement) }

	div := assign(0).Value.(*ast.BinaryExpr)
	for name, e := range map[string]ast.Expression{"1 / 2": div, "1": div.Left, "2": div.Right} {
		if got := info.Types[e]; got != sema.Real {
			t.Errorf("%s: тип %v, ожидался REAL", name, got)
		}
	}
	neg := assign(1).Value.(*ast.UnaryExpr)
	if info.Types[neg] != sema.Real || info.Types[neg.Operand] != sema.Real {
		t.Errorf("-1 в REAL-контексте: %v / %v, ожидался REAL", info.Types[neg], info.Types[neg.Operand])
	}
	add := assign(2).Value.(*ast.BinaryExpr)
	if info.Types[add] != sema.Int || info.Types[add.Left] != sema.Int || info.Types[add.Right] != sema.Int {
		t.Errorf("REAL_TO_INT(r) + 1: %v / %v / %v, ожидался INT", info.Types[add], info.Types[add.Left], info.Types[add.Right])
	}
	cond := body[3].(*ast.IfStatement).Condition
	if info.Types[cond] != sema.Bool {
		t.Errorf("условие IF: %v, ожидался BOOL", info.Types[cond])
	}
	if !info.UsedBuiltins["REAL_TO_INT"] || info.UsedBuiltins["INT_TO_REAL"] {
		t.Errorf("UsedBuiltins: %v, ожидался только REAL_TO_INT", info.UsedBuiltins)
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
