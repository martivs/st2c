// Package sema — семантический анализ между парсером и codegen: проверки,
// без которых сгенерированный C был бы молча некорректен.
//
// Таблица имён двухуровневая: глобальная таблица POU (имя → узел, строится
// первым проходом) и локальная таблица переменных — своя на каждый POU.
//
// Слой типов (этап 2 REAL): тип — сущность (TypeKind), типы выражений
// выводятся снизу вверх (typeOf) и записываются в side-table Info.Types,
// которую забирает codegen. Смешение INT и REAL запрещено — только явные
// конверсии INT_TO_REAL / REAL_TO_INT (встроенные функции). Целый литерал
// адаптивен: в контексте REAL он становится REAL (`r := 1 / 2;` —
// вещественное деление). Условие IF обязано быть BOOL (результатом
// сравнения), FOR — только по INT.
//
// Экземпляры ФБ (этап 6): объявление `inst : Counter;` резолвится в
// объявленный FUNCTION_BLOCK, нерезолвившийся тип — ошибка. Снаружи доступен
// только интерфейс экземпляра: вход — только запись, выход — только чтение.
// Вызов ФБ — только оператором и только с именованными аргументами; аргументы
// необязательны (непереданный вход хранит значение с прошлого вызова).
// Циклическая вложенность экземпляров запрещена — struct в C был бы
// бесконечного размера.
package sema

import (
	"fmt"
	"math"
	"strings"

	"st2c/src/ast"
	"st2c/src/lexer"
)

// Пределы IEC-типа INT: 16 бит со знаком.
const (
	intMin = -32768
	intMax = 32767
)

// TypeKind — тип как сущность sema. Invalid и Unknown — служебные:
// Invalid означает «тип не выведен, ошибка уже выдана» и гасит каскад
// повторных ошибок выше по дереву; Unknown возвращает hintOf, когда
// контекст типа не задаёт (выражение из одних целых литералов).
type TypeKind int

const (
	Invalid TypeKind = iota
	Unknown
	Int
	Real
	Bool   // результат сравнения; объявить `x : BOOL;` пока нельзя
	FBInst // экземпляр ФБ — не значение и не lvalue
)

var typeNames = map[TypeKind]string{
	Invalid: "<invalid>", Unknown: "<unknown>",
	Int: "INT", Real: "REAL", Bool: "BOOL", FBInst: "FUNCTION_BLOCK",
}

func (k TypeKind) String() string {
	if s, ok := typeNames[k]; ok {
		return s
	}
	return "UNKNOWN"
}

// builtinType — единственная точка «встроенные скаляры»: имя типа из
// объявления → TypeKind. Новый скаляр — строка здесь (и в cType codegen).
func builtinType(name string) (TypeKind, bool) {
	switch strings.ToUpper(name) {
	case "INT":
		return Int, true
	case "REAL":
		return Real, true
	}
	return Invalid, false
}

// builtin — встроенная функция-конверсия: позиционные параметры и тип
// результата. Ключ таблицы — имя в верхнем регистре.
type builtin struct {
	params []TypeKind
	result TypeKind
}

var builtins = map[string]builtin{
	"INT_TO_REAL": {params: []TypeKind{Int}, result: Real},
	"REAL_TO_INT": {params: []TypeKind{Real}, result: Int},
}

// Info — результат sema для codegen (side-table, AST остаётся
// синтаксическим): тип каждого выражения (у адаптивного целого литерала —
// принятый тип контекста) и какие встроенные конверсии реально вызваны
// (хелпер округления эмитится в прологе C-файла только при надобности).
type Info struct {
	Types        map[ast.Expression]TypeKind
	UsedBuiltins map[string]bool // ключ — имя в верхнем регистре
}

// Symbol — объявленная переменная в символьной таблице.
type Symbol struct {
	Name     string      // оригинальное написание из объявления
	TypeName string      // имя типа как записано в объявлении (для сообщений)
	Type     TypeKind    // отрезолвленный тип; Invalid — ошибка уже выдана
	Tok      lexer.Token // позиция объявления
	// FB — узел FUNCTION_BLOCK, если переменная — экземпляр ФБ (тип
	// объявления отрезолвился в ФБ); nil для обычных переменных.
	FB *ast.FunctionBlock
}

// callEdge — ребро графа вызовов «функция → функция» для запрета рекурсии.
type callEdge struct {
	callee string      // ключ ToUpper вызываемой функции
	name   string      // оригинальное написание в месте вызова
	tok    lexer.Token // позиция вызова
}

// checker — состояние прохода по одному POU. В отличие от fail-fast парсера
// ошибки копятся: пользователь видит все проблемы за один запуск.
type checker struct {
	// pous — глобальная таблица POU файла; ключ — strings.ToUpper(имя),
	// идентификаторы в IEC 61131-3 регистронезависимы.
	pous map[string]ast.POU
	// syms — локальная символьная таблица текущего POU; ключ — ToUpper(имя).
	syms map[string]*Symbol
	info *Info // общий на весь файл
	errs []error
	// curFunc — ключ функции, чьё тело обходится ("" вне функций); только
	// функции попадают в граф вызовов: PROGRAM и ФБ из выражений не вызываются.
	curFunc string
	graph   map[string][]callEdge
}

// Check проверяет дерево и возвращает side-table типов и все найденные
// семантические ошибки (nil — ошибок нет). При ошибках Info неполна и
// codegen запускать нельзя.
func Check(sf *ast.SourceFile) (*Info, []error) {
	var errs []error
	info := &Info{Types: map[ast.Expression]TypeKind{}, UsedBuiltins: map[string]bool{}}

	// Первый проход — глобальная таблица POU: тела могут вызывать функции,
	// объявленные ниже по файлу.
	pous := map[string]ast.POU{}
	for _, pou := range sf.POUs {
		name, tok := pouName(pou)
		key := strings.ToUpper(name)
		if _, reserved := builtins[key]; reserved {
			errs = append(errs, fmt.Errorf("line %d:%d: %q is a reserved name (built-in function)",
				tok.Line, tok.Col, name))
			continue
		}
		if prev, ok := pous[key]; ok {
			prevName, prevTok := pouName(prev)
			errs = append(errs, fmt.Errorf("line %d:%d: duplicate POU %q (first declared as %q at line %d)",
				tok.Line, tok.Col, name, prevName, prevTok.Line))
			continue
		}
		pous[key] = pou
	}

	// Второй проход — тела в порядке файла, свой scope на каждый POU.
	graph := map[string][]callEdge{}
	for _, pou := range sf.POUs {
		c := &checker{pous: pous, syms: map[string]*Symbol{}, info: info, graph: graph}
		switch p := pou.(type) {
		case *ast.Program:
			c.checkBody(p.VarBlocks, p.Body)
		case *ast.Function:
			c.checkFunction(p)
		case *ast.FunctionBlock:
			c.checkBody(p.VarBlocks, p.Body)
		}
		errs = append(errs, c.errs...)
	}

	errs = append(errs, checkRecursion(sf, graph)...)
	errs = append(errs, checkInstanceCycles(sf, pous)...)
	return info, errs
}

// resolveType резолвит имя типа из объявления (переменной или возврата
// функции): встроенные скаляры — через builtinType, остальные имена ищутся
// в глобальной таблице POU. Найденный FUNCTION_BLOCK делает переменную
// экземпляром ФБ (FBInst + узел); другой POU или неизвестное имя — ошибка
// на tok, тип Invalid.
func (c *checker) resolveType(name string, tok lexer.Token) (TypeKind, *ast.FunctionBlock) {
	if k, ok := builtinType(name); ok {
		return k, nil
	}
	pou, ok := c.pous[strings.ToUpper(name)]
	if !ok {
		c.errorf(tok, "unknown type %q", name)
		return Invalid, nil
	}
	fb, isFB := pou.(*ast.FunctionBlock)
	if !isFB {
		c.errorf(tok, "%q is not a type", name)
		return Invalid, nil
	}
	return FBInst, fb
}

// kindOf — молчаливый резолв имени типа чужого объявления (вход функции,
// член ФБ, тип возврата вызываемой функции): ошибку о нерезолвящемся типе
// выдаёт проверка того POU, где он объявлен, здесь достаточно Invalid.
func (c *checker) kindOf(name string) TypeKind {
	if k, ok := builtinType(name); ok {
		return k
	}
	if _, isFB := c.pous[strings.ToUpper(name)].(*ast.FunctionBlock); isFB {
		return FBInst
	}
	return Invalid
}

// scalarOnly сводит тип к подсказке для hintOf: значимы только скаляры.
func scalarOnly(k TypeKind) TypeKind {
	if k == Int || k == Real {
		return k
	}
	return Unknown
}

// pouName возвращает имя POU и токен-якорь для сообщений об ошибках.
func pouName(pou ast.POU) (string, lexer.Token) {
	switch p := pou.(type) {
	case *ast.Program:
		return p.Name, p.Tok
	case *ast.Function:
		return p.Name, p.Tok
	case *ast.FunctionBlock:
		return p.Name, p.Tok
	}
	return "", lexer.Token{}
}

func (c *checker) errorf(tok lexer.Token, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	c.errs = append(c.errs, fmt.Errorf("line %d:%d: %s", tok.Line, tok.Col, msg))
}

// checkFunction: имя функции живёт в её локальной таблице как переменная
// возврата с типом ReturnType — по IEC возврат записывается присваиванием
// имени функции (`Add := x + y;`). Параметр или переменная с тем же именем
// даст обычную ошибку дубликата. ReturnType резолвится как тип объявления;
// ФБ возвращать нельзя (экземпляр — не значение).
func (c *checker) checkFunction(f *ast.Function) {
	key := strings.ToUpper(f.Name)
	kind, fb := c.resolveType(f.ReturnType, f.Tok)
	if fb != nil {
		c.errorf(f.Tok, "function %q cannot return a function block (%q)", f.Name, f.ReturnType)
		kind = Invalid
	}
	c.syms[key] = &Symbol{Name: f.Name, TypeName: f.ReturnType, Type: kind, Tok: f.Tok}
	c.curFunc = key
	c.checkBody(f.VarBlocks, f.Body)
}

// checkBody — общий обход POU: объявления, инициализаторы, операторы.
func (c *checker) checkBody(blocks []*ast.VarBlock, body []ast.Statement) {
	// Первый проход — собрать все объявления: инициализаторы проверяются
	// отдельным проходом, чтобы порядок объявлений не влиял на объявленность.
	for _, blk := range blocks {
		for _, d := range blk.Decls {
			kind, fb := c.resolveType(d.TypeName, d.Tok)
			if fb != nil && d.Init != nil {
				c.errorf(d.Tok, "function block instance cannot have an initializer")
			}
			for _, name := range d.Names {
				key := strings.ToUpper(name.Name)
				if prev, ok := c.syms[key]; ok {
					c.errorf(name.Tok, "duplicate declaration of %q (first declared as %q at line %d)",
						name.Name, prev.Name, prev.Tok.Line)
					continue
				}
				c.syms[key] = &Symbol{Name: name.Name, TypeName: d.TypeName, Type: kind, Tok: name.Tok, FB: fb}
			}
		}
	}
	for _, blk := range blocks {
		for _, d := range blk.Decls {
			if d.Init == nil {
				continue
			}
			want := c.kindOf(d.TypeName)
			got := c.typeOf(d.Init, want)
			if want != FBInst { // у экземпляра инициализатор уже отвергнут выше
				c.checkAssign(d.Tok, d.Names[0].Name, want, got)
			}
		}
	}
	c.checkStatements(body)
}

func (c *checker) checkStatements(stmts []ast.Statement) {
	for _, s := range stmts {
		c.checkStatement(s)
	}
}

func (c *checker) checkStatement(s ast.Statement) {
	switch st := s.(type) {
	case *ast.AssignStatement:
		want := c.checkWrite(st.Target)
		got := c.typeOf(st.Value, want)
		c.checkAssign(st.Tok, targetName(st.Target), want, got)
	case *ast.IfStatement:
		// Условие — только BOOL, т.е. результат сравнения (решение 6 плана
		// REAL): `IF x THEN` с INT/REAL — ошибка.
		if t := c.typeOf(st.Condition, Bool); t != Invalid && t != Bool {
			c.errorf(exprTok(st.Condition), "IF condition must be BOOL (a comparison), got %s", t)
		}
		c.checkStatements(st.Then)
		c.checkStatements(st.Else)
	case *ast.ForStatement:
		// FOR остаётся целочисленным (решение 5 плана REAL): у REAL нет
		// широкого счётчика, а накопление ошибки шага делает число итераций
		// непредсказуемым.
		if t := c.checkWrite(st.Var); t != Invalid && t != Int {
			c.errorf(st.Var.Tok, "FOR variable %q must be INT, got %s", st.Var.Name, t)
		}
		c.checkForBound(st.Start, "start")
		c.checkForBound(st.End, "end")
		if st.Step != nil {
			c.checkForBound(st.Step, "step")
		}
		c.checkStatements(st.Body)
	case *ast.CallStatement:
		c.checkFBCall(st.Call)
	}
}

func (c *checker) checkForBound(e ast.Expression, what string) {
	if t := c.typeOf(e, Int); t != Invalid && t != Int {
		c.errorf(exprTok(e), "FOR %s must be INT, got %s", what, t)
	}
}

// checkAssign — совместимость при записи: тип значения обязан совпадать с
// типом цели (BOOL в цель нельзя — переменных BOOL нет). Invalid с любой
// стороны — ошибка уже выдана, каскада не нужно.
func (c *checker) checkAssign(tok lexer.Token, name string, want, got TypeKind) {
	if want == Invalid || got == Invalid || want == got {
		return
	}
	c.errorf(tok, "cannot assign %s to %q of type %s%s", got, name, want, convHint(want, got))
}

// convHint — подсказка про явные конверсии при смешении INT и REAL.
func convHint(a, b TypeKind) string {
	if (a == Int && b == Real) || (a == Real && b == Int) {
		return " (use INT_TO_REAL / REAL_TO_INT)"
	}
	return ""
}

// targetName — текст цели записи для сообщений: `x` или `inst.member`.
func targetName(e ast.Expression) string {
	switch t := e.(type) {
	case *ast.Identifier:
		return t.Name
	case *ast.MemberExpr:
		return targetName(t.Base) + "." + t.Member
	}
	return "<expression>"
}

// exprTok — токен начала выражения (позиция для сообщений об условии IF
// и границах FOR; у узлов-операций Tok — сам оператор, что для «всего
// выражения» менее наглядно).
func exprTok(e ast.Expression) lexer.Token {
	switch ex := e.(type) {
	case *ast.Identifier:
		return ex.Tok
	case *ast.IntLiteral:
		return ex.Tok
	case *ast.RealLiteral:
		return ex.Tok
	case *ast.UnaryExpr:
		return ex.Tok
	case *ast.BinaryExpr:
		return exprTok(ex.Left)
	case *ast.MemberExpr:
		return exprTok(ex.Base)
	case *ast.CallExpr:
		return exprTok(ex.Callee)
	}
	return lexer.Token{}
}

// checkWrite проверяет цель записи (цель присваивания, переменную FOR,
// цель `name => target`) и возвращает её тип (Invalid — ошибка уже выдана).
// Экземпляр ФБ целиком — не lvalue.
func (c *checker) checkWrite(target ast.Expression) TypeKind {
	switch t := target.(type) {
	case *ast.Identifier:
		sym := c.lookup(t)
		if sym == nil {
			return Invalid
		}
		if sym.FB != nil {
			c.errorf(t.Tok, "cannot assign to function block instance %q", t.Name)
			return Invalid
		}
		return sym.Type
	case *ast.MemberExpr:
		return c.resolveMember(t, true)
	default:
		c.typeOf(target, Unknown)
	}
	return Invalid
}

// typeOf выводит тип выражения снизу вверх и записывает его в side-table.
// want — тип контекста: в контексте REAL целый литерал становится REAL
// (адаптивный литерал, решение 2 плана REAL); Invalid — контекст уже
// ошибочен, диапазон литералов не проверяется (нет каскада). Совместимость
// с want здесь не проверяется — это делает вызывающий, знающий контекст
// (присваивание, аргумент, условие).
func (c *checker) typeOf(e ast.Expression, want TypeKind) TypeKind {
	t := c.infer(e, want)
	c.info.Types[e] = t
	return t
}

func (c *checker) infer(e ast.Expression, want TypeKind) TypeKind {
	switch ex := e.(type) {
	case *ast.Identifier:
		sym := c.lookup(ex)
		if sym == nil {
			return Invalid
		}
		if sym.FB != nil {
			c.errorf(ex.Tok, "function block instance %q cannot be used as a value", ex.Name)
			return Invalid
		}
		return sym.Type
	case *ast.IntLiteral:
		if want == Real {
			return Real
		}
		c.checkIntRange(int64(ex.Value), ex.Tok, want)
		return Int
	case *ast.RealLiteral:
		c.checkRealRange(ex)
		return Real
	case *ast.UnaryExpr:
		// `-32768` разбирается как NEG(Int(32768)): знак учитывается до
		// проверки диапазона, иначе минимум INT ложно выпадал бы из него.
		// Тип литерала записывается отдельно — codegen эмитит его сам.
		if lit, ok := ex.Operand.(*ast.IntLiteral); ok && ex.Op == ast.NEG {
			if want == Real {
				c.info.Types[lit] = Real
				return Real
			}
			c.checkIntRange(-int64(lit.Value), ex.Tok, want)
			c.info.Types[lit] = Int
			return Int
		}
		t := c.typeOf(ex.Operand, want)
		if t == Bool {
			c.errorf(ex.Tok, "unary minus is not applicable to BOOL")
			return Invalid
		}
		return t
	case *ast.BinaryExpr:
		return c.inferBinary(ex, want)
	case *ast.MemberExpr:
		// Чтение inst.Out в выражении: разрешены только выходы.
		return c.resolveMember(ex, false)
	case *ast.CallExpr:
		return c.checkCall(ex)
	}
	return Invalid
}

// inferBinary: операнды одного типа; арифметика возвращает его же,
// сравнение — BOOL. Контекст операндов — подсказка hintOf от операнда с
// известным типом (так `IF 1 > r` и `IF r > 1` симметричны, а в `i + 1`
// литерал не уезжает в REAL из-за цели присваивания), и только для
// выражения из одних литералов — want сверху (`r := 1 / 2` — вещественное
// деление).
func (c *checker) inferBinary(ex *ast.BinaryExpr, want TypeKind) TypeKind {
	cmp := isComparison(ex.Op)
	w := c.hintOf(ex.Left)
	if w == Unknown {
		w = c.hintOf(ex.Right)
	}
	if w == Unknown && !cmp {
		w = want
	}
	lt := c.typeOf(ex.Left, w)
	rt := c.typeOf(ex.Right, w)
	if lt == Invalid || rt == Invalid {
		return Invalid
	}
	if lt == Bool || rt == Bool {
		if cmp {
			c.errorf(ex.Tok, "cannot compare BOOL values")
		} else {
			c.errorf(ex.Tok, "operator %q is not applicable to BOOL", ex.Op)
		}
		return Invalid
	}
	if lt != rt {
		c.errorf(ex.Tok, "operands of %q have different types: %s and %s%s", ex.Op, lt, rt, convHint(lt, rt))
		return Invalid
	}
	if cmp {
		return Bool
	}
	return lt
}

func isComparison(op ast.Op) bool {
	switch op {
	case ast.LT, ast.LE, ast.GT, ast.GE, ast.EQ, ast.NE:
		return true
	}
	return false
}

// hintOf — предварительный вывод типа без записи в side-table и без ошибок:
// тип переменной, члена, возврата функции, вещественного литерала; для
// сравнения — Bool. Unknown — контекст не задаёт (одни целые литералы,
// необъявленное имя, экземпляр ФБ).
func (c *checker) hintOf(e ast.Expression) TypeKind {
	switch ex := e.(type) {
	case *ast.Identifier:
		if sym, ok := c.syms[strings.ToUpper(ex.Name)]; ok && sym.FB == nil {
			return scalarOnly(sym.Type)
		}
	case *ast.RealLiteral:
		return Real
	case *ast.UnaryExpr:
		return c.hintOf(ex.Operand)
	case *ast.BinaryExpr:
		if isComparison(ex.Op) {
			return Bool
		}
		if h := c.hintOf(ex.Left); h != Unknown {
			return h
		}
		return c.hintOf(ex.Right)
	case *ast.MemberExpr:
		if id, ok := ex.Base.(*ast.Identifier); ok {
			if sym, ok := c.syms[strings.ToUpper(id.Name)]; ok && sym.FB != nil {
				if typeName, _, found := fbMember(sym.FB, ex.Member); found {
					return scalarOnly(c.kindOf(typeName))
				}
			}
		}
	case *ast.CallExpr:
		if id, ok := ex.Callee.(*ast.Identifier); ok {
			key := strings.ToUpper(id.Name)
			if fn, ok := c.pous[key].(*ast.Function); ok {
				return scalarOnly(c.kindOf(fn.ReturnType))
			}
			if b, ok := builtins[key]; ok {
				return b.result
			}
		}
	}
	return Unknown
}

// checkCall проверяет вызов в выражении и возвращает тип результата.
// Резолв callee: глобальная таблица POU → встроенные конверсии → локальная
// таблица (экземпляр ФБ, вызов которого в выражении — ошибка).
func (c *checker) checkCall(call *ast.CallExpr) TypeKind {
	id, ok := call.Callee.(*ast.Identifier)
	if !ok {
		c.errorf(call.Tok, "call target must be a function name")
		c.checkArgsBlind(call)
		return Invalid
	}
	// Сначала глобальная таблица POU, потом локальная: имя функции лежит в
	// её собственной локальной таблице как переменная возврата, и проверка
	// «локальная → пропустить» первой молча проглотила бы прямую рекурсию.
	key := strings.ToUpper(id.Name)
	pou, found := c.pous[key]
	if !found {
		if b, isBuiltin := builtins[key]; isBuiltin {
			return c.checkBuiltinCall(call, id, b)
		}
		if sym, isLocal := c.syms[key]; isLocal {
			if sym.FB != nil {
				// У ФБ нет возвращаемого значения — в выражении ему делать
				// нечего, вызов экземпляра пишется оператором.
				c.errorf(id.Tok, "function block instance %q cannot be called in an expression (call it as a statement)", id.Name)
			} else {
				c.errorf(id.Tok, "%q is not a function", id.Name)
			}
			c.checkArgsBlind(call)
			return Invalid
		}
		c.errorf(id.Tok, "undeclared function %q", id.Name)
		c.checkArgsBlind(call)
		return Invalid
	}
	fn, isFunc := pou.(*ast.Function)
	if !isFunc {
		c.errorf(id.Tok, "%q is not a function", id.Name)
		c.checkArgsBlind(call)
		return Invalid
	}

	if c.curFunc != "" {
		c.graph[c.curFunc] = append(c.graph[c.curFunc],
			callEdge{callee: key, name: id.Name, tok: id.Tok})
	}

	inputs := c.functionInputs(fn)
	index := map[string]int{}
	for i, in := range inputs {
		index[strings.ToUpper(in.Name)] = i
	}
	bound := make([]bool, len(inputs))
	sawNamed := false
	for i, a := range call.Args {
		switch {
		case a.Output:
			// У функции нет выходов VAR_OUTPUT — привязывать нечего.
			c.errorf(call.Tok, "output binding %q => is not applicable to function %q", a.Name, id.Name)
			c.checkWrite(a.Value)
		case a.Name == "":
			if sawNamed {
				c.errorf(call.Tok, "positional argument after named argument in call of %q", id.Name)
			}
			if i >= len(inputs) {
				c.typeOf(a.Value, Unknown)
				continue
			}
			bound[i] = true
			c.checkArg(call.Tok, id.Name, inputs[i], c.typeOf(a.Value, inputs[i].Type))
		default:
			sawNamed = true
			idx, known := index[strings.ToUpper(a.Name)]
			if !known {
				c.errorf(call.Tok, "function %q has no input %q", id.Name, a.Name)
				c.typeOf(a.Value, Unknown)
				continue
			}
			if bound[idx] {
				c.errorf(call.Tok, "input %q of function %q bound more than once", inputs[idx].Name, id.Name)
			}
			bound[idx] = true
			c.checkArg(call.Tok, id.Name, inputs[idx], c.typeOf(a.Value, inputs[idx].Type))
		}
	}
	// У функции все входы обязательны (в отличие от ФБ — этап 6).
	if len(call.Args) != len(inputs) {
		c.errorf(call.Tok, "function %q expects %d argument(s), got %d",
			id.Name, len(inputs), len(call.Args))
	}
	return c.returnKind(fn)
}

// returnKind — тип результата вызова функции; ошибку о нерезолвящемся или
// ФБ-типе возврата выдаёт проверка самой функции.
func (c *checker) returnKind(fn *ast.Function) TypeKind {
	if k := c.kindOf(fn.ReturnType); k == Int || k == Real {
		return k
	}
	return Invalid
}

// checkArg — тип аргумента равен типу входа.
func (c *checker) checkArg(tok lexer.Token, fnName string, in input, got TypeKind) {
	if in.Type == Invalid || got == Invalid || in.Type == got {
		return
	}
	c.errorf(tok, "argument for input %q of function %q must be %s, got %s%s",
		in.Name, fnName, in.Type, got, convHint(in.Type, got))
}

// checkBuiltinCall — вызов встроенной конверсии: только позиционные
// аргументы, их число и типы — по таблице builtins. Факт вызова
// запоминается в Info.UsedBuiltins. В граф рекурсии builtins не попадают.
func (c *checker) checkBuiltinCall(call *ast.CallExpr, id *ast.Identifier, b builtin) TypeKind {
	c.info.UsedBuiltins[strings.ToUpper(id.Name)] = true
	for i, a := range call.Args {
		switch {
		case a.Output:
			c.errorf(call.Tok, "output binding %q => is not applicable to built-in function %q", a.Name, id.Name)
			c.checkWrite(a.Value)
		case a.Name != "":
			c.errorf(call.Tok, "built-in function %q takes positional arguments only", id.Name)
			c.typeOf(a.Value, Unknown)
		case i >= len(b.params):
			c.typeOf(a.Value, Unknown)
		default:
			want := b.params[i]
			got := c.typeOf(a.Value, want)
			if got != Invalid && got != want {
				c.errorf(call.Tok, "argument of %q must be %s, got %s", id.Name, want, got)
			}
		}
	}
	if len(call.Args) != len(b.params) {
		c.errorf(call.Tok, "built-in function %q expects %d argument(s), got %d",
			id.Name, len(b.params), len(call.Args))
	}
	return b.result
}

// checkArgsBlind обходит значения аргументов без знания входов — чтобы
// ошибки в них не терялись, когда сам вызов уже некорректен. Цель `=>`
// проверяется как запись, остальные значения — как чтение.
func (c *checker) checkArgsBlind(call *ast.CallExpr) {
	for _, a := range call.Args {
		if a.Output {
			c.checkWrite(a.Value)
		} else {
			c.typeOf(a.Value, Unknown)
		}
	}
}

// input — один вход функции (имя из VAR_INPUT в порядке объявления).
type input struct {
	Name     string
	TypeName string
	Type     TypeKind
}

func (c *checker) functionInputs(fn *ast.Function) []input {
	var ins []input
	for _, blk := range fn.VarBlocks {
		if blk.Kind != ast.VarInput {
			continue
		}
		for _, d := range blk.Decls {
			kind := c.kindOf(d.TypeName)
			for _, n := range d.Names {
				ins = append(ins, input{Name: n.Name, TypeName: d.TypeName, Type: kind})
			}
		}
	}
	return ins
}

// fbMember ищет имя среди входов и выходов интерфейса ФБ (ключ ToUpper).
// Внутренние VAR/VAR_TEMP снаружи недоступны — для них ok == false.
func fbMember(fb *ast.FunctionBlock, name string) (typeName string, kind ast.VarKind, ok bool) {
	key := strings.ToUpper(name)
	for _, blk := range fb.VarBlocks {
		if blk.Kind != ast.VarInput && blk.Kind != ast.VarOutput {
			continue
		}
		for _, d := range blk.Decls {
			for _, n := range d.Names {
				if strings.ToUpper(n.Name) == key {
					return d.TypeName, blk.Kind, true
				}
			}
		}
	}
	return "", 0, false
}

// resolveMember резолвит `inst.Member`: база — экземпляр ФБ, член — вход или
// выход его интерфейса. Доступ извне направленный: вход — только запись,
// выход — только чтение (write задаёт контекст). Возвращает тип члена
// (Invalid — ошибка уже выдана).
func (c *checker) resolveMember(m *ast.MemberExpr, write bool) TypeKind {
	id, ok := m.Base.(*ast.Identifier)
	if !ok {
		// Вложенный доступ m.inner.sum в MVP не поддерживается: внутреннее
		// состояние выводится наружу через VAR_OUTPUT.
		c.errorf(m.Tok, "member access base must be a function block instance")
		return Invalid
	}
	sym, found := c.syms[strings.ToUpper(id.Name)]
	if !found {
		c.errorf(id.Tok, "undeclared variable %q", id.Name)
		return Invalid
	}
	if sym.FB == nil {
		c.errorf(m.Tok, "%q is not a function block instance", id.Name)
		return Invalid
	}
	typeName, kind, found := fbMember(sym.FB, m.Member)
	if !found {
		c.errorf(m.Tok, "function block %q has no input or output %q", sym.FB.Name, m.Member)
		return Invalid
	}
	if write && kind != ast.VarInput {
		c.errorf(m.Tok, "cannot assign to output %q of instance %q", m.Member, id.Name)
		return Invalid
	}
	if !write && kind != ast.VarOutput {
		c.errorf(m.Tok, "cannot read input %q of instance %q", m.Member, id.Name)
		return Invalid
	}
	return c.kindOf(typeName)
}

// checkFBCall проверяет вызов ФБ как оператор: `inst(In := x, Out => y);`.
// Аргументы только именованные; список необязателен и может быть неполным —
// непереданный вход сохраняет значение с прошлого вызова, в этом смысл
// состояния (у функций наоборот: все входы обязательны). Тип значения
// равен типу входа, тип выхода — типу цели `=>`.
func (c *checker) checkFBCall(call *ast.CallExpr) {
	id, ok := call.Callee.(*ast.Identifier)
	if !ok {
		c.errorf(call.Tok, "call statement target must be a function block instance")
		return
	}
	sym, isLocal := c.syms[strings.ToUpper(id.Name)]
	if !isLocal || sym.FB == nil {
		c.errorf(id.Tok, "%q is not a function block instance", id.Name)
		c.checkArgsBlind(call)
		return
	}
	fb := sym.FB
	bound := map[string]bool{}
	for _, a := range call.Args {
		if a.Name == "" {
			c.errorf(call.Tok, "function block call arguments must be named (In := x, Out => y)")
			c.typeOf(a.Value, Unknown)
			continue
		}
		typeName, kind, found := fbMember(fb, a.Name)
		if !found {
			c.errorf(call.Tok, "function block %q has no input or output %q", fb.Name, a.Name)
			if a.Output {
				c.checkWrite(a.Value)
			} else {
				c.typeOf(a.Value, Unknown)
			}
			continue
		}
		key := strings.ToUpper(a.Name)
		if bound[key] {
			c.errorf(call.Tok, "%q bound more than once in call of instance %q", a.Name, id.Name)
		}
		bound[key] = true
		memberType := c.kindOf(typeName)
		switch {
		case a.Output && kind != ast.VarOutput:
			c.errorf(call.Tok, "%q is an input of %q: pass it with :=, not =>", a.Name, fb.Name)
			c.checkWrite(a.Value)
		case !a.Output && kind != ast.VarInput:
			c.errorf(call.Tok, "%q is an output of %q: bind it with =>, not :=", a.Name, fb.Name)
			c.typeOf(a.Value, Unknown)
		case a.Output:
			target := c.checkWrite(a.Value)
			if target != Invalid && memberType != Invalid && target != memberType {
				c.errorf(call.Tok, "output %q of %q is %s, cannot bind it to %q of type %s%s",
					a.Name, fb.Name, memberType, targetName(a.Value), target, convHint(memberType, target))
			}
		default:
			got := c.typeOf(a.Value, memberType)
			if got != Invalid && memberType != Invalid && got != memberType {
				c.errorf(call.Tok, "input %q of %q must be %s, got %s%s",
					a.Name, fb.Name, memberType, got, convHint(memberType, got))
			}
		}
	}
}

// fbEdge — ребро графа вложенности «ФБ содержит экземпляр ФБ».
type fbEdge struct {
	inner string      // ключ ToUpper вложенного ФБ
	name  string      // имя поля-экземпляра
	tok   lexer.Token // позиция объявления поля
}

// checkInstanceCycles ловит циклическую вложенность экземпляров: ФБ, прямо
// или косвенно содержащий экземпляр самого себя, дал бы в C struct
// бесконечного размера. DFS тремя цветами по образцу checkRecursion; ошибка
// на позиции объявления, замыкающего цикл, одна на цикл.
func checkInstanceCycles(sf *ast.SourceFile, pous map[string]ast.POU) []error {
	graph := map[string][]fbEdge{}
	for _, pou := range sf.POUs {
		fb, ok := pou.(*ast.FunctionBlock)
		if !ok {
			continue
		}
		key := strings.ToUpper(fb.Name)
		for _, blk := range fb.VarBlocks {
			for _, d := range blk.Decls {
				inner, isFB := pous[strings.ToUpper(d.TypeName)].(*ast.FunctionBlock)
				if !isFB {
					continue
				}
				for _, n := range d.Names {
					graph[key] = append(graph[key],
						fbEdge{inner: strings.ToUpper(inner.Name), name: n.Name, tok: n.Tok})
				}
			}
		}
	}
	const (
		white = iota
		gray
		black
	)
	state := map[string]int{}
	var errs []error
	var visit func(key string)
	visit = func(key string) {
		state[key] = gray
		for _, e := range graph[key] {
			switch state[e.inner] {
			case gray:
				errs = append(errs, fmt.Errorf("line %d:%d: instance %q creates cyclic nesting of function blocks (a block cannot contain itself)",
					e.tok.Line, e.tok.Col, e.name))
			case white:
				visit(e.inner)
			}
		}
		state[key] = black
	}
	for _, pou := range sf.POUs {
		if fb, ok := pou.(*ast.FunctionBlock); ok {
			if key := strings.ToUpper(fb.Name); state[key] == white {
				visit(key)
			}
		}
	}
	return errs
}

// checkRecursion ищет циклы в графе вызовов функций: по IEC функции
// нерекурсивны, прямая и взаимная рекурсия — ошибка. DFS с тремя цветами;
// ребро в «серую» вершину замыкает цикл, ошибка — на позиции этого вызова
// (одна на цикл).
func checkRecursion(sf *ast.SourceFile, graph map[string][]callEdge) []error {
	const (
		white = iota
		gray
		black
	)
	state := map[string]int{}
	var errs []error
	var visit func(key string)
	visit = func(key string) {
		state[key] = gray
		for _, e := range graph[key] {
			switch state[e.callee] {
			case gray:
				errs = append(errs, fmt.Errorf("line %d:%d: recursive call of function %q (recursion is not allowed)",
					e.tok.Line, e.tok.Col, e.name))
			case white:
				visit(e.callee)
			}
		}
		state[key] = black
	}
	for _, pou := range sf.POUs {
		if fn, ok := pou.(*ast.Function); ok {
			if key := strings.ToUpper(fn.Name); state[key] == white {
				visit(key)
			}
		}
	}
	return errs
}

// lookup находит символ по регистронезависимому имени; необъявленная
// переменная — ошибка (на каждое использование).
func (c *checker) lookup(id *ast.Identifier) *Symbol {
	if sym, ok := c.syms[strings.ToUpper(id.Name)]; ok {
		return sym
	}
	c.errorf(id.Tok, "undeclared variable %q", id.Name)
	return nil
}

// checkIntRange — диапазон целого литерала, принявшего тип INT. При
// want == Invalid контекст уже ошибочен (неизвестный тип объявления) —
// диапазон не навязывается, чтобы не плодить каскад.
func (c *checker) checkIntRange(v int64, tok lexer.Token, want TypeKind) {
	if want == Invalid {
		return
	}
	if v < intMin || v > intMax {
		c.errorf(tok, "literal %d out of range for INT (%d..%d)", v, intMin, intMax)
	}
}

// checkRealRange — вещественный литерал обязан помещаться в float (REAL —
// 32 бита по IEC). Критерий — представимость: округление к float32 не даёт
// бесконечности. Сравнение с MaxFloat32 в float64 отвергло бы сам максимум
// `3.4028235e38`: его десятичная запись чуть больше точного значения, хотя
// в float32 округляется ровно к нему.
func (c *checker) checkRealRange(lit *ast.RealLiteral) {
	if math.IsInf(float64(float32(lit.Value)), 0) {
		c.errorf(lit.Tok, "literal %s out of range for REAL (magnitude up to %.8g)", lit.Text, math.MaxFloat32)
	}
}
