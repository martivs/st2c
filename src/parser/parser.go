// Package parser строит AST из потока токенов лексера методом рекурсивного
// спуска (recursive descent): одна функция на одно правило грамматики.
//
// Обработка ошибок — fail-fast: при первой синтаксической ошибке поле err
// заполняется, все методы после этого быстро сворачиваются, а ParseSourceFile
// возвращает эту ошибку. Ошибка передаётся как значение (result, error).
package parser

import (
	"fmt"
	"strconv"

	"st2c/src/ast"
	"st2c/src/lexer"
)

// Parser держит лексер и окно из двух токенов: текущий (cur) и один вперёд
// (peek). Лексер отдаёт токены по одному (pull-модель), поэтому буфер
// маленький.
type Parser struct {
	lex  *lexer.Lexer
	cur  lexer.Token
	peek lexer.Token
	err  error // первая встреченная ошибка (fail-fast)
}

// New создаёт парсер и заполняет cur/peek первыми двумя токенами.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{lex: l}
	p.nextToken() // заполнить peek
	p.nextToken() // сдвинуть в cur, peek — следующий
	return p
}

// ParseSourceFile — точка входа: {POU} до EOF. Возвращает корень дерева и
// первую ошибку (nil, если разбор успешен). Диспетчер по стартовому ключевому
// слову: PROGRAM, FUNCTION, FUNCTION_BLOCK.
func (p *Parser) ParseSourceFile() (*ast.SourceFile, error) {
	sf := &ast.SourceFile{}
	for p.err == nil && !p.curIs(lexer.EOF) {
		switch p.cur.Type {
		case lexer.PROGRAM:
			sf.POUs = append(sf.POUs, p.parseProgram())
		case lexer.FUNCTION:
			sf.POUs = append(sf.POUs, p.parseFunction())
		case lexer.FUNCTION_BLOCK:
			sf.POUs = append(sf.POUs, p.parseFunctionBlock())
		default:
			p.fail(fmt.Sprintf("expected PROGRAM, FUNCTION or FUNCTION_BLOCK at top level, got %s %q", p.cur.Type, p.cur.Literal))
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	return sf, nil
}

// ---------------------------------------------------------------------------
// Работа с токенами
// ---------------------------------------------------------------------------

func (p *Parser) nextToken() {
	p.cur = p.peek
	p.peek = p.lex.NextToken()
}

func (p *Parser) curIs(t lexer.TokenType) bool  { return p.cur.Type == t }
func (p *Parser) peekIs(t lexer.TokenType) bool { return p.peek.Type == t }

// expect проверяет тип текущего токена, возвращает его и шагает дальше. При
// несовпадении фиксирует ошибку (fail-fast) и возвращает текущий токен без
// продвижения.
func (p *Parser) expect(t lexer.TokenType) lexer.Token {
	if p.err != nil {
		return p.cur
	}
	if !p.curIs(t) {
		p.fail(fmt.Sprintf("expected %s, got %s %q", t, p.cur.Type, p.cur.Literal))
		return p.cur
	}
	tok := p.cur
	p.nextToken()
	return tok
}

// fail фиксирует первую синтаксическую ошибку с номером строки.
func (p *Parser) fail(msg string) {
	if p.err == nil {
		p.err = fmt.Errorf("line %d: %s", p.cur.Line, msg)
	}
}

// ---------------------------------------------------------------------------
// Грамматика: правила сверху вниз
// ---------------------------------------------------------------------------

// parseProgram: PROGRAM IDENT {VAR-block} {statement} END_PROGRAM
func (p *Parser) parseProgram() *ast.Program {
	tok := p.expect(lexer.PROGRAM)
	name := p.expect(lexer.IDENT)

	prog := &ast.Program{Name: name.Literal, Tok: tok}
	prog.VarBlocks = p.parseVarBlocks()
	prog.Body = p.parseStatements()

	p.expect(lexer.END_PROGRAM)
	return prog
}

// parseFunction: FUNCTION IDENT : type {VAR-block} {statement} END_FUNCTION
//
// Тип возврата — через parseType: встроенный или пользовательский, решает
// sema. Возврат значения по IEC — присваивание имени функции в теле.
func (p *Parser) parseFunction() *ast.Function {
	tok := p.expect(lexer.FUNCTION)
	name := p.expect(lexer.IDENT)
	p.expect(lexer.COLON)

	fn := &ast.Function{Name: name.Literal, ReturnType: p.parseType(), Tok: tok}
	fn.VarBlocks = p.parseVarBlocks()
	fn.Body = p.parseStatements()

	p.expect(lexer.END_FUNCTION)
	return fn
}

// parseFunctionBlock: FUNCTION_BLOCK IDENT {VAR-block} {statement}
// END_FUNCTION_BLOCK — как parseFunction, но без типа возврата.
func (p *Parser) parseFunctionBlock() *ast.FunctionBlock {
	tok := p.expect(lexer.FUNCTION_BLOCK)
	name := p.expect(lexer.IDENT)

	fb := &ast.FunctionBlock{Name: name.Literal, Tok: tok}
	fb.VarBlocks = p.parseVarBlocks()
	fb.Body = p.parseStatements()

	p.expect(lexer.END_FUNCTION_BLOCK)
	return fb
}

// varBlockKinds — стартовые токены семейства VAR* → вид блока.
var varBlockKinds = map[lexer.TokenType]ast.VarKind{
	lexer.VAR:        ast.VarPlain,
	lexer.VAR_INPUT:  ast.VarInput,
	lexer.VAR_OUTPUT: ast.VarOutput,
	lexer.VAR_IN_OUT: ast.VarInOut,
	lexer.VAR_TEMP:   ast.VarTemp,
}

// parseVarBlocks — блоки объявлений подряд, каждый из семейства VAR*.
func (p *Parser) parseVarBlocks() []*ast.VarBlock {
	var blocks []*ast.VarBlock
	for p.err == nil {
		kind, ok := varBlockKinds[p.cur.Type]
		if !ok {
			break
		}
		blocks = append(blocks, p.parseVarBlock(kind))
	}
	return blocks
}

// parseVarBlock: VAR* [CONSTANT] [RETAIN] { var-decl } END_VAR
func (p *Parser) parseVarBlock(kind ast.VarKind) *ast.VarBlock {
	blk := &ast.VarBlock{Kind: kind, Tok: p.cur}
	p.nextToken()

	for {
		if p.curIs(lexer.CONSTANT) && !blk.Constant {
			blk.Constant = true
			p.nextToken()
		} else if p.curIs(lexer.RETAIN) && !blk.Retain {
			blk.Retain = true
			p.nextToken()
		} else {
			break
		}
	}

	for p.err == nil && !p.curIs(lexer.END_VAR) && !p.curIs(lexer.EOF) {
		blk.Decls = append(blk.Decls, p.parseVarDecl())
	}

	p.expect(lexer.END_VAR)
	return blk
}

// parseVarDecl: IDENT {, IDENT} : type [:= expression] ;
func (p *Parser) parseVarDecl() *ast.VarDecl {
	first := p.expect(lexer.IDENT)
	decl := &ast.VarDecl{
		Names: []*ast.Identifier{{Name: first.Literal, Tok: first}},
		Tok:   first,
	}
	for p.err == nil && p.curIs(lexer.COMMA) {
		p.nextToken()
		name := p.expect(lexer.IDENT)
		decl.Names = append(decl.Names, &ast.Identifier{Name: name.Literal, Tok: name})
	}

	p.expect(lexer.COLON)
	decl.TypeName = p.parseType()

	if p.curIs(lexer.ASSIGN) {
		p.nextToken()
		decl.Init = p.parseExpression(lowestPrec)
	}
	p.expect(lexer.SEMICOLON)
	return decl
}

// parseType — имя типа: ключевое слово встроенного типа (INT, REAL, BOOL)
// или любой идентификатор (пользовательский тип). Допустимость имени — задача
// sema, не парсера.
func (p *Parser) parseType() string {
	switch p.cur.Type {
	case lexer.INT, lexer.REAL, lexer.BOOL, lexer.IDENT:
		tok := p.cur
		p.nextToken()
		return tok.Literal
	default:
		p.fail(fmt.Sprintf("expected type name, got %s %q", p.cur.Type, p.cur.Literal))
		return ""
	}
}

// parseStatements читает операторы до терминатора блока (END_*, ELSE, EOF).
func (p *Parser) parseStatements() []ast.Statement {
	var stmts []ast.Statement
	for p.err == nil && !p.isBlockEnd() {
		s := p.parseStatement()
		if s == nil {
			break
		}
		stmts = append(stmts, s)
	}
	return stmts
}

// isBlockEnd — токен закрывает текущий блок операторов?
func (p *Parser) isBlockEnd() bool {
	switch p.cur.Type {
	case lexer.END_PROGRAM, lexer.END_FUNCTION, lexer.END_FUNCTION_BLOCK,
		lexer.END_IF, lexer.END_FOR, lexer.ELSE, lexer.EOF:
		return true
	default:
		return false
	}
}

// parseStatement — диспетчер по типу текущего токена.
func (p *Parser) parseStatement() ast.Statement {
	switch p.cur.Type {
	case lexer.IDENT:
		return p.parseAssignOrCall()
	case lexer.IF:
		return p.parseIf()
	case lexer.FOR:
		return p.parseFor()
	default:
		p.fail(fmt.Sprintf("unexpected token %s %q at statement start", p.cur.Type, p.cur.Literal))
		return nil
	}
}

// parseAssignOrCall: оператор, начинающийся с идентификатора. Сначала
// разбирается postfix-выражение (parsePrimary даёт Identifier, MemberExpr
// или CallExpr), затем решает текущий токен: `:=` — присваивание (цель
// обязана быть Identifier или MemberExpr), `;` после CallExpr — вызов ФБ
// как оператор, иначе ошибка.
func (p *Parser) parseAssignOrCall() ast.Statement {
	tok := p.cur
	target := p.parsePrimary()
	if p.err != nil {
		return nil
	}
	if call, ok := target.(*ast.CallExpr); ok {
		p.expect(lexer.SEMICOLON)
		if p.err != nil {
			return nil
		}
		return &ast.CallStatement{Call: call}
	}
	switch target.(type) {
	case *ast.Identifier, *ast.MemberExpr:
		// допустимые цели присваивания
	default:
		p.err = fmt.Errorf("line %d: invalid assignment target %q", tok.Line, tok.Literal)
		return nil
	}
	p.expect(lexer.ASSIGN)
	value := p.parseExpression(lowestPrec)
	p.expect(lexer.SEMICOLON)
	return &ast.AssignStatement{Target: target, Value: value, Tok: tok}
}

// parseIf: IF expression THEN {statement} [ELSE {statement}] END_IF ;?
func (p *Parser) parseIf() ast.Statement {
	tok := p.expect(lexer.IF)
	cond := p.parseExpression(lowestPrec)
	p.expect(lexer.THEN)

	stmt := &ast.IfStatement{Condition: cond, Tok: tok}
	stmt.Then = p.parseStatements()

	if p.curIs(lexer.ELSE) {
		p.expect(lexer.ELSE)
		stmt.Else = p.parseStatements()
	}

	p.expect(lexer.END_IF)
	p.optionalSemicolon()
	return stmt
}

// parseFor: FOR IDENT := expression TO expression [BY expression] DO
//           {statement} END_FOR ;?
func (p *Parser) parseFor() ast.Statement {
	tok := p.expect(lexer.FOR)
	name := p.expect(lexer.IDENT)
	p.expect(lexer.ASSIGN)
	start := p.parseExpression(lowestPrec)
	p.expect(lexer.TO)
	end := p.parseExpression(lowestPrec)

	stmt := &ast.ForStatement{
		Var:   &ast.Identifier{Name: name.Literal, Tok: name},
		Start: start,
		End:   end,
		Tok:   tok,
	}
	if p.curIs(lexer.BY) {
		p.nextToken()
		stmt.Step = p.parseExpression(lowestPrec)
	}
	p.expect(lexer.DO)
	stmt.Body = p.parseStatements()

	p.expect(lexer.END_FOR)
	p.optionalSemicolon()
	return stmt
}

// optionalSemicolon съедает `;` после END_IF/END_FOR, если он есть
// (в example.st он присутствует).
func (p *Parser) optionalSemicolon() {
	if p.curIs(lexer.SEMICOLON) {
		p.nextToken()
	}
}

// ---------------------------------------------------------------------------
// Выражения: Pratt / precedence climbing
//
// Одна функция parseExpression(minPrec) + таблица приоритетов prec. Уровни —
// по IEC 61131-3, от слабого к сильному: OR, XOR, AND (и синоним &),
// сравнение (= <>), отношения (< > <= >=), аддитивные, мультипликативные,
// унарные (-, NOT), атомы с постфиксами. Новый бинарный оператор = по одной
// строке в prec и binOps, унарный — строка в unaryOps.
// ---------------------------------------------------------------------------

// prec — приоритеты бинарных операторов: больше — связывает сильнее.
// Логические уровни (1–3) ниже всех остальных: `x > 1 AND y < 2` — AND над
// двумя сравнениями, `a OR b AND c` — OR над AND. Относительный порядок
// прежних уровней (сравнение < отношения < аддитивные < мультипликативные)
// сохранён, поэтому разбор старых программ не изменился (golden .ast).
var prec = map[lexer.TokenType]int{
	lexer.OR:  1,
	lexer.XOR: 2,
	lexer.AND: 3, lexer.AMP: 3,
	lexer.EQ: 4, lexer.NE: 4,
	lexer.LT: 5, lexer.GT: 5, lexer.LE: 5, lexer.GE: 5,
	lexer.PLUS: 6, lexer.MINUS: 6,
	lexer.STAR: 7, lexer.SLASH: 7,
}

// lowestPrec — минимальный приоритет: parseExpression(lowestPrec) разбирает
// выражение целиком.
const lowestPrec = 1

// binOps — перевод токена-оператора в операцию AST. `&` и `AND` дают одну
// и ту же ast.AND: в дереве синоним неотличим от слова (решение 4 работы по
// BOOL), обратная печать исходника — задача будущего форматтера.
var binOps = map[lexer.TokenType]ast.Op{
	lexer.PLUS: ast.ADD, lexer.MINUS: ast.SUB,
	lexer.STAR: ast.MUL, lexer.SLASH: ast.DIV,
	lexer.LT: ast.LT, lexer.LE: ast.LE,
	lexer.GT: ast.GT, lexer.GE: ast.GE,
	lexer.EQ: ast.EQ, lexer.NE: ast.NE,
	lexer.AND: ast.AND, lexer.AMP: ast.AND,
	lexer.OR: ast.OR, lexer.XOR: ast.XOR,
}

// unaryOps — перевод токена унарного оператора в операцию AST.
var unaryOps = map[lexer.TokenType]ast.Op{
	lexer.MINUS: ast.NEG,
	lexer.NOT:   ast.NOT,
}

// parseExpression разбирает выражение, состоящее из операторов с приоритетом
// не ниже minPrec. Правый операнд берётся с приоритетом pr+1 — это даёт
// левую ассоциативность (a - b - c → (a-b)-c).
func (p *Parser) parseExpression(minPrec int) ast.Expression {
	left := p.parseUnary()
	for p.err == nil {
		pr, ok := prec[p.cur.Type]
		if !ok || pr < minPrec {
			break
		}
		op := p.cur
		p.nextToken()
		right := p.parseExpression(pr + 1)
		left = &ast.BinaryExpr{Left: left, Op: binOps[op.Type], Right: right, Tok: op}
	}
	return left
}

// parseUnary — унарные минус и NOT, оба рекурсивно: `--x`, `NOT NOT b`,
// `NOT -x` разбираются (допустимость типов — за sema). Унарный уровень сильнее
// любого бинарного: `NOT a = b` — это `(NOT a) = b`, как и по IEC. Унарный `+`
// пока не поддерживается.
func (p *Parser) parseUnary() ast.Expression {
	if p.err != nil {
		return nil
	}
	if op, ok := unaryOps[p.cur.Type]; ok {
		tok := p.cur
		p.nextToken()
		return &ast.UnaryExpr{Op: op, Operand: p.parseUnary(), Tok: tok}
	}
	return p.parsePrimary()
}

// parsePrimary — атом (идентификатор, целый, вещественный или булев литерал,
// ( expression )) плюс
// постфиксы: пока текущий токен `.` или `(`, атом наращивается в MemberExpr
// (`inst.Out`) или CallExpr (`Add(1, 2)`). `(` после primary — постфиксный
// оператор вызова из таблицы приоритетов Pratt (максимальный уровень —
// сильнее унарного минуса: `-f(x)` это `-(f(x))`).
func (p *Parser) parsePrimary() ast.Expression {
	if p.err != nil {
		return nil
	}
	var expr ast.Expression
	switch p.cur.Type {
	case lexer.IDENT:
		tok := p.cur
		p.nextToken()
		expr = &ast.Identifier{Name: tok.Literal, Tok: tok}
	case lexer.INT_LIT:
		tok := p.cur
		p.nextToken()
		val, convErr := strconv.Atoi(tok.Literal)
		if convErr != nil {
			p.fail(fmt.Sprintf("invalid integer literal %q", tok.Literal))
			return nil
		}
		expr = &ast.IntLiteral{Value: val, Tok: tok}
	case lexer.REAL_LIT:
		tok := p.cur
		p.nextToken()
		val, convErr := strconv.ParseFloat(tok.Literal, 64)
		if convErr != nil {
			p.fail(fmt.Sprintf("invalid real literal %q", tok.Literal))
			return nil
		}
		expr = &ast.RealLiteral{Value: val, Text: tok.Literal, Tok: tok}
	case lexer.TRUE, lexer.FALSE:
		tok := p.cur
		p.nextToken()
		expr = &ast.BoolLiteral{Value: tok.Type == lexer.TRUE, Tok: tok}
	case lexer.LPAREN:
		p.nextToken()
		expr = p.parseExpression(lowestPrec)
		p.expect(lexer.RPAREN)
	default:
		p.fail(fmt.Sprintf("expected expression, got %s %q", p.cur.Type, p.cur.Literal))
		return nil
	}

	for p.err == nil {
		switch p.cur.Type {
		case lexer.DOT:
			tok := p.cur
			p.nextToken()
			member := p.expect(lexer.IDENT)
			expr = &ast.MemberExpr{Base: expr, Member: member.Literal, Tok: tok}
		case lexer.LPAREN:
			tok := p.cur
			p.nextToken()
			expr = &ast.CallExpr{Callee: expr, Args: p.parseCallArgs(), Tok: tok}
		default:
			return expr
		}
	}
	return expr
}

// parseCallArgs — аргументы вызова до `)`, через запятую. Формы аргумента:
// `name := expr` (именованный вход), `name => lvalue` (привязка выхода ФБ;
// цель обязана быть Identifier или MemberExpr), иначе позиционное выражение.
// Различение — по peek после IDENT; `f()` без аргументов допустим.
func (p *Parser) parseCallArgs() []*ast.Arg {
	var args []*ast.Arg
	for p.err == nil && !p.curIs(lexer.RPAREN) {
		args = append(args, p.parseCallArg())
		if !p.curIs(lexer.COMMA) {
			break
		}
		p.nextToken()
	}
	p.expect(lexer.RPAREN)
	return args
}

// parseCallArg — один аргумент вызова (см. parseCallArgs).
func (p *Parser) parseCallArg() *ast.Arg {
	if p.curIs(lexer.IDENT) && p.peekIs(lexer.ASSIGN) {
		name := p.cur
		p.nextToken() // имя
		p.nextToken() // :=
		return &ast.Arg{Name: name.Literal, Value: p.parseExpression(lowestPrec)}
	}
	if p.curIs(lexer.IDENT) && p.peekIs(lexer.ARROW) {
		name := p.cur
		p.nextToken() // имя
		arrow := p.cur
		p.nextToken() // =>
		target := p.parsePrimary()
		if p.err == nil {
			switch target.(type) {
			case *ast.Identifier, *ast.MemberExpr:
				// допустимые цели привязки выхода
			default:
				p.err = fmt.Errorf("line %d: output binding %s => requires a variable, got expression", arrow.Line, name.Literal)
				return nil
			}
		}
		return &ast.Arg{Name: name.Literal, Value: target, Output: true}
	}
	return &ast.Arg{Value: p.parseExpression(lowestPrec)}
}
