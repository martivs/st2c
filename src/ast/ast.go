// Package ast описывает узлы абстрактного синтаксического дерева (AST),
// которое строит parser из потока токенов лексера.
//
// Иерархия узлов задана интерфейсами: конкретный тип «реализует» интерфейс
// тем, что определяет все его методы.
package ast

import (
	"fmt"
	"strings"

	"st2c/src/lexer"
)

// Node — базовый интерфейс любого узла дерева.
type Node interface {
	// String рекурсивно печатает поддерево с отступами.
	String() string
	// Line — номер строки в исходнике; пригодится для сообщений об ошибках.
	Line() int
}

// Statement — узел-оператор (присваивание, IF, FOR).
type Statement interface {
	Node
	statementNode() // маркерный метод-тег, см. комментарий ниже
}

// Expression — узел-выражение (идентификатор, литерал, бинарная операция).
type Expression interface {
	Node
	expressionNode()
}

// Маркерные методы statementNode()/expressionNode() пустые и ничего не
// делают. Их роль — на уровне типов различать операторы и выражения:
// интерфейсы в Go структурные, и без такого «тега» любой узел подошёл бы
// под оба интерфейса. Так компилятор не даст подставить выражение туда,
// где нужен оператор.

// ---------------------------------------------------------------------------
// Корень и объявления
// ---------------------------------------------------------------------------

// Program — корень дерева: всё, что между PROGRAM и END_PROGRAM.
type Program struct {
	Name string
	Vars []*VarDecl
	Body []Statement
	Tok  lexer.Token // токен PROGRAM
}

func (p *Program) Line() int { return p.Tok.Line }
func (p *Program) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Program(%s)\n", p.Name)
	for _, v := range p.Vars {
		indent(&b, v.String(), 1)
	}
	for _, s := range p.Body {
		indent(&b, s.String(), 1)
	}
	return b.String()
}

// VarDecl — одно объявление переменной: `x : INT;`. Тип в MVP всегда INT,
// поэтому отдельно его не храним.
type VarDecl struct {
	Name string
	Tok  lexer.Token // токен-идентификатор
}

func (v *VarDecl) Line() int      { return v.Tok.Line }
func (v *VarDecl) String() string { return fmt.Sprintf("VarDecl(%s : INT)", v.Name) }

// ---------------------------------------------------------------------------
// Операторы (реализуют Statement)
// ---------------------------------------------------------------------------

// AssignStatement — присваивание: `Target := Value;`.
type AssignStatement struct {
	Target string
	Value  Expression
	Tok    lexer.Token // токен-идентификатор слева
}

func (s *AssignStatement) statementNode() {}
func (s *AssignStatement) Line() int      { return s.Tok.Line }
func (s *AssignStatement) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Assign(%s :=)\n", s.Target)
	indent(&b, s.Value.String(), 1)
	return b.String()
}

// IfStatement — `IF Condition THEN Then [ELSE Else] END_IF`.
// Else == nil, если ветки нет.
type IfStatement struct {
	Condition Expression
	Then      []Statement
	Else      []Statement
	Tok       lexer.Token // токен IF
}

func (s *IfStatement) statementNode() {}
func (s *IfStatement) Line() int      { return s.Tok.Line }
func (s *IfStatement) String() string {
	var b strings.Builder
	b.WriteString("If\n")
	indent(&b, "Cond:", 1)
	indent(&b, s.Condition.String(), 2)
	indent(&b, "Then:", 1)
	for _, st := range s.Then {
		indent(&b, st.String(), 2)
	}
	if s.Else != nil {
		indent(&b, "Else:", 1)
		for _, st := range s.Else {
			indent(&b, st.String(), 2)
		}
	}
	return b.String()
}

// ForStatement — `FOR Var := Start TO End DO Body END_FOR`.
type ForStatement struct {
	Var   string
	Start Expression
	End   Expression
	Body  []Statement
	Tok   lexer.Token // токен FOR
}

func (s *ForStatement) statementNode() {}
func (s *ForStatement) Line() int      { return s.Tok.Line }
func (s *ForStatement) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "For(%s)\n", s.Var)
	indent(&b, "Start:", 1)
	indent(&b, s.Start.String(), 2)
	indent(&b, "End:", 1)
	indent(&b, s.End.String(), 2)
	indent(&b, "Do:", 1)
	for _, st := range s.Body {
		indent(&b, st.String(), 2)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Выражения (реализуют Expression)
// ---------------------------------------------------------------------------

// Identifier — ссылка на переменную.
type Identifier struct {
	Name string
	Tok  lexer.Token
}

func (e *Identifier) expressionNode() {}
func (e *Identifier) Line() int       { return e.Tok.Line }
func (e *Identifier) String() string  { return fmt.Sprintf("Ident(%s)", e.Name) }

// IntLiteral — целочисленный литерал.
type IntLiteral struct {
	Value int
	Tok   lexer.Token
}

func (e *IntLiteral) expressionNode() {}
func (e *IntLiteral) Line() int       { return e.Tok.Line }
func (e *IntLiteral) String() string  { return fmt.Sprintf("Int(%d)", e.Value) }

// BinaryExpr — бинарная операция: и арифметика (+ - * /), и сравнения
// (> < =). Различаются полем Op. Приоритет операций задаётся не типом
// узла, а формой дерева, которую строит parser.
type BinaryExpr struct {
	Left  Expression
	Op    lexer.TokenType
	Right Expression
	Tok   lexer.Token // токен-оператор
}

func (e *BinaryExpr) expressionNode() {}
func (e *BinaryExpr) Line() int       { return e.Tok.Line }
func (e *BinaryExpr) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Binary(%s)\n", e.Op)
	indent(&b, e.Left.String(), 1)
	indent(&b, e.Right.String(), 1)
	return b.String()
}

// ---------------------------------------------------------------------------
// Хелпер печати
// ---------------------------------------------------------------------------

// indent добавляет к builder текст s, сдвинутый на depth уровней (по 2
// пробела), с завершающим переводом строки. Многострочный s сдвигается
// целиком построчно.
func indent(b *strings.Builder, s string, depth int) {
	prefix := strings.Repeat("  ", depth)
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteByte('\n')
	}
}
