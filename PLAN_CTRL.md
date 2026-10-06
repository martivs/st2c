# План: операторы ELSIF / WHILE / REPEAT / EXIT (временный документ)

Рабочий план реализации управляющих конструкций `ELSIF`, `WHILE … DO`,
`REPEAT … UNTIL` и `EXIT` в st2c. **Временный**: по завершении этапа 4
удаляется вместе с `CTRL_PROMPTS.md`, существенное переезжает в `CLAUDE.md` —
так же, как это было сделано с `PLAN_BOOL.md` и `BOOL_PROMPTS.md` (коммит
0.1.21).

Этот файл — источник контекста для каждой новой сессии: этап читается отсюда,
результат отмечается здесь же, в том же коммите, что и код этапа.

**Один этап = одна сессия = один коммит.** Следующий этап не начинать без
явного «переходим дальше» от Mart.

---

## Зачем

Из управляющих конструкций сейчас есть только `IF / ELSE` и `FOR`. Следствия:

- выбор из нескольких вариантов пишется лесенкой вложенных
  `IF … ELSE IF … END_IF END_IF` — каждый уровень добавляет свой `END_IF`;
- цикл «пока условие истинно» не выражается вовсе: `FOR` требует заранее
  известного числа итераций;
- цикл «выполнить хотя бы раз, потом проверить» — тоже;
- досрочно выйти из цикла нельзя.

Это первый пункт раздела «Операторы» дорожной карты `CLAUDE.md` («`ELSIF /
WHILE / REPEAT..UNTIL / CASE / EXIT / RETURN` — встают в `parseStatement` и
`isBlockEnd` независимо друг от друга»). `CASE` и `RETURN` в эту работу не
входят.

Результат по завершении:

```
IF t < 10 THEN mode := 1;
ELSIF t < 20 THEN mode := 2;
ELSIF t < 30 THEN mode := 3;
ELSE mode := 4;
END_IF;

WHILE n > 0 DO
    n := n - 1;
    IF n = 3 THEN EXIT; END_IF;
END_WHILE;

REPEAT
    k := k + 1;
UNTIL k >= 5
END_REPEAT;
```

— разбирается, проверяется sema и превращается в C, который собирается
`gcc -std=c99 -Wall -Wextra` и работает.

**Объём работы меньше, чем у BOOL.** Слой типов и таблицы codegen не
трогаются вовсе: условия новых конструкций — обычные BOOL-выражения, и для
них уже есть всё, что сделано для условия `IF`. Каждая новая конструкция —
это близкий родственник существующей:

| Новая конструкция | Образец в коде | Что в C |
|---|---|---|
| `ELSIF` | ветка `ELSE` у `IF` | `} else if (…) {` |
| `WHILE c DO … END_WHILE` | `FOR` (тело) + `IF` (условие) | `while (c) { … }` |
| `REPEAT … UNTIL c END_REPEAT` | то же | `do { … } while (!(c));` |
| `EXIT;` | — (простой оператор, как `;`-вызов ФБ) | `break;` |

Оценка: ~250 строк кода + ~400 строк тестов + новый пример с эталонами.

---

## Принятые решения (не переоткрывать)

1. **`ELSIF` хранится в AST явным списком**, а не превращается во вложенные
   `IF` внутри `Else`. У `IfStatement` появляется поле
   `ElsIfs []*ElsIfClause`, где `ElsIfClause{Cond Expression, Then
   []Statement, Tok lexer.Token}`. Причина — решение «AST остаётся
   синтаксическим»: если склеить `ELSIF` во вложенный `IF`, дерево перестанет
   отличать `ELSIF` от `ELSE IF … END_IF` (а это разный текст программы, и
   будущий форматтер не сможет вернуть исходное написание). Вторая выгода —
   в C получается плоская цепочка `else if`, а не матрёшка из `else { if … }`.
   В `String()` ветки `ElsIf:` печатаются **только если список не пуст** —
   поэтому golden `.ast` всех существующих примеров не меняются.

2. **Условия `ELSIF`, `WHILE`, `UNTIL` — по тому же правилу, что `IF`**:
   любое выражение типа BOOL (сравнение, логическая связка, переменная,
   `NOT`, член ФБ, вызов функции с BOOL-возвратом). `INT`/`REAL` — ошибка.
   Проверка — одна функция `checkCondition(kw string, e ast.Expression)` в
   sema, ключевое слово попадает в сообщение: `WHILE condition must be BOOL,
   got INT`. Сообщение для `IF` остаётся побайтово прежним
   (`IF condition must be BOOL, got INT`) — прежние тесты не правятся.

3. **`WHILE c DO … END_WHILE` → `while (c) { … }`.** Условие вычисляется
   **перед каждой итерацией** заново — в отличие от границ `FOR`, которые по
   решению 6 вычисляются один раз до входа. Если условие ложно на входе, тело
   не выполняется ни разу.

4. **`REPEAT … UNTIL c END_REPEAT` → `do { … } while (!(c));`.** Тело
   выполняется **хотя бы один раз**, условие проверяется после тела. Смысл
   `UNTIL` — «повторять, **пока не** станет истинным», поэтому в C условие
   продолжения — отрицание: `!(c)`. После `UNTIL c` точка с запятой **не
   ставится** (по грамматике IEC: `REPEAT stmts UNTIL expr END_REPEAT`);
   `UNTIL c; END_REPEAT` — синтаксическая ошибка
   `expected END_REPEAT, got ;`.

5. **`EXIT;` → `break;`** — выход из **ближайшего** объемлющего цикла
   (`FOR`, `WHILE` или `REPEAT`). `EXIT` вне цикла — ошибка sema
   `EXIT outside of a loop`. `EXIT` внутри `IF` внутри цикла законен и
   выходит из цикла (у `if` в C нет своего `break`). С `FOR` `break` работает
   корректно: обёртка `{ … }` вокруг `for` (решение 6) — блок, а не цикл, и
   `break` уходит именно из `for`. Точка с запятой после `EXIT` обязательна,
   как после любого простого оператора.

6. **`;` после `END_WHILE` и `END_REPEAT` необязательна** — так же, как после
   `END_IF`/`END_FOR` (`optionalSemicolon`).

7. **Инвариант codegen для `EXIT`, и что из него следует для будущего `CASE`.**
   `goto` в сгенерированном C **не используется никогда**. Сейчас
   единственные конструкции C, которые «перехватывают» `break`, — циклы,
   полученные из циклов ST, поэтому `EXIT` — это всегда просто `break`. Если
   в будущем codegen начнёт эмитить конструкцию, которая тоже перехватывает
   `break` (C-`switch` для `CASE`), то `EXIT`, стоящий внутри неё, будет
   передаваться через **флаг выхода** `__exitN` (подробно — раздел «Почему
   `switch` и `EXIT` конфликтуют» ниже). В этой работе `switch` нет, флаг не
   реализуется; решение фиксируется сейчас, чтобы при работе над `CASE` не
   изобретать заново и не поддаться соблазну `goto`.

---

## Почему `switch` и `EXIT` конфликтуют (разбор для будущей работы по `CASE`)

Этот раздел не про текущие этапы — он объясняет решение 7, чтобы к нему можно
было вернуться при планировании `CASE`. При переносе в `CLAUDE.md` сжимается до
одного решения и пометки в дорожной карте.

### Что такое `break` в C

В C есть **одно** слово `break`, и оно означает «выйти из ближайшей
объемлющей конструкции, которая умеет ловить `break`». Таких конструкций
**четыре**: `for`, `while`, `do … while` — и `switch`. Обычный `if` и голый
блок `{ … }` `break` не ловят — он проходит сквозь них наружу.

В ST же слова разные: `EXIT` выходит из **цикла**, а ветка `CASE` своего
выхода не требует вовсе (она заканчивается сама, когда начинается следующая
метка).

### Где возникает проблема

Пока `CASE` нет, всё просто: `EXIT` → `break`, и ближайший ловец `break` —
всегда наш цикл.

Теперь представим, что `CASE` появился и генерируется через C-`switch`.
Программа на ST:

```
WHILE TRUE DO
    CASE cmd OF
        1: x := x + 1;
        2: EXIT;            (* хотим выйти из WHILE *)
    END_CASE;
END_WHILE;
```

Наивная трансляция:

```c
while (1) {
    switch (cmd) {
    case 1: x = x + 1; break;   /* этот break — конец ветки, так и задумано */
    case 2: break;              /* это EXIT... но ближайший ловец — switch! */
    }
}
```

Второй `break` выйдет **из `switch`**, а не из `while`. Цикл продолжится —
вместо выхода получится бесконечный цикл. Компилятор ничего не скажет: код
формально корректен, просто делает не то.

### Почему не `goto`

`goto` решил бы это одной строкой (`goto end_loop;`), но это прыжок в
произвольное место функции — код, который трудно читать и проверять. Решено:
`goto` в выходе транслятора не бывает (решение 7).

### Решение: флаг выхода

Перед `switch` заводится переменная-флаг, `EXIT` внутри `switch` ставит флаг
и выходит из `switch`, а сразу после `switch` флаг проверяется и, если он
поднят, делается второй `break` — уже из цикла:

```c
while (1) {
    {
        _Bool __exit3 = 0;
        switch (cmd) {
        case 1: x = x + 1; break;
        case 2: __exit3 = 1; break;     /* EXIT: поднять флаг, выйти из switch */
        }
        if (__exit3) break;             /* мы вне switch, ближайший ловец — while */
    }
}
```

Важно: флаг нужен **только** когда внутри `CASE` есть `EXIT`, который
относится к циклу **снаружи** `CASE`. Это видно из дерева на этапе генерации,
поэтому в обычных `CASE` (без `EXIT`) никакого флага не будет, C останется
чистым. `EXIT` внутри цикла, который сам лежит внутри ветки `CASE`, флага не
требует — его `break` и так попадает в свой (внутренний) цикл.

### Вторая, отдельная трудность `switch` — диапазоны меток

В ST метка `CASE` может быть диапазоном: `1..5: …`. В стандартном C99 у
`switch` диапазонов нет. Варианты (решаются в плане `CASE`, не здесь):
расширение GCC `case 1 ... 5:` (непереносимо: MSVC не поймёт), развёртка в
`case 1: case 2: … case 5:` (по стандарту, но `1..30000` даст 30000 меток),
либо гибрид. Списки меток `1, 3, 5:` проблемы не составляют — это просто
`case 1: case 3: case 5:`.

---

## Чек-лист этапов

Статус: `[ ]` не начат · `[~]` в работе · `[x]` готов (дописать номер коммита).
Обновляется в том же коммите, что и код этапа.

| Этап | Коммит | Содержание | Объём | Статус |
|------|--------|------------|-------|--------|
| 0 | 0.2.23 | план и промпты | — | [x] 0.2.23 |
| 1 | 0.2.24 | лексер + AST: 7 ключевых слов, узлы, `String()` | ~90 строк + ~50 тестов | [ ] |
| 2 | 0.2.25 | парсер: `ELSIF`, `WHILE`, `REPEAT`, `EXIT`, `isBlockEnd`; заглушка в codegen | ~70 строк + ~150 тестов | [ ] |
| 3 | 0.2.26 | sema (условия, `EXIT` вне цикла) + codegen (`else if`, `while`, `do-while`, `break`) | ~90 строк + ~200 тестов | [ ] |
| 4 | 0.2.27 | корпус `ctrl_all.st`, эталоны, `CLAUDE.md`, удаление плана | пример + документация | [ ] |

**Разбивка по слоям** (как у REAL и BOOL): каждая сессия проходит **один
слой** конвейера `lexer → parser → sema → codegen`, но сразу для **всех
четырёх** конструкций. Sema и codegen объединены в один этап: правка sema —
десятки строк, отдельная сессия под неё избыточна.

Каждый этап оставляет дерево зелёным (`go build ./...`, `go vet ./...`,
`go test ./...`) и идёт отдельным коммитом. Промежуточные состояния честные:
то, что ещё не реализовано, падает внятной ошибкой, а **неверного C не
возникает ни на одном этапе** (см. заглушку этапа 2).

---

## Этап 1 — лексер и AST

**Цель:** новые токены и узлы существуют. Парсер их ещё не знает, поэтому
`WHILE x DO` после этого этапа падает синтаксической ошибкой парсера
(`WHILE` перестал быть `IDENT`) — правильное промежуточное состояние.

### `src/lexer/lexer.go`

- Семь новых ключевых слов: `ELSIF`, `WHILE`, `END_WHILE`, `REPEAT`, `UNTIL`,
  `END_REPEAT`, `EXIT` — константы в enum `TokenType` (рядом с `IF … END_IF` и
  `FOR … END_FOR`, `lexer.go:52-59`) и строки в `keywords`
  (`lexer.go:99-112`). `DO` уже есть (от `FOR`), `WHILE` его переиспользует.
  В `tokenNames` руками **ничего не вписывать**: ключевые слова туда заливает
  `init()` (`lexer.go:86-90`). Станет 39 ключевых слов вместо 32.
- `readIdent` не трогать: `END_WHILE` целиком читается как идентификатор
  (подчёркивание — часть имени) и находится в `keywords` по
  `strings.ToUpper`.

**Ожидаемая правка старого теста** (не регрессия!): в `TestAllTokens`, кейс
«number rollbacks» (`lexer_test.go:178-190`), фрагмент `1EXIT` сейчас
ожидается как `{INT_LIT, "1"}, {IDENT, "EXIT"}`. Кейс проверяет, что
`readNumber` не съедает буквы после числа; после этапа `EXIT` — ключевое
слово, и ожидание становится `{INT_LIT, "1"}, {EXIT, "EXIT"}`. Суть проверки
(откат `readNumber`) не меняется. Это **единственный** прежний тест, который
правится в этой работе.

### `src/ast/ast.go`

Рядом с `IfStatement`/`ForStatement` (`ast.go:297-360`):

- `ElsIfClause{Cond Expression, Then []Statement, Tok lexer.Token}` — **не**
  `Statement` (маркерный метод не нужен, как и у `Arg`): это часть
  `IfStatement`, а не самостоятельный оператор. Метод `Line()` — для позиций.
- `IfStatement` — новое поле `ElsIfs []*ElsIfClause` между `Then` и `Else`.
  Комментарий узла: `IF Condition THEN Then {ELSIF Cond THEN Then}
  [ELSE Else] END_IF`.
- `WhileStatement{Cond Expression, Body []Statement, Tok}` — токен `WHILE`.
- `RepeatStatement{Body []Statement, Cond Expression, Tok}` — токен `REPEAT`;
  поля в порядке исходника (тело, потом условие).
- `ExitStatement{Tok}`.

Формат `String()` (это формат golden `.ast` — фиксируется здесь):

```
If                      While                Repeat           Exit
  Cond:                   Cond:                Body:
    ...                     ...                  ...
  Then:                   Body:                Until:
    ...                     ...                  ...
  ElsIf:
    Cond:
      ...
    Then:
      ...
  ElsIf:                <- каждая ветка ELSIF — своим блоком, в порядке исходника
    ...
  Else:
    ...
```

`ElsIf:` печатается **только для непустого списка** — иначе разъедутся все
golden `.ast` с `IF` (решение 1).

### Тесты этапа 1

`src/lexer/lexer_test.go`, табличный `TestAllTokens` (`lexer_test.go:44`):

- все семь слов в верхнем регистре и в смешанном (`elsif`, `While`,
  `end_repeat`, `Exit`); `TestKeywordsCaseInsensitive` (`lexer_test.go:344`)
  — по образцу;
- префиксы остаются `IDENT`: `WHILEX`, `EXITS`, `REPEATED`, `UNTIL1`,
  `ELSIFX`, `END_WHILEX`;
- `ELSE IF` — **два** токена `ELSE`, `IF` (не `ELSIF`), а `ELSIF` — один;
- правка кейса `1EXIT` (см. выше).

Тестов AST отдельно нет — `String()` новых узлов проверит парсер на этапе 2.

---

## Этап 2 — парсер

**Цель:** новый синтаксис разбирается в дерево; `-dump-ast` его показывает.

### `src/parser/parser.go`

- `parseStatement` (`parser.go:252-268`) — три новые ветки:
  `lexer.WHILE` → `parseWhile`, `lexer.REPEAT` → `parseRepeat`,
  `lexer.EXIT` → `parseExit`.
- `isBlockEnd` (`parser.go:241-249`) — добавить `ELSIF`, `END_WHILE`,
  `UNTIL`, `END_REPEAT`. Список операторов внутри тела заканчивается на любом
  из них; какой именно ожидался — проверяет `expect` в правиле конструкции.
  Комментарий у `parseStatements` (`parser.go:227`) поправить.
- `parseIf` (`parser.go:297-313`) — между `Then` и `ELSE` цикл:
  ```go
  for p.err == nil && p.curIs(lexer.ELSIF) {
      tok := p.expect(lexer.ELSIF)
      cond := p.parseExpression(lowestPrec)
      p.expect(lexer.THEN)
      stmt.ElsIfs = append(stmt.ElsIfs, &ast.ElsIfClause{Cond: cond, Then: p.parseStatements(), Tok: tok})
  }
  ```
  `ELSIF` после `ELSE` — ошибка сама собой: тело `ELSE` закончится на
  `ELSIF` (он в `isBlockEnd`), а дальше `expect(END_IF)` скажет
  `expected END_IF, got ELSIF`.
- `parseWhile`: `WHILE expr DO {stmt} END_WHILE ;?` — по образцу `parseFor`.
- `parseRepeat`: `REPEAT {stmt} UNTIL expr END_REPEAT ;?` — после `UNTIL`
  выражение, затем **сразу** `expect(END_REPEAT)` (решение 4: `;` перед
  `END_REPEAT` — ошибка).
- `parseExit`: `EXIT ;` — `;` обязательна.
- Комментарий у `optionalSemicolon` (`parser.go:344`) — дописать
  `END_WHILE`/`END_REPEAT`.

### Заглушка в codegen (обязательно на этом же этапе)

Без неё после этапа 2 появляется **молча неверный C**: sema не смотрит в
новые поля, а ветка `*ast.IfStatement` в `codegen.go:686-706` про `ElsIfs`
не знает и просто **выбросит** ветки `ELSIF`. Поэтому в начале этой ветки —
временная защита:

```go
if len(s.ElsIfs) > 0 {
    return fmt.Errorf("line %d: codegen: ELSIF is not supported yet", s.Line())
}
```

`WHILE`/`REPEAT`/`EXIT` такой заглушки не требуют: это новые типы узлов, и
`default` в `stmt` (`codegen.go:714`) уже даёт `unsupported statement`. Этап 3
заглушку удаляет.

### Тесты этапа 2

`src/parser/parser_test.go`:

- Структурные (по образцу `TestForBy`, `parser_test.go:445`, — новая
  табличная функция `TestControl`): `IF` с одним `ELSIF`; с тремя `ELSIF` и
  `ELSE`; с `ELSIF` без `ELSE`; **`ELSE IF … END_IF END_IF` даёт вложенный
  `If` внутри `Else`, а не `ElsIf`** (решение 1 — дерево различает);
  `WHILE` с пустым телом; `REPEAT` с телом из нескольких операторов;
  `EXIT;` внутри `IF` внутри `WHILE`; вложенные циклы трёх видов; `;` после
  `END_WHILE`/`END_REPEAT` есть и нет; регистр (`while … end_while`).
- Негативные в `TestParseErrors` (`parser_test.go:751`):

  | Вход | Подстрока ошибки |
  |---|---|
  | `IF a THEN x := 1; ELSE x := 2; ELSIF b THEN x := 3; END_IF` | `expected END_IF, got ELSIF` |
  | `IF a THEN x := 1; ELSIF THEN x := 2; END_IF` | `expected expression, got THEN` |
  | `IF a THEN x := 1; ELSIF b x := 2; END_IF` | `expected THEN` |
  | `WHILE a x := 1; END_WHILE` | `expected DO` |
  | `WHILE a DO x := 1; END_FOR` | `expected END_WHILE, got END_FOR` |
  | `REPEAT x := 1; END_REPEAT` | `expected UNTIL, got END_REPEAT` |
  | `REPEAT x := 1; UNTIL END_REPEAT` | `expected expression, got END_REPEAT` |
  | `REPEAT x := 1; UNTIL a; END_REPEAT` | `expected END_REPEAT, got ;` |
  | `EXIT` без `;` (перед `END_PROGRAM`) | `expected ;` |
  | `UNTIL a` в теле программы | `expected END_PROGRAM, got UNTIL` |
  | `x := WHILE;` | `expected expression, got WHILE` |

  Точные формулировки сверить с `expect`/`fail`; в таблице — смысл.

**Обязательная проверка этапа:** `go test ./src/parser` **без** `-update` —
девять golden `.ast` совпадают побайтово (`git diff --stat
src/parser/testdata` пуст). Если разъехалось — `ElsIf:` печатается при пустом
списке.

---

## Этап 3 — sema и codegen

### `src/sema/sema.go`

- **Хелпер условия** `checkCondition(kw string, e ast.Expression)`:
  `typeOf(e, Bool)`, при типе не `Invalid` и не `Bool` —
  `"%s condition must be BOOL, got %s"` на `exprTok(e)`. Ветку
  `*ast.IfStatement` (`sema.go:320`) перевести на него с `kw = "IF"` —
  сообщение побайтово прежнее. Комментарий про решение 6 REAL переезжает к
  хелперу.
- `IfStatement`: после `Then` — для каждой `ElsIfClause`
  `checkCondition("ELSIF", cl.Cond)` и `checkStatements(cl.Then)`; порядок —
  как в исходнике (ошибки sema копятся в порядке следования, тесты на это
  опираются).
- `WhileStatement`: `checkCondition("WHILE", …)`, затем тело.
- `RepeatStatement`: **сначала тело, потом** `checkCondition("UNTIL", …)` —
  порядок исходника.
- **Счётчик циклов** — поле `loops int` в `checker` (`sema.go:127-139`):
  `FOR`/`WHILE`/`REPEAT` делают `c.loops++` перед обходом тела и `c.loops--`
  после. `ExitStatement` при `c.loops == 0` →
  `EXIT outside of a loop` на токене `EXIT`. Сбрасывать между POU не нужно:
  счётчик возвращается в 0 после каждого цикла, а тела POU не вложены. В
  `ELSIF`/`IF` счётчик не трогается — `EXIT` в `IF` внутри цикла законен.
- В `TestInfo`-смысле ничего нового: условия выводятся через `typeOf`, их
  типы и адаптивные литералы (`WHILE r < 10` с REAL `r` → `10` как REAL)
  попадают в side-table автоматически.

### `src/codegen/codegen.go`

В `stmt` (`codegen.go:672-717`):

- `IfStatement`: **удалить заглушку этапа 2**; после тела `Then` для каждой
  ветки — `} else if (<cond>) {`, тело, затем прежний `else`. Условие — через
  `g.expr`, как у `if`.
- `WhileStatement` → `while (<cond>) {` / тело / `}`.
- `RepeatStatement` → `do {` / тело / `} while (!<cond>);`. Внимание к скобкам:
  `g.expr` у бинарной операции уже возвращает `(a > b)`, а у переменной —
  `self->b` без скобок. Писать всегда `!(%s)` — получится `!((a > b))` в
  первом случае и `!(self->b)` во втором; двойные скобки `-Wparentheses` не
  смущают. Это согласуется с правилом «скобки надёжнее читаемости».
- `ExitStatement` → `break;`. Плюс внутренняя защита по образцу прочих
  «sema must reject this»: счётчик `loops` в `gen`, `EXIT` при нуле —
  ошибка `codegen: internal: EXIT outside of a loop (sema must reject
  this)`. Без неё обход sema (`skipSema`) дал бы C, который `gcc` отвергнет
  с невнятным `break statement not within loop or switch`.
- Комментарий у `stmt` или у ветки `EXIT` — **инвариант решения 7**: `goto`
  не эмитится; `switch` и служебные циклы-обёртки не эмитятся; если появятся
  — `EXIT` внутри них через флаг `__exitN`.
- `reservedCNames` трогать не нужно: `while`, `do`, `break` — ключевые слова
  C99, они там уже есть.

### Тесты этапа 3

**Sema** — новая табличная функция `TestControl` в `src/sema/sema_test.go`
(формат кейса — `src` + `want []string`, как в `TestBool`,
`sema_test.go:969`).

Позитив (ошибок быть не должно): цепочка `ELSIF` с BOOL-условиями разных
видов (сравнение, переменная, `NOT`, связка, член ФБ, вызов функции);
`WHILE` по BOOL-переменной и по сравнению; `WHILE r < 10.0` и `WHILE r < 10`
с REAL `r` (адаптивный литерал); `REPEAT … UNTIL`; `EXIT` в каждом из трёх
циклов; `EXIT` внутри `IF`/`ELSIF` внутри цикла; `EXIT` во вложенном цикле;
`EXIT` в цикле внутри `FUNCTION` и внутри тела ФБ.

Негатив:

| Вход | Ожидаемая ошибка |
|---|---|
| `ELSIF i THEN` (INT) | `ELSIF condition must be BOOL, got INT` |
| `WHILE i DO` | `WHILE condition must be BOOL, got INT` |
| `UNTIL r` (REAL) | `UNTIL condition must be BOOL, got REAL` |
| `EXIT;` на верхнем уровне PROGRAM | `EXIT outside of a loop` |
| `EXIT;` в `IF` вне цикла | `EXIT outside of a loop` |
| `EXIT;` в FUNCTION вне цикла | то же |
| `EXIT;` после `END_WHILE` (цикл уже закрыт) | то же — счётчик вернулся в 0 |
| необъявленное имя в теле `ELSIF` / `WHILE` / `REPEAT` | `undeclared` — тела обходятся |
| `WHILE undeclared DO` | только `undeclared`, без каскада `must be BOOL` |
| ошибка в теле `REPEAT` и в `UNTIL` | обе, тело раньше условия |

Прежние тесты sema не меняются (сообщение `IF` побайтово то же).

**Codegen:**

- Регрессия: golden `.c` всех тринадцати существующих примеров побайтово
  прежние:
  ```bash
  set -o pipefail
  go test ./src/codegen -update && git diff --stat src/codegen/testdata   # ожидается: пусто
  ```
  Ловушка `-update` через пайп — см. `CLAUDE.md`, раздел «Тесты»;
  восстанавливать `git checkout -- src/codegen/testdata`.
- `TestNameErrors` (`codegen_test.go:169`) — кейс со `skipSema`: `EXIT;` вне
  цикла → подстрока `EXIT outside of a loop`.
- Позитив руками — раздел «Верификация» ниже. Полноценный корпус — этап 4.

---

## Этап 4 — корпус, эталоны, документация

### `examples/ctrl_all.st`

Новый пример (единый корпус — только `examples/`, решение 9). Драйвер — 3
скана (в `exampleScans`), чтобы было видно состояние ФБ с циклом. Значения
`.expected` посчитать вручную, расчёт — в комментариях `.st` (как в
`bool_all.st`). Что должно быть покрыто — и как пример ловит ошибки
реализации:

- **Цепочка `ELSIF` выбирает первую истинную ветку.** Условия
  пересекаются (`t < 10`, `t < 20`, `t < 30`), при `t = 5` истинны все три —
  верный ответ только у первой; если codegen перепутает порядок или склеит
  ветки, число будет другим. Плюс цепочка **без** `ELSE`, где ни одна ветка
  не истинна (переменная не меняется), и вложенный `IF` внутри `ELSIF`.
- **`WHILE` с ложным условием на входе — ноль итераций, `REPEAT` с истинным
  `UNTIL` на входе — ровно одна.** Эта пара сразу ловит перепутанные
  `while`/`do-while` и забытое отрицание `!(…)` в `REPEAT`.
- **Обычные `WHILE` и `REPEAT`** с несколькими итерациями (счётчик,
  накопление суммы).
- **`EXIT`** из каждого из трёх видов циклов; **`EXIT` из `FOR`** — значение
  переменной цикла после выхода (она равна значению той итерации, где
  сработал `EXIT`: копия счётчика делается первой строкой тела);
  **`EXIT` из вложенного цикла выходит только из внутреннего** — внешний
  доходит до конца (счётчик внешних итераций полный); **`WHILE TRUE … EXIT`**
  — идиома бесконечного цикла с выходом.
- **`FOR` внутри `WHILE`** — временные `__iN` живут внутри тела `while`.
- **`WHILE` в теле ФБ за три скана** — состояние между сканами; **`REPEAT` в
  `FUNCTION`**.
- **Условия разных типов**: BOOL-флаг, `NOT`, связка сравнений, REAL с
  адаптивным литералом.

Важно: все циклы обязаны завершаться — бесконечный цикл здесь не даёт
неверное число, а **вешает** `TestGCC` до таймаута 10 с.

### Эталоны и тесты

- `src/parser/testdata/ctrl_all.ast` — **сначала создать пустым файлом**
  (glob по `testdata/*.ast`, иначе `-update` его не увидит), затем
  `go test ./src/parser -update`, diff просмотреть глазами.
- `src/codegen/testdata/ctrl_all.c` (`-update`) и рукописный
  `ctrl_all.expected`; `"ctrl_all"` дописать в `goldenExamples`
  (`codegen_test.go:51`) и `exampleScans` (`codegen_test.go:70`, 3 скана).
- `TestGCC` и `TestExamplesClean` подхватят пример сами.

### `CLAUDE.md` (тем же коммитом)

- Шапка: абзац про эту работу (план `PLAN_CTRL.md` + `CTRL_PROMPTS.md`, четыре
  этапа, коммиты 0.2.24–0.2.27, оба файла удалены коммитом 0.2.27).
- Заголовок статуса и абзац о конвейере.
- Список примеров: четырнадцать, `ctrl_all` с ожидаемыми значениями.
- «Ограничения языка → Конструкции»: `IF / ELSIF / ELSE`, `WHILE`, `REPEAT`,
  `EXIT`; условие `IF`/`ELSIF`/`WHILE`/`UNTIL` — любое BOOL-выражение.
- Разделы по пакетам: число ключевых слов в лексере (39); новые узлы AST и
  формат `String()`; правила парсера и `isBlockEnd`; `checkCondition` и
  счётчик циклов в sema; новые ветки codegen.
- «Тесты»: `TestControl` в парсере и sema, число эталонов (`.ast` — десять),
  корпус — четырнадцать примеров.
- «Архитектурные решения» — решения 1–7 этого файла, сжато; решение 7 —
  инвариант «`goto` нет, `EXIT` → `break`, через `switch` — флагом».
- «Дорожная карта»: из «Операторов» убрать `ELSIF / WHILE / REPEAT / EXIT`,
  оставить `CASE / RETURN`; к `CASE` — пометка: «C-`switch` + флаг выхода
  `__exitN` для `EXIT` внутри (решение 7); способ для диапазонов меток `1..5`
  — решается в плане `CASE`».
- «Известные долги» — добавить: бесконечный цикл внутри скана (`WHILE TRUE`
  без `EXIT`, условие, которое никогда не станет ложным) не диагностируется,
  сторожевого таймера (watchdog) нет — программа зависнет; `UNTIL c;` с
  точкой с запятой (диалект CODESYS) не принимается.

### Уборка

**Удалить `PLAN_CTRL.md` и `CTRL_PROMPTS.md`** этим же коммитом.

---

## Верификация

После каждого этапа:

```bash
go build ./... && go vet ./...
go test ./...
```

После этапа 2 — разбор существующих программ не изменился, новые видны:

```bash
go test ./src/parser && git diff --stat src/parser/testdata   # ожидается: пусто
printf 'PROGRAM P\nVAR n : INT := 3; END_VAR\nWHILE n > 0 DO n := n - 1; END_WHILE\nEND_PROGRAM\n' | go run ./src -dump-ast
printf 'PROGRAM P\nVAR n : INT; END_VAR\nIF n < 1 THEN n := 1; ELSIF n < 2 THEN n := 2; END_IF\nEND_PROGRAM\n' | go run ./src   # ошибка: ELSIF is not supported yet
```

После этапа 3 — регрессия генерации и позитив через gcc:

```bash
set -o pipefail
go test ./src/codegen -update && git diff --stat src/codegen/testdata   # ожидается: пусто

printf 'PROGRAM P\nVAR n : INT := 10; k : INT; m : INT; END_VAR\nWHILE n > 0 DO n := n - 1; IF n = 4 THEN EXIT; END_IF; END_WHILE\nREPEAT k := k + 1; UNTIL k >= 3 END_REPEAT\nIF k < 2 THEN m := 1; ELSIF k < 5 THEN m := 2; ELSE m := 3; END_IF\nEND_PROGRAM\n' \
  | go run ./src -main | gcc -std=c99 -Wall -Wextra -x c - -o /tmp/p && /tmp/p
# ожидается: n=4 k=3 m=2
```

Негативы — каждый обязан дать ошибку с `line:col`, а не пройти в C:

```bash
printf 'PROGRAM P\nVAR i : INT; END_VAR\nWHILE i DO i := 0; END_WHILE\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; END_VAR\nREPEAT i := i + 1; UNTIL i END_REPEAT\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; END_VAR\nIF i > 0 THEN i := 1; ELSIF i THEN i := 2; END_IF\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; END_VAR\nEXIT;\nEND_PROGRAM\n' | go run ./src
printf 'PROGRAM P\nVAR i : INT; END_VAR\nREPEAT i := 1; UNTIL i > 0; END_REPEAT\nEND_PROGRAM\n' | go run ./src
```

После этапа 4 — полная цепочка нового примера руками:

```bash
go run ./src -main -scans 3 -o /tmp/ctrl_all.c examples/ctrl_all.st
gcc -std=c99 -Wall -Wextra /tmp/ctrl_all.c -o /tmp/ctrl_all && /tmp/ctrl_all
diff <(/tmp/ctrl_all) src/codegen/testdata/ctrl_all.expected
```

---

## Журнал сессий

Дописывается в конце каждой сессии — чтобы следующая видела не только
«сделано», но и почему что-то отклонилось от плана.

| Дата | Этап | Коммит | Что сделано | Что всплыло сверх плана |
|------|------|--------|-------------|-------------------------|
| 2026-10-06 | 0 | 0.2.23 | План составлен, семь решений зафиксированы (`ELSIF` явным списком в AST, общий `checkCondition`, `WHILE` → `while`, `REPEAT` → `do … while (!(c))`, `EXIT` → `break` с проверкой «вне цикла», необязательная `;` после `END_*`, инвариант «без `goto`, через `switch` — флагом»); созданы `PLAN_CTRL.md` и `CTRL_PROMPTS.md` | `EXIT` добавлен к исходным трём конструкциям: без него `WHILE`/`REPEAT` неполноценны. Обсуждение `CASE` → `switch` и конфликта с `EXIT` записано отдельным разделом. Проверено: в `examples/` и эталонах нет идентификаторов `elsif`/`while`/`repeat`/`until`/`exit` — превращение их в ключевые слова корпус не ломает; единственный затронутый тест — `1EXIT` в лексере. Найдена ловушка этапа 2: без заглушки codegen молча выбросил бы ветки `ELSIF` |
