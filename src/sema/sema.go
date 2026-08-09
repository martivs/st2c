// Package sema — семантический анализ между парсером и codegen: проверки,
// без которых сгенерированный C был бы молча некорректен.
//
// Таблица имён двухуровневая: глобальная таблица POU (имя → узел, строится
// первым проходом) и локальная таблица переменных — своя на каждый POU.
// Полный вывод типов выражений отложен — пока проверяются объявленность,
// вызовы функций и диапазон литералов при известном целевом типе.
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
	"strings"

	"st2c/src/ast"
	"st2c/src/lexer"
)

// Пределы IEC-типа INT: 16 бит со знаком.
const (
	intMin = -32768
	intMax = 32767
)

// Symbol — объявленная переменная в символьной таблице.
type Symbol struct {
	Name     string      // оригинальное написание из объявления
	TypeName string      // имя типа как записано в объявлении
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
	errs []error
	// curFunc — ключ функции, чьё тело обходится ("" вне функций); только
	// функции попадают в граф вызовов: PROGRAM и ФБ из выражений не вызываются.
	curFunc string
	graph   map[string][]callEdge
}

// Check проверяет дерево и возвращает все найденные семантические ошибки
// (nil — ошибок нет).
func Check(sf *ast.SourceFile) []error {
	var errs []error

	// Первый проход — глобальная таблица POU: тела могут вызывать функции,
	// объявленные ниже по файлу.
	pous := map[string]ast.POU{}
	for _, pou := range sf.POUs {
		name, tok := pouName(pou)
		key := strings.ToUpper(name)
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
		c := &checker{pous: pous, syms: map[string]*Symbol{}, graph: graph}
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
	return errs
}

// resolveType резолвит тип объявления: INT — встроенный, остальные имена
// ищутся в глобальной таблице POU. Найденный FUNCTION_BLOCK делает переменную
// экземпляром ФБ; другой POU или неизвестное имя — ошибка (молчаливый пропуск
// пользовательских типов ушёл с этапом 6).
func (c *checker) resolveType(d *ast.VarDecl) *ast.FunctionBlock {
	if strings.EqualFold(d.TypeName, "INT") {
		return nil
	}
	pou, ok := c.pous[strings.ToUpper(d.TypeName)]
	if !ok {
		c.errorf(d.Tok, "unknown type %q", d.TypeName)
		return nil
	}
	fb, isFB := pou.(*ast.FunctionBlock)
	if !isFB {
		c.errorf(d.Tok, "%q is not a type", d.TypeName)
		return nil
	}
	return fb
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
// даст обычную ошибку дубликата.
func (c *checker) checkFunction(f *ast.Function) {
	key := strings.ToUpper(f.Name)
	c.syms[key] = &Symbol{Name: f.Name, TypeName: f.ReturnType, Tok: f.Tok}
	c.curFunc = key
	c.checkBody(f.VarBlocks, f.Body)
}

// checkBody — общий обход POU: объявления, инициализаторы, операторы.
func (c *checker) checkBody(blocks []*ast.VarBlock, body []ast.Statement) {
	// Первый проход — собрать все объявления: инициализаторы проверяются
	// отдельным проходом, чтобы порядок объявлений не влиял на объявленность.
	for _, blk := range blocks {
		for _, d := range blk.Decls {
			fb := c.resolveType(d)
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
				c.syms[key] = &Symbol{Name: name.Name, TypeName: d.TypeName, Tok: name.Tok, FB: fb}
			}
		}
	}
	for _, blk := range blocks {
		for _, d := range blk.Decls {
			if d.Init != nil {
				c.checkExpr(d.Init, d.TypeName)
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
		c.checkExpr(st.Value, c.checkWrite(st.Target))
	case *ast.IfStatement:
		// Целевой тип операндов условия — INT: других типов в MVP нет,
		// полноценный вывод типов (BOOL у сравнений) придёт со слоем типов.
		c.checkExpr(st.Condition, "INT")
		c.checkStatements(st.Then)
		c.checkStatements(st.Else)
	case *ast.ForStatement:
		loopType := c.checkWrite(st.Var)
		c.checkExpr(st.Start, loopType)
		c.checkExpr(st.End, loopType)
		if st.Step != nil {
			c.checkExpr(st.Step, loopType)
		}
		c.checkStatements(st.Body)
	case *ast.CallStatement:
		c.checkFBCall(st.Call)
	}
}

// checkWrite проверяет цель записи (цель присваивания, переменную FOR,
// цель `name => target`) и возвращает её тип ("" — тип неизвестен, диапазон
// литералов не проверяется). Экземпляр ФБ целиком — не lvalue.
func (c *checker) checkWrite(target ast.Expression) string {
	switch t := target.(type) {
	case *ast.Identifier:
		if sym := c.lookup(t); sym != nil {
			if sym.FB != nil {
				c.errorf(t.Tok, "cannot assign to function block instance %q", t.Name)
				return ""
			}
			return sym.TypeName
		}
	case *ast.MemberExpr:
		return c.resolveMember(t, true)
	default:
		c.checkExpr(target, "")
	}
	return ""
}

// checkExpr обходит выражение: ссылки должны быть объявлены, целочисленные
// литералы — попадать в диапазон целевого типа. targetType == "" — целевой
// тип неизвестен, диапазон не проверяется.
func (c *checker) checkExpr(e ast.Expression, targetType string) {
	switch ex := e.(type) {
	case *ast.Identifier:
		if sym := c.lookup(ex); sym != nil && sym.FB != nil {
			c.errorf(ex.Tok, "function block instance %q cannot be used as a value", ex.Name)
		}
	case *ast.IntLiteral:
		c.checkIntRange(int64(ex.Value), ex.Tok, targetType)
	case *ast.UnaryExpr:
		// `-32768` разбирается как NEG(Int(32768)): знак учитывается до
		// проверки диапазона, иначе минимум INT ложно выпадал бы из него.
		if lit, ok := ex.Operand.(*ast.IntLiteral); ok && ex.Op == ast.NEG {
			c.checkIntRange(-int64(lit.Value), ex.Tok, targetType)
			return
		}
		c.checkExpr(ex.Operand, targetType)
	case *ast.BinaryExpr:
		c.checkExpr(ex.Left, targetType)
		c.checkExpr(ex.Right, targetType)
	case *ast.MemberExpr:
		// Чтение inst.Out в выражении: разрешены только выходы.
		c.resolveMember(ex, false)
	case *ast.CallExpr:
		c.checkCall(ex)
	}
}

// checkCall проверяет вызов функции в выражении. Вызов через локальную
// переменную (экземпляр ФБ) молча пропускается до этапа 6.
func (c *checker) checkCall(call *ast.CallExpr) {
	id, ok := call.Callee.(*ast.Identifier)
	if !ok {
		c.errorf(call.Tok, "call target must be a function name")
		return
	}
	// Сначала глобальная таблица POU, потом локальная: имя функции лежит в
	// её собственной локальной таблице как переменная возврата, и проверка
	// «локальная → пропустить» первой молча проглотила бы прямую рекурсию.
	key := strings.ToUpper(id.Name)
	pou, found := c.pous[key]
	if !found {
		if sym, isLocal := c.syms[key]; isLocal {
			if sym.FB != nil {
				// У ФБ нет возвращаемого значения — в выражении ему делать
				// нечего, вызов экземпляра пишется оператором.
				c.errorf(id.Tok, "function block instance %q cannot be called in an expression (call it as a statement)", id.Name)
			} else {
				c.errorf(id.Tok, "%q is not a function", id.Name)
			}
			c.checkArgsBlind(call)
			return
		}
		c.errorf(id.Tok, "undeclared function %q", id.Name)
		c.checkArgsBlind(call)
		return
	}
	fn, isFunc := pou.(*ast.Function)
	if !isFunc {
		c.errorf(id.Tok, "%q is not a function", id.Name)
		c.checkArgsBlind(call)
		return
	}

	if c.curFunc != "" {
		c.graph[c.curFunc] = append(c.graph[c.curFunc],
			callEdge{callee: key, name: id.Name, tok: id.Tok})
	}

	inputs := functionInputs(fn)
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
			argType := ""
			if i < len(inputs) {
				bound[i] = true
				argType = inputs[i].TypeName
			}
			c.checkExpr(a.Value, argType)
		default:
			sawNamed = true
			idx, known := index[strings.ToUpper(a.Name)]
			if !known {
				c.errorf(call.Tok, "function %q has no input %q", id.Name, a.Name)
				c.checkExpr(a.Value, "")
				continue
			}
			if bound[idx] {
				c.errorf(call.Tok, "input %q of function %q bound more than once", inputs[idx].Name, id.Name)
			}
			bound[idx] = true
			c.checkExpr(a.Value, inputs[idx].TypeName)
		}
	}
	// У функции все входы обязательны (в отличие от ФБ — этап 6).
	if len(call.Args) != len(inputs) {
		c.errorf(call.Tok, "function %q expects %d argument(s), got %d",
			id.Name, len(inputs), len(call.Args))
	}
}

// checkArgsBlind обходит значения аргументов без знания входов — чтобы
// ошибки в них не терялись, когда сам вызов уже некорректен. Цель `=>`
// проверяется как запись, остальные значения — как чтение.
func (c *checker) checkArgsBlind(call *ast.CallExpr) {
	for _, a := range call.Args {
		if a.Output {
			c.checkWrite(a.Value)
		} else {
			c.checkExpr(a.Value, "")
		}
	}
}

// input — один вход функции (имя из VAR_INPUT в порядке объявления).
type input struct {
	Name     string
	TypeName string
}

func functionInputs(fn *ast.Function) []input {
	var ins []input
	for _, blk := range fn.VarBlocks {
		if blk.Kind != ast.VarInput {
			continue
		}
		for _, d := range blk.Decls {
			for _, n := range d.Names {
				ins = append(ins, input{Name: n.Name, TypeName: d.TypeName})
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
// ("" — ошибка уже выдана, диапазон литералов не проверяется).
func (c *checker) resolveMember(m *ast.MemberExpr, write bool) string {
	id, ok := m.Base.(*ast.Identifier)
	if !ok {
		// Вложенный доступ m.inner.sum в MVP не поддерживается: внутреннее
		// состояние выводится наружу через VAR_OUTPUT.
		c.errorf(m.Tok, "member access base must be a function block instance")
		return ""
	}
	sym, found := c.syms[strings.ToUpper(id.Name)]
	if !found {
		c.errorf(id.Tok, "undeclared variable %q", id.Name)
		return ""
	}
	if sym.FB == nil {
		c.errorf(m.Tok, "%q is not a function block instance", id.Name)
		return ""
	}
	typeName, kind, found := fbMember(sym.FB, m.Member)
	if !found {
		c.errorf(m.Tok, "function block %q has no input or output %q", sym.FB.Name, m.Member)
		return ""
	}
	if write && kind != ast.VarInput {
		c.errorf(m.Tok, "cannot assign to output %q of instance %q", m.Member, id.Name)
		return ""
	}
	if !write && kind != ast.VarOutput {
		c.errorf(m.Tok, "cannot read input %q of instance %q", m.Member, id.Name)
		return ""
	}
	return typeName
}

// checkFBCall проверяет вызов ФБ как оператор: `inst(In := x, Out => y);`.
// Аргументы только именованные; список необязателен и может быть неполным —
// непереданный вход сохраняет значение с прошлого вызова, в этом смысл
// состояния (у функций наоборот: все входы обязательны).
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
			c.checkExpr(a.Value, "")
			continue
		}
		typeName, kind, found := fbMember(fb, a.Name)
		if !found {
			c.errorf(call.Tok, "function block %q has no input or output %q", fb.Name, a.Name)
			if a.Output {
				c.checkWrite(a.Value)
			} else {
				c.checkExpr(a.Value, "")
			}
			continue
		}
		key := strings.ToUpper(a.Name)
		if bound[key] {
			c.errorf(call.Tok, "%q bound more than once in call of instance %q", a.Name, id.Name)
		}
		bound[key] = true
		switch {
		case a.Output && kind != ast.VarOutput:
			c.errorf(call.Tok, "%q is an input of %q: pass it with :=, not =>", a.Name, fb.Name)
			c.checkWrite(a.Value)
		case !a.Output && kind != ast.VarInput:
			c.errorf(call.Tok, "%q is an output of %q: bind it with =>, not :=", a.Name, fb.Name)
			c.checkExpr(a.Value, "")
		case a.Output:
			c.checkWrite(a.Value)
		default:
			c.checkExpr(a.Value, typeName)
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

func (c *checker) checkIntRange(v int64, tok lexer.Token, targetType string) {
	if !strings.EqualFold(targetType, "INT") {
		return
	}
	if v < intMin || v > intMax {
		c.errorf(tok, "literal %d out of range for INT (%d..%d)", v, intMin, intMax)
	}
}
