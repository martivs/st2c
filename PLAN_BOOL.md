# План: тип BOOL и логические операторы AND / OR / XOR / NOT (временный документ)

Рабочий план реализации булева типа и логических операций в st2c. **Временный**:
по завершении этапа 5 удаляется вместе с `BOOL_PROMPTS.md`, существенное
переезжает в `CLAUDE.md` — так же, как это было сделано с `PLAN_REAL.md` и
`REAL_PROMPTS.md` (коммит 0.1.14).

Этот файл — источник контекста для каждой новой сессии: этап читается отсюда,
результат отмечается здесь же, в том же коммите, что и код этапа.

**Один этап = одна сессия = один коммит.** Следующий этап не начинать без
явного «переходим дальше» от Mart.

---

## Зачем

Слой типов уже построен (работа по `REAL`): тип — сущность `sema.TypeKind`,
типы выражений выводятся снизу вверх (`typeOf`/`hintOf`) и складываются в
side-table `sema.Info`, codegen берёт маппинг из четырёх таблиц
(`cTypes`/`cWideTypes`/`cZeros`/`cFormats`). Внутренний тип `Bool` в sema тоже
уже есть — как результат сравнения (`src/sema/sema.go:50`).

Чего нет: объявить `x : BOOL;` нельзя (`builtinType` знает только `INT` и
`REAL`, `src/sema/sema.go:68`), булевых литералов нет, логических операций нет
вовсе. Практическое следствие — условие `IF` обязано быть **одним** сравнением:
`IF a > 1 AND b < 2 THEN` не выражается никак, флаг состояния не сохранить,
`IF NOT done THEN` не написать. Это следующий пункт дорожной карты `CLAUDE.md`
(«скалярные типы: `BOOL` как объявляемый тип — `TypeKind` уже есть»).

Результат по завершении: `BOOL` объявляется, хранится в состоянии POU,
передаётся во входы/выходы функций и ФБ и печатается драйвером; `TRUE`/`FALSE` —
литералы; `AND`, `OR`, `XOR`, `NOT` и синоним `&` работают с приоритетами IEC;
условие `IF` — любое выражение типа BOOL; смешение BOOL с INT/REAL
диагностируется sema.

Объём работы меньше, чем у `REAL`: строить слой типов не нужно, нужно аккуратно
в него встроиться.

## Принятые решения (не переоткрывать)

1. **Набор операций**: `AND`, `OR`, `XOR`, `NOT` и IEC-синоним `&` для `AND`.
   Приоритет по IEC, от слабого к сильному:
   `OR` < `XOR` < `AND`/`&` < `= <>` < `< > <= >=` < `+ -` < `* /` <
   унарные (`-`, `NOT`) < постфиксы (`.`, вызов).
2. **`BOOL` → `_Bool`** — ключевое слово C99, новых `#include` не требуется.
   Прямое следствие, которое и делает решение правильным: golden `.c` всех
   двенадцати существующих примеров остаются **побайтово прежними**
   (регрессионный сигнал этапа 4). `_Bool` самонормализуется: присваивание
   любого ненулевого даёт ровно 1, поэтому `%d` в драйвере печатает `0`/`1`.
   Ноль — `0` (`cZeros`). `bool` + `<stdbool.h>` отвергнут: либо лишний include
   во всех эталонах, либо условная эмиссия include ради читаемости.
3. **Логические операции в C**: `AND` → `&&`, `OR` → `||`, `NOT` → `!`,
   `XOR` → `!=`. `!=` вместо `^` сознательно: на `_Bool` это то же самое, но не
   зависит от нормализации значения, если BOOL когда-нибудь переедет на
   `uint8_t`. Короткое замыкание `&&`/`||` ненаблюдаемо: функции ST без
   состояния и побочных эффектов, ФБ в выражениях не вызываются (это проверяет
   sema), так что единственное отличие — невычисление правого операнда — ничем
   себя не проявляет.
4. **`&` не оставляет следа в AST**: токены `AND` и `AMP` отображаются в одну
   операцию `ast.AND`. Обратная печать исходника — задача будущего форматтера
   (у него уже есть такой же долг: литерал ключевых слов приводится к верхнему
   регистру).
5. **Адаптивного булева литерала нет.** Адаптивность целого литерала (решение 2
   работы по REAL) касается только пары INT/REAL. `b : BOOL := 1;`,
   `i := TRUE;`, `IF i THEN` (INT в условии) — ошибки sema. Конверсий
   `BOOL_TO_INT` / `INT_TO_BOOL` в этой работе нет (долг на будущее; таблица
   `builtins` к ним готова).
6. **Сравнение BOOL**: `=` и `<>` над BOOL **разрешены** (по IEC равенство
   булевых законно и нужно на практике: `IF done = enabled THEN`), порядковые
   `< > <= >=` над BOOL — **ошибка**. Запрет порядковых сохраняет диагностику
   известной ловушки `a < b < c`: она разбирается как `(a < b) < c`, левый
   операнд — BOOL, ошибка остаётся. В codegen решение не стоит ничего: `==`/`!=`
   в `cOps` уже есть.

---

## Чек-лист этапов

Статус: `[ ]` не начат · `[~]` в работе · `[x]` готов (дописать номер коммита).
Обновляется в том же коммите, что и код этапа.

| Этап | Содержание | Объём | Статус |
|------|------------|-------|--------|
| 1 | лексер + AST: ключевые слова, `&`, `ast.Op`, `BoolLiteral` | ~60 строк + ~60 тестов | [ ] |
| 2 | парсер: приоритеты, `NOT`, `TRUE`/`FALSE`, `: BOOL` | ~40 строк + ~120 тестов | [ ] |
| 3 | sema: BOOL как тип, логические операции, `NOT`, сравнение | ~120 строк + ~200 тестов | [ ] |
| 4 | codegen: четыре таблицы, `cOps`, `NOT`, булев литерал | ~30 строк | [ ] |
| 5 | корпус, эталоны, `CLAUDE.md`, удаление плана | ~60 строк + корпус | [ ] |

Каждый этап оставляет дерево зелёным (`go build ./...`, `go vet ./...`,
`go test ./...`) и идёт отдельным коммитом.

---

## Этап 1 — лексер и AST

**Цель:** новые токены и узлы существуют. Парсер их ещё не знает, поэтому
`x : BOOL;` и `a AND b` после этого этапа падают **синтаксической** ошибкой
парсера (`BOOL` перестал быть `IDENT`, `AND` — не оператор) — корректное
промежуточное состояние, все прежние тесты остаются зелёными.

### `src/lexer/lexer.go`

- Семь новых ключевых слов: `BOOL`, `TRUE`, `FALSE`, `AND`, `OR`, `XOR`, `NOT` —
  константы в enum `TokenType` (`lexer.go:37-62`) и строки в `keywords`
  (`lexer.go:91-100`). В `tokenNames` их вписывать **не надо**: ключевые слова
  туда заливает `init()` (`lexer.go:78-82`).
- Новый токен-оператор `AMP` (`&`): константа в enum рядом с прочими
  операторами **и строка в `tokenNames` руками** (`lexer.go:68-76`) — не-ключевые
  токены автоматически туда не попадают, иначе `String()` вернёт `UNKNOWN` в
  сообщениях парсера (та же ловушка, что была с `REAL_LIT`).
- Ветка разбора в `switch` `NextToken` (`lexer.go:148-213`), рядом с
  односимвольными операторами: `case ch == '&'`. Двухсимвольного `&&` в ST нет,
  поэтому ловушки «порядок веток» (как у `=>` перед `=`) здесь не возникает.
- `readIdent` не трогать: `TRUE`, `and`, `NoT` уже разбираются как
  идентификаторы и находятся в `keywords` по `strings.ToUpper` (`lexer.go:290`).

### `src/ast/ast.go`

- `ast.Op` (`ast.go:45-75`): бинарные `AND`, `OR`, `XOR` и унарная `NOT`.
  В `opNames` они печатаются **словами** (`AND`, `OR`, `XOR`, `NOT`), а не
  символами — `Binary(AND)`, `Unary(NOT)` в golden `.ast`.
- `BoolLiteral{Value bool, Tok lexer.Token}` рядом с `IntLiteral`/`RealLiteral`
  (`ast.go:381-402`): маркерный `expressionNode()`, `Line()`, `String()` →
  `Bool(TRUE)` / `Bool(FALSE)` — **всегда в верхнем регистре**, независимо от
  написания в исходнике, чтобы golden `.ast` не зависел от регистра.

### Тесты этапа 1

`src/lexer/lexer_test.go`, табличный `TestAllTokens` (`lexer_test.go:44`) —
кейсы: `a AND b OR NOT c`, `x & y` (`AMP` отдельным токеном), `b := TRUE;`,
`v : BOOL;`, регистр (`and`, `Not`, `TrUe`, `xor`), и то, что `AND1`, `NOTx`,
`ORDER` остаются `IDENT` (ключевое слово — целиком, а не префикс).
Плюс кейс в `TestColumns` (`lexer_test.go:357`) на колонку после `&`.

---

## Этап 2 — парсер

**Цель:** новый синтаксис доходит до sema. После этапа `x : BOOL;` даёт ошибку
sema `unknown type "BOOL"`, а `i AND j` над INT-переменными проходит sema как
арифметика (она ещё не знает логических операций) и падает в codegen
`unsupported operator AND` — оба состояния промежуточные и правильные.

### `src/parser/parser.go`

- **Перенумерация таблицы `prec`** (`parser.go:361-368`) — уровни по IEC:

  ```go
  var prec = map[lexer.TokenType]int{
      lexer.OR:  1,
      lexer.XOR: 2,
      lexer.AND: 3, lexer.AMP: 3,
      lexer.EQ:  4, lexer.NE: 4,
      lexer.LT:  5, lexer.GT: 5, lexer.LE: 5, lexer.GE: 5,
      lexer.PLUS: 6, lexer.MINUS: 6,
      lexer.STAR: 7, lexer.SLASH: 7,
  }
  ```

  `lowestPrec` остаётся `1`. **Относительный порядок старых уровней сохранён**,
  поэтому разбор существующих программ не меняется — это проверяется golden
  `.ast` (см. ниже).
- `binOps` (`parser.go:373-380`): `lexer.OR` → `ast.OR`, `lexer.XOR` → `ast.XOR`,
  `lexer.AND` → `ast.AND`, `lexer.AMP` → `ast.AND` (решение 4: `&` неотличим от
  `AND` в дереве).
- `parseUnary` (`parser.go:400-412`): ветка `lexer.NOT` →
  `&ast.UnaryExpr{Op: ast.NOT, Operand: p.parseUnary(), Tok: tok}` — рекурсивно,
  как у минуса: `NOT NOT b` и `NOT -x` разбираются (второе отвергнет sema).
  Комментарий «сюда же позже лягут унарный `+` и NOT» пора поправить.
- `parsePrimary` (`parser.go:414-436`): ветки `lexer.TRUE` и `lexer.FALSE` →
  `&ast.BoolLiteral{Value: true/false, Tok: tok}`. Постфиксный цикл после атома
  трогать не надо (`TRUE.x` синтаксически пройдёт, отвергнет sema).
- `parseType` (`parser.go:212-224`): `lexer.BOOL` в `case` рядом с `lexer.INT` и
  `lexer.REAL`.

### Тесты этапа 2

`src/parser/parser_test.go`, табличный `TestExpressionStructure`
(`parser_test.go:85`, формат: `expr` → ожидаемый `String()` с отступами):

| Выражение | Ожидаемая форма |
|-----------|-----------------|
| `a OR b AND c` | `OR{Ident(a), AND{b, c}}` |
| `a AND b OR c` | `OR{AND{a, b}, Ident(c)}` |
| `a OR b XOR c AND d` | `OR{a, XOR{b, AND{c, d}}}` |
| `NOT a AND b` | `AND{Unary(NOT){a}, b}` |
| `NOT (a AND b)` | `Unary(NOT){AND{a, b}}` |
| `x > 1 AND y < 2` | `AND{Binary(>), Binary(<)}` |
| `a OR b OR c` | `OR{OR{a, b}, c}` (левая ассоциативность) |
| `a & b` | ровно то же дерево, что у `a AND b` |
| `b = TRUE` | `Binary(=){Ident(b), Bool(TRUE)}` |
| `NOT NOT b` | `Unary(NOT){Unary(NOT){b}}` |

Плюс: `: BOOL` в `VAR` (по образцу `TestVarBlocks`, `parser_test.go:373`) и в
типе возврата функции (по образцу `TestRealLiteral`, `parser_test.go:451`);
негативные в `TestParseErrors` (`parser_test.go:532`) — `x := AND b;`
(`expected expression, got AND`), `x := NOT;`, `x := a AND;`.

**Обязательная проверка этапа:** `go test ./src/parser` **без** `-update` —
восемь golden `.ast` должны совпасть побайтово. Если что-то разъехалось,
ошибка в перенумерации `prec`, а не в эталонах.

---

## Этап 3 — sema (ядро работы)

`src/sema/sema.go`. Слой типов уже есть, задача — встроить в него BOOL.

### BOOL как объявляемый тип

- `builtinType` (`sema.go:68-76`): `case "BOOL": return Bool, true`.
- Комментарий у `TypeKind.Bool` (`sema.go:50`) «объявить `x : BOOL;` пока
  нельзя» снять; шапку пакета (`sema.go:7-13`) дополнить абзацем про BOOL.
- `scalarOnly` (`sema.go:216-221`) — пропускать и `Bool`. Иначе `hintOf` у
  BOOL-переменной, члена ФБ и возврата функции вернёт `Unknown`, и контекст
  операндов потеряется.
- `returnKind` (`sema.go:660-665`) — добавить `Bool`: функция может возвращать
  BOOL (`FUNCTION IsHot : BOOL`).
- Правок не требуют (BOOL проходит через `kindOf` → `builtinType` сам):
  `resolveType`, `functionInputs`, `fbMember`/`resolveMember`, `checkArg`,
  `checkFBCall`, `checkBody` (инициализаторы).

### Литерал и унарный NOT

- `infer` (`sema.go:425-474`): ветка `*ast.BoolLiteral` → `Bool` (диапазонов
  проверять нечего).
- `infer`, `*ast.UnaryExpr`: до существующей ветки `NEG` — обработка `ast.NOT`:
  операнд выводится с `want = Bool`, обязан быть `Bool`, результат `Bool`;
  иначе ошибка на токене оператора, например
  `line N:C: NOT requires a BOOL operand, got INT`. Трюк `NEG` над целым
  литералом (`-32768`, `sema.go:450-458`) не трогать.
- `exprTok` (`sema.go:370-388`) — ветка `*ast.BoolLiteral` (позиция для
  сообщений об условии `IF` и границах `FOR`).
- `hintOf` (`sema.go:526-564`): `*ast.BoolLiteral` → `Bool`; `UnaryExpr` с
  `NOT` → `Bool`; `BinaryExpr` с логической операцией → `Bool`.

### Логические операции и сравнение — `inferBinary`

`inferBinary` (`sema.go:482-512`) переписывается вокруг классификации операции.
Ввести `isLogical(op)` (`AND`/`OR`/`XOR`) рядом с существующей `isComparison`
(`sema.go:514-520`) и развести три случая:

1. **Логическая**: контекст обоих операндов — всегда `Bool` (подсказка соседа не
   нужна и не используется), оба обязаны быть `Bool`, результат `Bool`. Ошибка
   вида `operator "AND" requires BOOL operands, got INT and BOOL` на токене
   оператора.
2. **Сравнение**: `Bool` допускается только для `EQ`/`NE` (решение 6), результат
   `Bool`. Для `LT`/`LE`/`GT`/`GE` с BOOL-операндом — ошибка
   `cannot order BOOL values` (прежняя формулировка
   `cannot compare BOOL values` больше не годится: сравнивать теперь можно).
   Правило «операнды одного типа» остаётся общим.
3. **Арифметика**: `Bool` в операнде — прежняя ошибка
   `operator %q is not applicable to BOOL`.

Механизм подсказок (`hintOf` соседа приоритетнее `want` сверху) для
арифметики и сравнения сохраняется без изменений.

### Мелочи и сообщения

- `checkAssign` (`sema.go:341-346`) — логика не меняется (`BOOL := BOOL`
  законно, `INT := BOOL` даёт `cannot assign BOOL to "i" of type INT` без
  подсказки про конверсии, что верно: их нет). Убрать из комментария «BOOL в
  цель нельзя».
- Сообщение об условии `IF` (`sema.go:309-311`): «IF condition must be BOOL (a
  comparison)» → «IF condition must be BOOL»: условие больше не обязано быть
  сравнением.
- `checkIntRange` при `want == Bool` не трогать: литерал остаётся `Int`, ошибку
  выдаст присваивание («cannot assign INT to "b" of type BOOL»).

### Тесты этапа 3

Новая табличная функция `TestBool` в `src/sema/sema_test.go` (формат кейса —
`src` + `want []string`, см. `TestTypes`, `sema_test.go:505`).

Позитив (ошибок быть не должно): `b : BOOL := TRUE;`; `b := x > 1;`;
`b := b1 AND NOT b2;`; `b := b1 & b2;`; `IF b THEN`; `IF NOT b OR (x > 1) THEN`;
`b := b1 = b2;`; BOOL как вход и выход `FUNCTION_BLOCK` (включая
`inst(En := TRUE, Out => flag)`); BOOL как вход и как тип возврата `FUNCTION`.

Негатив: `b := 1;` · `i := TRUE;` · `b := TRUE + 1;` · `i AND j` над INT
(`requires BOOL operands`) · `NOT i` · `-b` (прежняя ошибка про унарный минус) ·
`b1 < b2` (`cannot order BOOL values`) · `IF i THEN` (`must be BOOL`) ·
`FOR b := TRUE TO FALSE DO` · `b AND 1` · аргумент `INT` в BOOL-вход функции.

Плюс кейс в `TestInfo` (`sema_test.go:963`): у `b1 AND b2` и у `TRUE` в
side-table записан `sema.Bool`.

Прежние тесты (`TestCheck`, `TestTypes`, `TestReal`, `TestFunctionBlocks`,
`TestExamplesClean`) правиться не должны — кроме случаев, где менялась
формулировка сообщения (условие `IF`, `cannot order BOOL values`): там поправить
ожидаемые подстроки.

---

## Этап 4 — codegen

`src/codegen/codegen.go`. Этап короткий: весь маппинг типов уже сведён в четыре
таблицы, операции — в одну.

- Таблицы (`codegen.go:178-207`): `cTypes["BOOL"] = "_Bool"`,
  `cZeros["BOOL"] = "0"`, `cFormats["BOOL"] = "%d"`. В `cWideTypes` BOOL
  **не добавлять**: `FOR` по BOOL отвергает sema, а ошибка `cWideType` остаётся
  внутренней защитой (ровно как с REAL).
- `cOps` (`codegen.go:849-853`): `ast.AND: "&&"`, `ast.OR: "||"`,
  `ast.XOR: "!="` (решение 3 — почему `!=`, а не `^`, объяснить комментарием).
- `expr` (`codegen.go:858+`): ветка `*ast.BoolLiteral` → `"1"` / `"0"`;
  в ветке `*ast.UnaryExpr` снять жёсткое `if e.Op != ast.NEG` и добавить `NOT` →
  `"(!" + operand + ")"`. Скобки вокруг каждой операции эмитятся и так, поэтому
  `a && b || c` без скобок не возникает и `-Wparentheses` молчит.
- `reservedCNames` (`codegen.go:260-277`) править **не нужно**: `_Bool` там уже
  есть, а `TRUE`/`FALSE`/`AND`/`OR`/`XOR`/`NOT` стали ключевыми словами ST и
  переменными быть не могут.
- Драйвер (`emitDriver`, `codegen.go:623`) правок не требует: спецификатор
  берётся из `cFormats`, `_Bool` в varargs промоутится до `int`.

**Регрессионный сигнал этапа:** golden `.c` всех двенадцати существующих
примеров обязаны остаться **побайтово прежними**:

```bash
set -o pipefail
go test ./src/codegen -update && git diff --stat src/codegen/testdata   # ожидается: пусто
```

Ловушка (обожглись на этапе 3 REAL): при прогоне через пайп (`go test … | tail`)
провал тестов не останавливает `&&`-цепочку, и эталоны перезаписываются неверным
выводом. Запускать с `set -o pipefail`; восстанавливать
`git checkout -- src/codegen/testdata`.

---

## Этап 5 — корпус, эталоны, документация

### `examples/bool_all.st`

Новый пример — единый корпус живёт только в `examples/` (решение 9 `CLAUDE.md`).
Должен покрыть: BOOL-переменные с инициализаторами `TRUE`/`FALSE`; присваивание
результата сравнения в BOOL; все четыре операции; проверку приоритета
(`a OR b AND c` против `(a OR b) AND c` — две переменные с разными значениями);
хотя бы один `&`; `IF` по BOOL-переменной и по `NOT`; BOOL как вход и выход
`FUNCTION_BLOCK`; BOOL-вход и BOOL-возврат `FUNCTION`; флаг, накапливаемый в
целочисленном `FOR` (например, «встретилось ли значение, кратное трём»).
Значения `.expected` посчитать вручную и показать расчёт: BOOL печатается
драйвером как `0`/`1`.

### Эталоны и тесты

- `src/parser/testdata/bool_all.ast` — парсерный golden (`TestGolden`,
  `parser_test.go:44`, glob по `testdata/*.ast`; файл создаётся прогоном с
  `-update`, diff просмотреть глазами).
- `src/codegen/testdata/bool_all.c` (`-update`) и рукописный
  `src/codegen/testdata/bool_all.expected`; имя `"bool_all"` дописать в явный
  список `goldenExamples` (`codegen_test.go:48-64`) и, если примеру нужно
  больше одного скана, в `exampleScans` (`codegen_test.go:66-70`).
- `TestGCC` (`codegen_test.go:125`) подхватит новый пример сам:
  `gcc -std=c99 -Wall -Wextra` без предупреждений, запуск, сверка с `.expected`.

### `CLAUDE.md` (тем же коммитом)

- Статус и версия; в «Ограничениях языка» — BOOL как объявляемый тип, литералы
  `TRUE`/`FALSE`, логические операции и их приоритет, `&` как синоним `AND`,
  условие `IF` — любое BOOL-выражение (не только сравнение).
- Разделы по пакетам `lexer` / `ast` / `parser` / `sema` / `codegen`.
- Архитектурные решения — все шесть из этого файла.
- **Снять** долги: «объявить `x : BOOL;` пока нельзя» (в описании `TypeKind`),
  «условие `IF` обязано быть сравнением», формулировку про `a < b < c` поправить
  на новое сообщение.
- **Добавить** долги: нет конверсий `BOOL_TO_INT` / `INT_TO_BOOL`; `AND`/`OR`/
  `XOR` только булевы — побитовых операций над `INT` нет; драйвер печатает BOOL
  как `1`/`0`, а не `TRUE`/`FALSE`; `&` неотличим от `AND` в AST (долг
  форматтера); нет `MOD` и `**`.
- В «Дорожной карте» отметить закрытым пункт про `BOOL`.

### Уборка

**Удалить `PLAN_BOOL.md` и `BOOL_PROMPTS.md`** этим же коммитом: работа закрыта,
существенное переехало в `CLAUDE.md`.

---

## Верификация

После каждого этапа:

```bash
go build ./... && go vet ./...
go test ./...
```

После этапа 2 — регрессия разбора (эталоны не должны измениться):

```bash
go test ./src/parser && git diff --stat src/parser/testdata   # ожидается: пусто
```

После этапа 4 — регрессия генерации:

```bash
set -o pipefail
go test ./src/codegen -update && git diff --stat src/codegen/testdata   # ожидается: пусто
```

После этапа 5 — полная цепочка руками:

```bash
go run ./src -main -o /tmp/bool_all.c examples/bool_all.st
gcc -std=c99 -Wall -Wextra /tmp/bool_all.c -o /tmp/bool_all && /tmp/bool_all
diff <(/tmp/bool_all) src/codegen/testdata/bool_all.expected
```

Плюс негативные случаи — каждый обязан дать ошибку sema с `line:col`, а не
пройти в C:

```bash
printf 'PROGRAM P\nVAR i : INT; b : BOOL; END_VAR\nb := i;\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; b : BOOL; END_VAR\ni := b;\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; j : INT; END_VAR\nIF i AND j THEN i := 1; END_IF\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR b : BOOL; c : BOOL; END_VAR\nIF b < c THEN b := TRUE; END_IF\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR b : BOOL := 1; END_VAR\nb := b;\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; END_VAR\ni := NOT i;\nEND_PROGRAM\n' | go run ./src
```

И позитивный — что BOOL доезжает до C:

```bash
printf 'PROGRAM P\nVAR a : BOOL := TRUE; b : BOOL; n : INT := 5; END_VAR\nb := a AND (n > 3);\nIF NOT b THEN n := 0; END_IF\nEND_PROGRAM\n' | go run ./src -main | gcc -std=c99 -Wall -Wextra -x c - -o /tmp/p && /tmp/p
```

---

## Журнал сессий

Дописывается в конце каждой сессии — чтобы следующая видела не только «сделано»,
но и почему что-то отклонилось от плана.

| Дата | Этап | Коммит | Что сделано | Что всплыло сверх плана |
|------|------|--------|-------------|-------------------------|
| 2026-09-18 | 0 | — | План составлен, решения зафиксированы (набор операций с `&`, `BOOL` → `_Bool`, `XOR` → `!=`, нет адаптивного булева литерала, `=`/`<>` над BOOL разрешены, порядковые — нет), созданы `PLAN_BOOL.md` и `BOOL_PROMPTS.md` | Проверено: в `examples/` и эталонах нет идентификаторов `and`/`or`/`not`/`xor`/`true`/`false`/`bool` — превращение их в ключевые слова корпус не ломает |
