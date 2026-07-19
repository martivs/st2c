// Package parser строит AST из потока токенов лексера методом рекурсивного
// спуска (recursive descent): одна функция на одно правило грамматики.
//
// Обработка ошибок — fail-fast: при первой синтаксической ошибке поле err
// заполняется, все методы после этого быстро сворачиваются, а ParseProgram
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

// ParseProgram — точка входа. Возвращает корень дерева и первую ошибку (nil,
// если разбор успешен).
func (p *Parser) ParseProgram() (*ast.Program, error) {
	prog := p.parseProgram()
	if p.err != nil {
		return nil, p.err
	}
	return prog, nil
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

// parseProgram: PROGRAM IDENT [VAR-block] {statement} END_PROGRAM
func (p *Parser) parseProgram() *ast.Program {
	tok := p.expect(lexer.PROGRAM)
	name := p.expect(lexer.IDENT)

	prog := &ast.Program{Name: name.Literal, Tok: tok}

	if p.curIs(lexer.VAR) {
		prog.Vars = p.parseVarBlock()
	}

	prog.Body = p.parseStatements()

	p.expect(lexer.END_PROGRAM)
	return prog
}

// parseVarBlock: VAR { IDENT : INT ; } END_VAR
func (p *Parser) parseVarBlock() []*ast.VarDecl {
	p.expect(lexer.VAR)

	var decls []*ast.VarDecl
	for p.err == nil && !p.curIs(lexer.END_VAR) && !p.curIs(lexer.EOF) {
		name := p.expect(lexer.IDENT)
		p.expect(lexer.COLON)
		p.expect(lexer.INT)
		p.expect(lexer.SEMICOLON)
		decls = append(decls, &ast.VarDecl{Name: name.Literal, Tok: name})
	}

	p.expect(lexer.END_VAR)
	return decls
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
	case lexer.END_PROGRAM, lexer.END_IF, lexer.END_FOR, lexer.ELSE, lexer.EOF:
		return true
	default:
		return false
	}
}

// parseStatement — диспетчер по типу текущего токена.
func (p *Parser) parseStatement() ast.Statement {
	switch p.cur.Type {
	case lexer.IDENT:
		return p.parseAssign()
	case lexer.IF:
		return p.parseIf()
	case lexer.FOR:
		return p.parseFor()
	default:
		p.fail(fmt.Sprintf("unexpected token %s %q at statement start", p.cur.Type, p.cur.Literal))
		return nil
	}
}

// parseAssign: lvalue := expression ;
//
// Цель разбирается как primary-выражение; допустимый lvalue пока только
// идентификатор (литерал/скобки слева → ошибка). Когда parsePrimary научится
// постфиксам (`arr[i]`, `fb.out`), они станут целями без смены формы AST.
func (p *Parser) parseAssign() ast.Statement {
	tok := p.cur
	target := p.parsePrimary()
	if p.err != nil {
		return nil
	}
	if _, ok := target.(*ast.Identifier); !ok {
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
// по IEC 61131-3: сравнение (= <>) слабее отношений (< > <= >=), дальше
// аддитивные, мультипликативные, унарные, атомы. Новый бинарный оператор =
// по одной строке в prec и binOps.
// ---------------------------------------------------------------------------

// prec — приоритеты бинарных операторов: больше — связывает сильнее.
var prec = map[lexer.TokenType]int{
	lexer.EQ: 1, lexer.NE: 1,
	lexer.LT: 2, lexer.GT: 2, lexer.LE: 2, lexer.GE: 2,
	lexer.PLUS: 3, lexer.MINUS: 3,
	lexer.STAR: 4, lexer.SLASH: 4,
}

// lowestPrec — минимальный приоритет: parseExpression(lowestPrec) разбирает
// выражение целиком.
const lowestPrec = 1

// binOps — перевод токена-оператора в операцию AST.
var binOps = map[lexer.TokenType]ast.Op{
	lexer.PLUS: ast.ADD, lexer.MINUS: ast.SUB,
	lexer.STAR: ast.MUL, lexer.SLASH: ast.DIV,
	lexer.LT: ast.LT, lexer.LE: ast.LE,
	lexer.GT: ast.GT, lexer.GE: ast.GE,
	lexer.EQ: ast.EQ, lexer.NE: ast.NE,
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

// parseUnary — унарный минус (рекурсивно: --x допустим); сюда же позже лягут
// унарный `+` и NOT.
func (p *Parser) parseUnary() ast.Expression {
	if p.err != nil {
		return nil
	}
	if p.curIs(lexer.MINUS) {
		tok := p.cur
		p.nextToken()
		return &ast.UnaryExpr{Op: ast.NEG, Operand: p.parseUnary(), Tok: tok}
	}
	return p.parsePrimary()
}

// parsePrimary — атом: идентификатор, целый литерал или ( expression ).
func (p *Parser) parsePrimary() ast.Expression {
	if p.err != nil {
		return nil
	}
	switch p.cur.Type {
	case lexer.IDENT:
		tok := p.cur
		p.nextToken()
		return &ast.Identifier{Name: tok.Literal, Tok: tok}
	case lexer.INT_LIT:
		tok := p.cur
		p.nextToken()
		val, convErr := strconv.Atoi(tok.Literal)
		if convErr != nil {
			p.fail(fmt.Sprintf("invalid integer literal %q", tok.Literal))
			return nil
		}
		return &ast.IntLiteral{Value: val, Tok: tok}
	case lexer.LPAREN:
		p.nextToken()
		expr := p.parseExpression(lowestPrec)
		p.expect(lexer.RPAREN)
		return expr
	default:
		p.fail(fmt.Sprintf("expected expression, got %s %q", p.cur.Type, p.cur.Literal))
		return nil
	}
}
