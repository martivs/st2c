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
// Операции
// ---------------------------------------------------------------------------

// Op — собственный enum операций AST. Отвязывает дерево от представления
// токенов лексера: семантике и codegen не важно, каким TokenType лексер
// закодировал `<>`.
type Op int

const (
	ADD Op = iota // +
	SUB           // -
	MUL           // *
	DIV           // /
	LT            // <
	LE            // <=
	GT            // >
	GE            // >=
	EQ            // =
	NE            // <>
	NEG           // унарный минус
)

var opNames = map[Op]string{
	ADD: "+", SUB: "-", MUL: "*", DIV: "/",
	LT: "<", LE: "<=", GT: ">", GE: ">=", EQ: "=", NE: "<>",
	NEG: "-",
}

// String возвращает исходный символ операции (`+`, `<=`, …) — те же метки,
// что печатал лексерный TokenType, поэтому golden-эталоны текстово стабильны.
func (o Op) String() string {
	if s, ok := opNames[o]; ok {
		return s
	}
	return "UNKNOWN"
}

// ---------------------------------------------------------------------------
// Корень и объявления
// ---------------------------------------------------------------------------

// POU — program organisation unit (IEC 61131-3): PROGRAM, позже FUNCTION и
// FUNCTION_BLOCK. Маркерный метод pouNode() — по образцу Statement/Expression.
type POU interface {
	Node
	pouNode()
}

// SourceFile — корень дерева: компилируемая единица, список POU. Сейчас в
// файле обычно одна PROGRAM, но форма корня уже не изменится с приходом
// FUNCTION/FUNCTION_BLOCK.
type SourceFile struct {
	POUs []POU
}

func (f *SourceFile) Line() int {
	if len(f.POUs) > 0 {
		return f.POUs[0].Line()
	}
	return 0
}

func (f *SourceFile) String() string {
	var b strings.Builder
	for _, p := range f.POUs {
		b.WriteString(p.String())
	}
	return b.String()
}

// Program — POU-программа: всё, что между PROGRAM и END_PROGRAM.
type Program struct {
	Name      string
	VarBlocks []*VarBlock
	Body      []Statement
	Tok       lexer.Token // токен PROGRAM
}

func (p *Program) pouNode()  {}
func (p *Program) Line() int { return p.Tok.Line }
func (p *Program) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Program(%s)\n", p.Name)
	for _, v := range p.VarBlocks {
		indent(&b, v.String(), 1)
	}
	for _, s := range p.Body {
		indent(&b, s.String(), 1)
	}
	return b.String()
}

// VarKind — вид блока объявлений. Для функций и функциональных блоков виды
// блоков задают интерфейс POU (входы/выходы), поэтому это данные, а не
// синтаксический шум.
type VarKind int

const (
	VarPlain  VarKind = iota // VAR
	VarInput                 // VAR_INPUT
	VarOutput                // VAR_OUTPUT
	VarInOut                 // VAR_IN_OUT
	VarTemp                  // VAR_TEMP
)

var varKindNames = map[VarKind]string{
	VarPlain: "VAR", VarInput: "VAR_INPUT", VarOutput: "VAR_OUTPUT",
	VarInOut: "VAR_IN_OUT", VarTemp: "VAR_TEMP",
}

func (k VarKind) String() string {
	if s, ok := varKindNames[k]; ok {
		return s
	}
	return "UNKNOWN"
}

// VarBlock — один блок объявлений `VAR* ... END_VAR`. Блоков в POU может быть
// несколько и разных видов; необязательные квалификаторы CONSTANT/RETAIN —
// флаги блока.
type VarBlock struct {
	Kind     VarKind
	Constant bool // VAR CONSTANT
	Retain   bool // VAR RETAIN
	Decls    []*VarDecl
	Tok      lexer.Token // токен VAR*
}

func (v *VarBlock) Line() int { return v.Tok.Line }
func (v *VarBlock) String() string {
	var b strings.Builder
	b.WriteString("VarBlock(")
	b.WriteString(v.Kind.String())
	if v.Constant {
		b.WriteString(" CONSTANT")
	}
	if v.Retain {
		b.WriteString(" RETAIN")
	}
	b.WriteString(")\n")
	for _, d := range v.Decls {
		indent(&b, d.String(), 1)
	}
	return b.String()
}

// VarDecl — одно объявление: `a, b, c : INT;` или `x : INT := 5;`.
// TypeName — имя типа как записано (встроенный или пользовательский —
// решает sema, не парсер). Init == nil, если инициализатора нет.
type VarDecl struct {
	Names    []string
	TypeName string
	Init     Expression
	Tok      lexer.Token // токен первого имени
}

func (v *VarDecl) Line() int { return v.Tok.Line }
func (v *VarDecl) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "VarDecl(%s : %s)", strings.Join(v.Names, ", "), v.TypeName)
	if v.Init != nil {
		b.WriteByte('\n')
		indent(&b, "Init:", 1)
		indent(&b, v.Init.String(), 2)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Операторы (реализуют Statement)
// ---------------------------------------------------------------------------

// AssignStatement — присваивание: `Target := Value;`. Target — выражение-
// lvalue: пока это всегда *Identifier, позже сюда лягут IndexExpr/MemberExpr
// (`arr[i] :=`, `fb.out :=`) без смены формы узла.
type AssignStatement struct {
	Target Expression
	Value  Expression
	Tok    lexer.Token // первый токен цели
}

func (s *AssignStatement) statementNode() {}
func (s *AssignStatement) Line() int      { return s.Tok.Line }
func (s *AssignStatement) String() string {
	var b strings.Builder
	b.WriteString("Assign\n")
	indent(&b, "Target:", 1)
	indent(&b, s.Target.String(), 2)
	indent(&b, "Value:", 1)
	indent(&b, s.Value.String(), 2)
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

// ForStatement — `FOR Var := Start TO End [BY Step] DO Body END_FOR`.
// Var по IEC — простой идентификатор; хранится узлом *Identifier ради
// позиции. Step == nil → шаг 1.
type ForStatement struct {
	Var   *Identifier
	Start Expression
	End   Expression
	Step  Expression
	Body  []Statement
	Tok   lexer.Token // токен FOR
}

func (s *ForStatement) statementNode() {}
func (s *ForStatement) Line() int      { return s.Tok.Line }
func (s *ForStatement) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "For(%s)\n", s.Var.Name)
	indent(&b, "Start:", 1)
	indent(&b, s.Start.String(), 2)
	indent(&b, "End:", 1)
	indent(&b, s.End.String(), 2)
	if s.Step != nil {
		indent(&b, "Step:", 1)
		indent(&b, s.Step.String(), 2)
	}
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
// (> < = <= >= <>). Различаются полем Op. Приоритет операций задаётся не
// типом узла, а формой дерева, которую строит parser.
type BinaryExpr struct {
	Left  Expression
	Op    Op
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

// UnaryExpr — унарная операция: `-x`. Сейчас Op всегда NEG; унарный `+` и
// NOT лягут в этот же узел.
type UnaryExpr struct {
	Op      Op
	Operand Expression
	Tok     lexer.Token // токен-оператор
}

func (e *UnaryExpr) expressionNode() {}
func (e *UnaryExpr) Line() int       { return e.Tok.Line }
func (e *UnaryExpr) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Unary(%s)\n", e.Op)
	indent(&b, e.Operand.String(), 1)
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
