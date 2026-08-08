// Package codegen — генерация C из проверенного sema дерева.
//
// Обход рекурсивный, по образцу String() из пакета ast. Генератор
// предполагает, что sema.Check прошёл без ошибок: необъявленная переменная
// здесь — внутренняя ошибка, а не диагностика пользователю. Собственные
// проверки codegen — только то, что фронтенд пропускает молча: имена,
// непредставимые в C (решение 8), и типы, для которых нет маппинга.
package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"st2c/src/ast"
	"st2c/src/lexer"
)

// Options — режим генерации.
type Options struct {
	Main  bool // дописать в конец файла драйвер main() (флаг -main)
	Scans int  // сколько раз драйвер вызывает _step (флаг -scans; <=0 → 1)
}

// Generate превращает дерево в текст C-файла. Ошибки — имена и типы,
// непредставимые в C, а также -main без единой PROGRAM в файле.
func Generate(sf *ast.SourceFile, opts Options) (string, error) {
	var infos []*progInfo
	for _, pou := range sf.POUs {
		p, ok := pou.(*ast.Program)
		if !ok {
			return "", fmt.Errorf("line %d: codegen: unsupported POU %T", pou.Line(), pou)
		}
		info, err := newProgInfo(p)
		if err != nil {
			return "", err
		}
		infos = append(infos, info)
	}
	if opts.Main && len(infos) == 0 {
		return "", fmt.Errorf("codegen: -main requires a PROGRAM in the source file")
	}

	g := &gen{}
	g.linef("#include <stdint.h>")
	if opts.Main {
		g.linef("#include <stdio.h>")
	}

	// Два прохода (решение 5): сначала все typedef и прототипы, потом все
	// тела — порядок POU в исходнике и взаимные ссылки перестают иметь
	// значение.
	for _, info := range infos {
		g.blank()
		if err := g.emitTypedef(info); err != nil {
			return "", err
		}
		g.linef("void %s_init(%s *self);", info.cName, info.cName)
		g.linef("void %s_step(%s *self);", info.cName, info.cName)
	}
	for _, info := range infos {
		g.blank()
		if err := g.emitInit(info); err != nil {
			return "", err
		}
		g.blank()
		if err := g.emitStep(info); err != nil {
			return "", err
		}
	}
	if opts.Main {
		g.blank()
		g.emitDriver(infos[0], opts.Scans)
	}
	return g.b.String(), nil
}

// ---------------------------------------------------------------------------
// Типы: единственная точка маппинга ST → C
// ---------------------------------------------------------------------------

// cTypes — таблица маппинга типов; литерала "int16_t" нет больше нигде.
// Ключ — верхний регистр (имена типов IEC регистронезависимы).
var cTypes = map[string]string{
	"INT": "int16_t",
}

// cWideTypes — тип счётчика FOR: шире переменной цикла, чтобы прибавление
// шага у границы диапазона не переполнялось (решение 6).
var cWideTypes = map[string]string{
	"INT": "int32_t",
}

func cType(typeName string, tok lexer.Token) (string, error) {
	if t, ok := cTypes[strings.ToUpper(typeName)]; ok {
		return t, nil
	}
	return "", fmt.Errorf("line %d:%d: codegen: no C mapping for type %q", tok.Line, tok.Col, typeName)
}

func cWideType(typeName string, tok lexer.Token) (string, error) {
	if t, ok := cWideTypes[strings.ToUpper(typeName)]; ok {
		return t, nil
	}
	return "", fmt.Errorf("line %d:%d: codegen: no C mapping for type %q", tok.Line, tok.Col, typeName)
}

// ---------------------------------------------------------------------------
// Имена: маппер ST → C (решение 8)
// ---------------------------------------------------------------------------

// reservedCNames — имена, которые нельзя выпускать в C как есть: ключевые
// слова C99, не являющиеся ключевыми словами ST (int, for, if, else, do —
// ключевые и в ST, переменными быть не могут); имена из <stdint.h>; main
// (коллизия с драйвером -main); self (параметр функций init/step).
var reservedCNames = map[string]bool{
	// C99
	"auto": true, "break": true, "case": true, "char": true, "const": true,
	"continue": true, "default": true, "double": true, "enum": true,
	"extern": true, "float": true, "goto": true, "inline": true, "long": true,
	"register": true, "restrict": true, "return": true, "short": true,
	"signed": true, "sizeof": true, "static": true, "struct": true,
	"switch": true, "typedef": true, "union": true, "unsigned": true,
	"void": true, "volatile": true, "while": true,
	"_Bool": true, "_Complex": true, "_Imaginary": true,
	// <stdint.h>
	"int8_t": true, "int16_t": true, "int32_t": true, "int64_t": true,
	"uint8_t": true, "uint16_t": true, "uint32_t": true, "uint64_t": true,
	"intmax_t": true, "uintmax_t": true, "intptr_t": true, "uintptr_t": true,
	// окружение генератора
	"main": true, "self": true,
}

// mapName возвращает C-имя для ST-идентификатора. Совпадение с таблицей или
// префикс `__` (в C зарезервирован за реализацией; из таких имён генератор
// строит временные FOR) → префикс st_. Не-ASCII имя — ошибка: валидный
// C-идентификатор из него не составить, транслитерация отложена (долг).
func mapName(name string, tok lexer.Token) (string, error) {
	for _, r := range name {
		ok := r == '_' || (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !ok {
			return "", fmt.Errorf("line %d:%d: codegen: identifier %q is not representable in C (non-ASCII character %q)",
				tok.Line, tok.Col, name, r)
		}
	}
	if reservedCNames[name] || strings.HasPrefix(name, "__") {
		return "st_" + name, nil
	}
	return name, nil
}

// varInfo — переменная POU глазами генератора.
type varInfo struct {
	stName string // оригинальное написание из объявления (для печати драйвером)
	cName  string // имя поля в struct
	stType string // имя ST-типа как объявлено
	tok    lexer.Token
}

// progInfo — PROGRAM с построенной таблицей имён. Ключ vars —
// strings.ToUpper: ссылка TOTAL на объявленное total обязана дать один и тот
// же C-идентификатор, иначе регистронезависимость IEC сломает компиляцию C.
type progInfo struct {
	p     *ast.Program
	cName string
	vars  map[string]*varInfo
	order []*varInfo // порядок объявления — для struct, _init и драйвера
}

func newProgInfo(p *ast.Program) (*progInfo, error) {
	cn, err := mapName(p.Name, p.Tok)
	if err != nil {
		return nil, err
	}
	info := &progInfo{p: p, cName: cn, vars: map[string]*varInfo{}}
	used := map[string]string{} // C-имя → ST-имя: ловим склейку после префиксации
	for _, blk := range p.VarBlocks {
		for _, d := range blk.Decls {
			for _, name := range d.Names {
				cn, err := mapName(name.Name, name.Tok)
				if err != nil {
					return nil, err
				}
				if prev, dup := used[cn]; dup {
					return nil, fmt.Errorf("line %d:%d: codegen: renamed %q collides with %q (both map to C name %q)",
						name.Tok.Line, name.Tok.Col, name.Name, prev, cn)
				}
				used[cn] = name.Name
				vi := &varInfo{stName: name.Name, cName: cn, stType: d.TypeName, tok: name.Tok}
				info.vars[strings.ToUpper(name.Name)] = vi
				info.order = append(info.order, vi)
			}
		}
	}
	return info, nil
}

// ---------------------------------------------------------------------------
// Эмиттер
// ---------------------------------------------------------------------------

// gen — состояние генерации: буфер, отступ, таблица имён текущего POU и
// счётчик суффиксов временных FOR (уникальность при вложенности).
type gen struct {
	b      strings.Builder
	ind    int
	cur    *progInfo
	forSeq int
}

func (g *gen) linef(format string, args ...any) {
	g.b.WriteString(strings.Repeat("    ", g.ind))
	fmt.Fprintf(&g.b, format, args...)
	g.b.WriteByte('\n')
}

func (g *gen) blank() { g.b.WriteByte('\n') }

func (g *gen) emitTypedef(info *progInfo) error {
	g.linef("typedef struct {")
	g.ind++
	for _, vi := range info.order {
		ct, err := cType(vi.stType, vi.tok)
		if err != nil {
			return err
		}
		g.linef("%s %s;", ct, vi.cName)
	}
	g.ind--
	g.linef("} %s;", info.cName)
	return nil
}

// emitInit — Name_init (решение 3): инициализаторы — присваиваниями, без
// инициализатора — явный 0 (сгенерированный C не зависит от содержимого
// памяти вызывающего). Инициализатор объявления-списка (a, b, c : INT := 7)
// применяется к каждому имени.
func (g *gen) emitInit(info *progInfo) error {
	g.cur, g.forSeq = info, 0
	g.linef("void %s_init(%s *self) {", info.cName, info.cName)
	g.ind++
	for _, blk := range info.p.VarBlocks {
		for _, d := range blk.Decls {
			for _, name := range d.Names {
				vi := info.vars[strings.ToUpper(name.Name)]
				if d.Init == nil {
					g.linef("self->%s = 0;", vi.cName)
					continue
				}
				val, err := g.expr(d.Init)
				if err != nil {
					return err
				}
				g.linef("self->%s = %s;", vi.cName, val)
			}
		}
	}
	g.ind--
	g.linef("}")
	return nil
}

func (g *gen) emitStep(info *progInfo) error {
	g.cur, g.forSeq = info, 0
	g.linef("void %s_step(%s *self) {", info.cName, info.cName)
	g.ind++
	if err := g.stmts(info.p.Body); err != nil {
		return err
	}
	g.ind--
	g.linef("}")
	return nil
}

// emitDriver — main() по решению 4: _init, затем Scans вызовов _step, после
// каждого — печать всех скалярных полей PROGRAM в порядке объявления, по
// строке `имя=значение` с оригинальным ST-именем.
func (g *gen) emitDriver(info *progInfo, scans int) {
	if scans <= 0 {
		scans = 1
	}
	g.linef("int main(void) {")
	g.ind++
	g.linef("%s st;", info.cName)
	g.linef("%s_init(&st);", info.cName)
	g.linef("for (int scan = 0; scan < %d; ++scan) {", scans)
	g.ind++
	g.linef("%s_step(&st);", info.cName)
	for _, vi := range info.order {
		g.linef("printf(\"%s=%%d\\n\", st.%s);", vi.stName, vi.cName)
	}
	g.ind--
	g.linef("}")
	g.linef("return 0;")
	g.ind--
	g.linef("}")
}

// ---------------------------------------------------------------------------
// Операторы
// ---------------------------------------------------------------------------

func (g *gen) stmts(list []ast.Statement) error {
	for _, s := range list {
		if err := g.stmt(s); err != nil {
			return err
		}
	}
	return nil
}

func (g *gen) stmt(s ast.Statement) error {
	switch s := s.(type) {
	case *ast.AssignStatement:
		target, err := g.expr(s.Target)
		if err != nil {
			return err
		}
		val, err := g.expr(s.Value)
		if err != nil {
			return err
		}
		g.linef("%s = %s;", target, val)
		return nil

	case *ast.IfStatement:
		cond, err := g.expr(s.Condition)
		if err != nil {
			return err
		}
		g.linef("if (%s) {", cond)
		g.ind++
		if err := g.stmts(s.Then); err != nil {
			return err
		}
		g.ind--
		if s.Else != nil {
			g.linef("} else {")
			g.ind++
			if err := g.stmts(s.Else); err != nil {
				return err
			}
			g.ind--
		}
		g.linef("}")
		return nil

	case *ast.ForStatement:
		return g.forStmt(s)

	default:
		return fmt.Errorf("line %d: codegen: unsupported statement %T", s.Line(), s)
	}
}

// constStepSign — знак шага FOR, когда он известен без вычисления: шаг
// опущен (→ 1), литерал, унарный минус над литералом. Общей свёртки констант
// в проекте нет (долг), поэтому, например, BY -(1) остаётся «неизвестным».
func constStepSign(e ast.Expression) (neg bool, ok bool) {
	switch e := e.(type) {
	case nil:
		return false, true
	case *ast.IntLiteral:
		return e.Value < 0, true
	case *ast.UnaryExpr:
		if lit, isLit := e.Operand.(*ast.IntLiteral); isLit && e.Op == ast.NEG {
			return lit.Value > 0, true
		}
	}
	return false, false
}

// forStmt — FOR по решению 6. Счётчик и границы — во временных типа шире
// переменной цикла: границы вычисляются один раз до входа (семантика IEC), а
// прибавление шага у границы диапазона INT не переполняется (наивный int16_t
// давал бы бесконечный цикл на 32767). Переменная цикла получает копию
// счётчика в начале каждой итерации; изменение её в теле на число итераций
// не влияет (по IEC так нельзя, диагностики нет — долг). Направление —
// тернарником по знаку шага; при константном шаге тернарник свёрнут в
// <= / >= (шаг 0 идёт по ветке >= 0 — как и в тернарнике).
func (g *gen) forStmt(s *ast.ForStatement) error {
	vi := g.cur.vars[strings.ToUpper(s.Var.Name)]
	if vi == nil {
		return fmt.Errorf("line %d: codegen: internal: undeclared FOR variable %q (sema must reject this)",
			s.Var.Line(), s.Var.Name)
	}
	wide, err := cWideType(vi.stType, vi.tok)
	if err != nil {
		return err
	}
	narrow, err := cType(vi.stType, vi.tok)
	if err != nil {
		return err
	}
	start, err := g.expr(s.Start)
	if err != nil {
		return err
	}
	end, err := g.expr(s.End)
	if err != nil {
		return err
	}
	step := "1"
	if s.Step != nil {
		if step, err = g.expr(s.Step); err != nil {
			return err
		}
	}

	n := g.forSeq
	g.forSeq++
	i, e, st := fmt.Sprintf("__i%d", n), fmt.Sprintf("__end%d", n), fmt.Sprintf("__step%d", n)
	g.linef("{")
	g.ind++
	g.linef("%s %s = %s, %s = %s, %s = %s;", wide, i, start, e, end, st, step)
	cond := fmt.Sprintf("%s >= 0 ? %s <= %s : %s >= %s", st, i, e, i, e)
	if neg, known := constStepSign(s.Step); known {
		if neg {
			cond = fmt.Sprintf("%s >= %s", i, e)
		} else {
			cond = fmt.Sprintf("%s <= %s", i, e)
		}
	}
	g.linef("for (; %s; %s += %s) {", cond, i, st)
	g.ind++
	g.linef("self->%s = (%s)%s;", vi.cName, narrow, i)
	if err := g.stmts(s.Body); err != nil {
		return err
	}
	g.ind--
	g.linef("}")
	g.ind--
	g.linef("}")
	return nil
}

// ---------------------------------------------------------------------------
// Выражения
// ---------------------------------------------------------------------------

// cOps — таблица ast.Op → оператор C: расходятся только EQ и NE.
var cOps = map[ast.Op]string{
	ast.ADD: "+", ast.SUB: "-", ast.MUL: "*", ast.DIV: "/",
	ast.LT: "<", ast.LE: "<=", ast.GT: ">", ast.GE: ">=",
	ast.EQ: "==", ast.NE: "!=",
}

// expr возвращает C-текст выражения. Каждая бинарная и унарная операция — в
// собственных скобках: надёжность важнее читаемости C-выхода (решение из
// CLAUDE.md).
func (g *gen) expr(e ast.Expression) (string, error) {
	switch e := e.(type) {
	case *ast.Identifier:
		vi := g.cur.vars[strings.ToUpper(e.Name)]
		if vi == nil {
			return "", fmt.Errorf("line %d: codegen: internal: undeclared variable %q (sema must reject this)",
				e.Line(), e.Name)
		}
		return "self->" + vi.cName, nil

	case *ast.IntLiteral:
		return strconv.Itoa(e.Value), nil

	case *ast.BinaryExpr:
		left, err := g.expr(e.Left)
		if err != nil {
			return "", err
		}
		right, err := g.expr(e.Right)
		if err != nil {
			return "", err
		}
		op, ok := cOps[e.Op]
		if !ok {
			return "", fmt.Errorf("line %d: codegen: unsupported operator %s", e.Line(), e.Op)
		}
		return fmt.Sprintf("(%s %s %s)", left, op, right), nil

	case *ast.UnaryExpr:
		if e.Op != ast.NEG {
			return "", fmt.Errorf("line %d: codegen: unsupported unary operator %s", e.Line(), e.Op)
		}
		operand, err := g.expr(e.Operand)
		if err != nil {
			return "", err
		}
		return "(-" + operand + ")", nil

	default:
		return "", fmt.Errorf("line %d: codegen: unsupported expression %T", e.Line(), e)
	}
}
