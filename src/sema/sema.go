// Package sema — семантический анализ между парсером и codegen: проверки,
// без которых сгенерированный C был бы молча некорректен.
//
// Таблица имён двухуровневая: глобальная таблица POU (имя → узел, строится
// первым проходом) и локальная таблица переменных — своя на каждый POU.
// Полный вывод типов выражений отложен — пока проверяются объявленность,
// вызовы функций и диапазон литералов при известном целевом типе.
//
// Экземпляры ФБ, доступ к членам (`inst.Out`) и вызов ФБ как оператор
// (`inst(In := x);`) на этом этапе молча пропускаются — их проверки придут
// на этапе 6 плана 2026-08-07.
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
	return errs
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
			for _, name := range d.Names {
				key := strings.ToUpper(name.Name)
				if prev, ok := c.syms[key]; ok {
					c.errorf(name.Tok, "duplicate declaration of %q (first declared as %q at line %d)",
						name.Name, prev.Name, prev.Tok.Line)
					continue
				}
				c.syms[key] = &Symbol{Name: name.Name, TypeName: d.TypeName, Tok: name.Tok}
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
		targetType := ""
		if id, ok := st.Target.(*ast.Identifier); ok {
			if sym := c.lookup(id); sym != nil {
				targetType = sym.TypeName
			}
		} else {
			// Иные lvalue (сейчас MemberExpr) обходим как выражение; сам
			// MemberExpr до этапа 6 пропускается молча.
			c.checkExpr(st.Target, "")
		}
		c.checkExpr(st.Value, targetType)
	case *ast.IfStatement:
		// Целевой тип операндов условия — INT: других типов в MVP нет,
		// полноценный вывод типов (BOOL у сравнений) придёт со слоем типов.
		c.checkExpr(st.Condition, "INT")
		c.checkStatements(st.Then)
		c.checkStatements(st.Else)
	case *ast.ForStatement:
		loopType := ""
		if sym := c.lookup(st.Var); sym != nil {
			loopType = sym.TypeName
		}
		c.checkExpr(st.Start, loopType)
		c.checkExpr(st.End, loopType)
		if st.Step != nil {
			c.checkExpr(st.Step, loopType)
		}
		c.checkStatements(st.Body)
	case *ast.CallStatement:
		// Вызов ФБ как оператор — проверки экземпляров придут на этапе 6.
	}
}

// checkExpr обходит выражение: ссылки должны быть объявлены, целочисленные
// литералы — попадать в диапазон целевого типа. targetType == "" — целевой
// тип неизвестен, диапазон не проверяется.
func (c *checker) checkExpr(e ast.Expression, targetType string) {
	switch ex := e.(type) {
	case *ast.Identifier:
		c.lookup(ex)
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
		// inst.Out — резолв членов экземпляров ФБ придёт на этапе 6.
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
		if _, isLocal := c.syms[key]; isLocal {
			// Локальная переменная — экземпляр ФБ; проверки вызова — этап 6.
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
			c.checkExpr(a.Value, "")
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
// ошибки в них не терялись, когда сам вызов уже некорректен.
func (c *checker) checkArgsBlind(call *ast.CallExpr) {
	for _, a := range call.Args {
		c.checkExpr(a.Value, "")
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
