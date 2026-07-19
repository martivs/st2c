// Package sema — семантический анализ между парсером и (будущим) codegen:
// проверки, без которых сгенерированный C был бы молча некорректен.
//
// Сейчас один уровень области видимости (свой на каждый POU); стек scope-ов
// появится вместе с функциями. Полный вывод типов выражений отложен до
// рабочего codegen — пока проверяются объявленность и диапазон литералов при
// известном целевом типе.
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

// checker — состояние прохода по одному POU. В отличие от fail-fast парсера
// ошибки копятся: пользователь видит все проблемы за один запуск.
type checker struct {
	// syms — символьная таблица; ключ — strings.ToUpper(имя), потому что
	// идентификаторы в IEC 61131-3 регистронезависимы (Sum и sum — одна
	// переменная).
	syms map[string]*Symbol
	errs []error
}

// Check проверяет дерево и возвращает все найденные семантические ошибки
// (nil — ошибок нет).
func Check(sf *ast.SourceFile) []error {
	var errs []error
	for _, pou := range sf.POUs {
		if prog, ok := pou.(*ast.Program); ok {
			c := &checker{syms: map[string]*Symbol{}}
			c.checkProgram(prog)
			errs = append(errs, c.errs...)
		}
	}
	return errs
}

func (c *checker) errorf(tok lexer.Token, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	c.errs = append(c.errs, fmt.Errorf("line %d:%d: %s", tok.Line, tok.Col, msg))
}

func (c *checker) checkProgram(p *ast.Program) {
	// Первый проход — собрать все объявления: инициализаторы проверяются
	// отдельным проходом, чтобы порядок объявлений не влиял на объявленность.
	for _, blk := range p.VarBlocks {
		for _, d := range blk.Decls {
			for _, name := range d.Names {
				key := strings.ToUpper(name)
				if prev, ok := c.syms[key]; ok {
					c.errorf(d.Tok, "duplicate declaration of %q (first declared as %q at line %d)",
						name, prev.Name, prev.Tok.Line)
					continue
				}
				c.syms[key] = &Symbol{Name: name, TypeName: d.TypeName, Tok: d.Tok}
			}
		}
	}
	for _, blk := range p.VarBlocks {
		for _, d := range blk.Decls {
			if d.Init != nil {
				c.checkExpr(d.Init, d.TypeName)
			}
		}
	}
	c.checkStatements(p.Body)
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
			// Парсер пока не пропускает иные lvalue; защитно обходим как
			// выражение, чтобы будущие IndexExpr/MemberExpr не проскочили.
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
	}
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
