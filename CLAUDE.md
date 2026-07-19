# ST → C Transpiler

Транспилятор: ST (Structured Text, IEC 61131-3) → C. Язык реализации: Go, стандартная библиотека.

## MVP-ограничения ST
- Только целочисленные переменные (`INT`)
- Конструкции: `IF / ELSE`, цикл `FOR`
- Простая арифметика над переменными

> Исходно MVP планировался без вложенных конструкций, но рекурсивный спуск
> в парсере поддерживает вложенность «бесплатно» (`IF`/`FOR` внутри тел
> друг друга на любую глубину) — см. Этап 2.

## Архитектура
Четыре последовательных этапа, каждый — отдельный Go-пакет:

| Этап | Пакет     | Статус         |
|------|-----------|----------------|
| 1    | `lexer`   | готов          |
| 2    | `parser`  | готов          |
| 2    | `ast`     | готов          |
| 3    | `codegen` | заглушка       |

## Текущее состояние (Этап 2 завершён)

### Созданные файлы
```
st2c/
├── go.mod                    # module st2c, go 1.22
├── examples/                 # примеры ST-программ для разработки
│   ├── example.st
│   ├── nested_if_in_for.st
│   ├── nested_for_in_if.st
│   └── deeply_nested.st
└── src/
    ├── main.go               # точка входа: лексер → парсер → печать AST
    ├── lexer/
    │   ├── lexer.go          # package lexer (готов)
    │   └── lexer_test.go     # table-driven тесты токенов
    ├── parser/
    │   ├── parser.go         # package parser (готов)
    │   ├── parser_test.go    # golden-тесты AST + тесты ошибок
    │   └── testdata/         # 4 примера *.st + эталоны *.ast
    ├── ast/ast.go            # package ast (готов)
    └── codegen/codegen.go    # package codegen (заглушка)
```

### Тесты
- `go test ./...` — обязательно зелено перед коммитом.
- Лексер: table-driven (вход → список `(Type, Literal)`), регистр ключевых
  слов, номера строк.
- Парсер: golden — `testdata/*.st` разбирается, `SourceFile.String()` сверяется с
  `testdata/*.ast`; эталоны перегенерируются `go test ./src/parser -update`
  (diff просматривать вручную). Плюс негативные тесты: вход → подстрока
  сообщения об ошибке с номером строки.

### Что реализовано

**`src/main.go`** — читает `.st`-файл, прогоняет лексер → парсер (`ParseSourceFile`), печатает AST каждого POU через `String()`. Лексер одноразовый (pull-модель), поэтому вывод потока токенов Этапа 1 в git-истории, а не в текущем `main`.

**`src/lexer/lexer.go`** — полный лексер:
- `TokenType` (iota): `EOF`, `ILLEGAL`, `IDENT`, `INT_LIT`, операторы (`+ - * / > < = <= >= <>`, `:=`), разделители (`: ; , ( )`), 20 ключевых слов (включая `BY`, семейство `VAR_INPUT/VAR_OUTPUT/VAR_IN_OUT/VAR_TEMP` и квалификаторы `CONSTANT`/`RETAIN`)
- `Token{Type, Literal, Line, Col}` — позиция токена: строка и колонка (1-based, в рунах; `lineStart` в лексере, сброс на `\n`)
- `Lexer.NextToken()` — посимвольный разбор: пропуск пробелов/переносов, комментарии, двухсимвольные операторы (`:=`, `<=`, `>=`, `<>`) через `peek()`, идентификаторы → ключевые слова (case-insensitive), целые литералы
- Комментарии: блочные `(* ... *)` с вложенностью (3-я ред. IEC) и строчные `// ...`; незакрытый `(*` → `ILLEGAL`-токен с позицией начала; одиночные `(` и `/` остаются `LPAREN`/`SLASH`
- `tokenNames` для ключевых слов строится из `keywords` в `init()` — новое слово добавляется в одном месте

**`src/ast/ast.go`** — узлы дерева через интерфейсы:
- Интерфейсы `Node` (`String()`, `Line()`), `Statement`, `Expression`; маркерные методы `statementNode()`/`expressionNode()` разделяют операторы и выражения на уровне типов
- `ast.Op` — собственный enum операций (`ADD … NE`, `NEG`) с `String()`, возвращающим исходные символы (`+`, `<=`, `<>`); AST отвязан от `lexer.TokenType`
- Корень — `SourceFile{POUs []POU}`: компилируемая единица как список POU; интерфейс `POU` (маркер `pouNode()`), пока его реализует только `Program` — `FUNCTION`/`FUNCTION_BLOCK` лягут рядом без смены корня
- Узлы: `Program` (`VarBlocks []*VarBlock`), `VarBlock{Kind VarKind, Constant, Retain bool, Decls}` — вид блока (`VAR`/`VAR_INPUT`/…) как данные (enum `VarKind` со `String()`), `VarDecl{Names []string, TypeName string, Init Expression}` — список имён (`a, b, c : INT;`), тип строкой (встроенный или пользовательский — решает sema), необязательный инициализатор (`x : INT := 5;`), `AssignStatement` (`Target Expression` — lvalue, пока всегда `*Identifier`; позже `IndexExpr`/`MemberExpr`), `IfStatement`, `ForStatement` (`Var *Identifier`, необязательный `Step Expression`, nil → шаг 1), `Identifier`, `IntLiteral`, `BinaryExpr` (арифметика и сравнения — единый тип, различаются полем `Op ast.Op`), `UnaryExpr` (унарный минус; позже `+`/`NOT`)
- У каждого узла `String()` рекурсивно печатает поддерево с отступами; поле `Tok lexer.Token` хранит якорь для номера строки

**`src/parser/parser.go`** — recursive descent:
- `New(*lexer.Lexer) *Parser`, `ParseSourceFile() (*ast.SourceFile, error)` — единственная точка входа: цикл по POU до EOF, диспетчер по стартовому ключевому слову (пока только `PROGRAM`; `FUNCTION`/`FUNCTION_BLOCK` — будущие ветки `switch`); мусор на верхнем уровне → ошибка
- Окно из двух токенов (`cur`/`peek`), хелперы `nextToken`/`expect`/`curIs`/`peekIs`
- Правила: `parseProgram`, `parseVarBlocks` (цикл по семейству `VAR*` через таблицу `varBlockKinds`, квалификаторы `CONSTANT`/`RETAIN`), `parseVarDecl` (список имён через запятую, тип, необязательный `:= init`), `parseType` (ключевое слово `INT` или любой `IDENT` — пользовательские типы не падают, допустимость проверит sema), `parseStatements`, `parseStatement`, `parseAssign` (цель — primary-lvalue, пока только идентификатор), `parseIf`, `parseFor` (необязательный `BY step` перед `DO`)
- Вложенные конструкции поддерживаются: `parseStatement` для `IF`/`FOR` рекурсивно вызывает `parseStatements` для тела, поэтому вложенность любой глубины разбирается без спец-обработки
- Выражения — Pratt / precedence climbing: одна `parseExpression(minPrec int)` + таблица `prec map[lexer.TokenType]int` (уровни по IEC: `= <>` слабее `< > <= >=`, дальше `+ -`, `* /`); новый бинарный оператор = строка в `prec` и `binOps`. `parseUnary` (унарный `-`, рекурсивно) → `parsePrimary` (`IDENT`, `INT_LIT`, `( expr )`); левая ассоциативность — правый операнд с `pr+1`
- Ошибки — **fail-fast**: поле `err`, `expect()`/`fail()` формируют сообщение с номером строки; `ParseProgram` возвращает первую ошибку
- `optionalSemicolon()` — необязательный `;` после `END_IF`/`END_FOR` (есть в `example.st`)

```
go run ./src examples/example.st
# ST source loaded: 214 bytes
#
# Program(Example)
#   VarBlock(VAR)
#     VarDecl(x : INT)
#   ...
#   For(x)
#     ...
#       Assign
#         Target:
#           Ident(sum)
#         Value:
#           Binary(+)
#             Ident(sum)
#             Ident(x)
#   If
#     Cond:
#       Binary(>) ...
```

### example.st — MVP-программа
Покрывает все целевые конструкции: объявление переменных (`VAR`), присваивание (`:=`), цикл `FOR ... TO ... DO`, условие `IF / ELSE`.

## Рабочий стиль
Этапы реализуются строго по одному. Переход к следующему — только после явного «переходим дальше» от пользователя.

## Следующий шаг — Этап 3: Codegen
Пакет `codegen` должен принимать корневой узел AST (`*ast.SourceFile`) и генерировать эквивалентный код на C.
Вход: `*ast.SourceFile`. Выход: строка с C-кодом (или запись в файл).
Обход дерева — рекурсивный (по аналогии с `String()`): объявления `VAR` → декларации переменных, `AssignStatement`, `IfStatement` (`if/else`), `ForStatement` (`for`), выражения `BinaryExpr` с расстановкой скобок по приоритету.
