package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"st2c/src/ast"
)

// crosses contains the EXITs and switches on paths to this loop that need a flag.
type loopExit struct {
	crosses map[ast.Statement]bool
	flag    string
}

func (l *loopExit) mark(body []ast.Statement, switches []*ast.CaseStatement) {
	for _, stmt := range body {
		switch s := stmt.(type) {
		case *ast.ExitStatement:
			if len(switches) != 0 {
				l.crosses[s] = true
				for i := len(switches) - 1; i >= 0 && !l.crosses[switches[i]]; i-- {
					l.crosses[switches[i]] = true
				}
			}
		case *ast.IfStatement:
			l.mark(s.Then, switches)
			for _, clause := range s.ElsIfs {
				l.mark(clause.Then, switches)
			}
			l.mark(s.Else, switches)
		case *ast.CaseStatement:
			path := append(switches, s)
			for _, branch := range s.Branches {
				l.mark(branch.Body, path)
			}
			l.mark(s.Else, path)
		}
		// Nested loops run this pass for their own bodies.
	}
}

func (g *gen) enterLoop(body []ast.Statement) func() {
	parent := g.loop
	l := &loopExit{crosses: make(map[ast.Statement]bool)}
	g.loop = l
	l.mark(body, nil)
	if len(l.crosses) != 0 {
		l.flag = g.freshName("st2c_exit")
		g.linef("{")
		g.ind++
		g.linef("int %s = 0;", l.flag)
	}
	return func() {
		if l.flag != "" {
			g.ind--
			g.linef("}")
		}
		g.loop = parent
	}
}

func hasCaseRange(branches []ast.CaseBranch) bool {
	for _, branch := range branches {
		for _, label := range branch.Labels {
			if label.High != nil {
				return true
			}
		}
	}
	return false
}

// Emit decimal constants even for ST spellings such as +0008 or -010.
func caseConstant(b ast.CaseBound) (string, error) {
	n, err := strconv.ParseInt(b.Text, 10, 16)
	if err != nil {
		return "", fmt.Errorf("line %d:%d: codegen: invalid CASE bound %q", b.Tok.Line, b.Tok.Col, b.Text)
	}
	return strconv.FormatInt(n, 10), nil
}

func caseCondition(sel string, labels []ast.CaseLabel) (string, error) {
	var tests []string
	for _, label := range labels {
		low, err := caseConstant(label.Low)
		if err != nil {
			return "", err
		}
		if label.High == nil {
			tests = append(tests, fmt.Sprintf("%s == %s", sel, low))
			continue
		}
		high, err := caseConstant(*label.High)
		if err != nil {
			return "", err
		}
		tests = append(tests, fmt.Sprintf("(%s >= %s && %s <= %s)", sel, low, sel, high))
	}
	return strings.Join(tests, " || "), nil
}

func (g *gen) caseStmt(s *ast.CaseStatement) error {
	expr, err := g.expr(s.Selector)
	if err != nil {
		return err
	}
	g.linef("{")
	g.ind++
	sel := g.freshName("st2c_sel")
	g.linef("%s %s = %s;", cWideTypes["INT"], sel, expr)
	key := sel
	ranged := hasCaseRange(s.Branches)
	if ranged {
		key = g.freshName("st2c_arm")
		g.linef("int %s = 0;", key)
		for i, branch := range s.Branches {
			cond, err := caseCondition(sel, branch.Labels)
			if err != nil {
				return err
			}
			keyword := "if"
			if i > 0 {
				keyword = "else if"
			}
			g.linef("%s (%s) %s = %d;", keyword, cond, key, i+1)
		}
	}
	g.linef("switch (%s) {", key)
	for i, branch := range s.Branches {
		if ranged {
			g.linef("case %d:", i+1)
		} else {
			for _, label := range branch.Labels {
				value, err := caseConstant(label.Low)
				if err != nil {
					return err
				}
				g.linef("case %s:", value)
			}
		}
		if err := g.caseBody(branch.Body); err != nil {
			return err
		}
	}
	if s.Else != nil {
		g.linef("default:")
		if err := g.caseBody(s.Else); err != nil {
			return err
		}
	}
	g.linef("}")
	if g.loop != nil && g.loop.crosses[s] {
		g.linef("if (%s) break;", g.loop.flag)
	}
	g.ind--
	g.linef("}")
	return nil
}

func (g *gen) caseBody(body []ast.Statement) error {
	g.linef("{")
	g.ind++
	if err := g.stmts(body); err != nil {
		return err
	}
	g.linef("break;")
	g.ind--
	g.linef("}")
	return nil
}
