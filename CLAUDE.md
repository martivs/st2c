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
- Парсер: golden — `testdata/*.st` разбирается, `Program.String()` сверяется с
  `testdata/*.ast`; эталоны перегенерируются `go test ./src/parser -update`
  (diff просматривать вручную). Плюс негативные тесты: вход → подстрока
  сообщения об ошибке с номером строки.

### Что реализовано

**`src/main.go`** — читает `.st`-файл, прогоняет лексер → парсер, печатает AST через `Program.String()`. Лексер одноразовый (pull-модель), поэтому вывод потока токенов Этапа 1 в git-истории, а не в текущем `main`.

**`src/lexer/lexer.go`** — полный лексер:
- `TokenType` (iota): `EOF`, `ILLEGAL`, `IDENT`, `INT_LIT`, операторы (`+ - * / > < =`, `:=`), разделители (`: ; ( )`), 13 ключевых слов
- `Token{Type, Literal, Line}` — структура токена с номером строки
- `Lexer.NextToken()` — побайтовый разбор: пропуск пробелов/переносов, `:=` vs `:`, идентификаторы → ключевые слова (case-insensitive), целые литералы
- Токены `LPAREN`/`RPAREN` добавлены на Этапе 2 для скобок в выражениях

**`src/ast/ast.go`** — узлы дерева через интерфейсы:
- Интерфейсы `Node` (`String()`, `Line()`), `Statement`, `Expression`; маркерные методы `statementNode()`/`expressionNode()` разделяют операторы и выражения на уровне типов
- Узлы: `Program`, `VarDecl`, `AssignStatement`, `IfStatement`, `ForStatement`, `Identifier`, `IntLiteral`, `BinaryExpr` (арифметика и сравнения — единый тип, различаются полем `Op`)
- У каждого узла `String()` рекурсивно печатает поддерево с отступами; поле `Tok lexer.Token` хранит якорь для номера строки

**`src/parser/parser.go`** — recursive descent:
- `New(*lexer.Lexer) *Parser`, `ParseProgram() (*ast.Program, error)`
- Окно из двух токенов (`cur`/`peek`), хелперы `nextToken`/`expect`/`curIs`/`peekIs`
- Правила: `parseProgram`, `parseVarBlock`, `parseStatements`, `parseStatement`, `parseAssign`, `parseIf`, `parseFor`
- Вложенные конструкции поддерживаются: `parseStatement` для `IF`/`FOR` рекурсивно вызывает `parseStatements` для тела, поэтому вложенность любой глубины разбирается без спец-обработки
- Приоритет операций — каскад по уровням: `parseExpression` (`> < =`) → `parseAdditive` (`+ -`) → `parseMultiplicative` (`* /`) → `parsePrimary` (`IDENT`, `INT_LIT`, `( expr )`); левая ассоциативность через цикл
- Ошибки — **fail-fast**: поле `err`, `expect()`/`fail()` формируют сообщение с номером строки; `ParseProgram` возвращает первую ошибку
- `optionalSemicolon()` — необязательный `;` после `END_IF`/`END_FOR` (есть в `example.st`)

```
go run ./src examples/example.st
# ST source loaded: 214 bytes
#
# Program(Example)
#   VarDecl(x : INT)
#   ...
#   For(x)
#     ...
#       Assign(sum :=)
#         Binary(+)
#           Ident(sum)
#           Ident(x)
#   If
#     Cond:
#       Binary(>) ...
```

### example.st — MVP-программа
Покрывает все целевые конструкции: объявление переменных (`VAR`), присваивание (`:=`), цикл `FOR ... TO ... DO`, условие `IF / ELSE`.

## Рабочий стиль
Этапы реализуются строго по одному. Переход к следующему — только после явного «переходим дальше» от пользователя.

## Следующий шаг — Этап 3: Codegen
Пакет `codegen` должен принимать корневой узел AST (`*ast.Program`) и генерировать эквивалентный код на C.
Вход: `*ast.Program`. Выход: строка с C-кодом (или запись в файл).
Обход дерева — рекурсивный (по аналогии с `String()`): объявления `VAR` → декларации переменных, `AssignStatement`, `IfStatement` (`if/else`), `ForStatement` (`for`), выражения `BinaryExpr` с расстановкой скобок по приоритету.
