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

// unit — один POU файла в порядке исходника; заполнено ровно одно поле.
type unit struct {
	state *stateInfo
	fn    *funcInfo
}

// Generate превращает дерево в текст C-файла. Ошибки — имена и типы,
// непредставимые в C, а также -main без единой PROGRAM в файле.
func Generate(sf *ast.SourceFile, opts Options) (string, error) {
	// Оболочки ФБ строятся до всего остального: экземпляр может быть объявлен
	// выше своего FUNCTION_BLOCK по файлу, а collectVars любого POU должен
	// уметь отличить тип-ФБ от скаляра.
	fbs := map[string]*stateInfo{}
	for _, pou := range sf.POUs {
		if fb, ok := pou.(*ast.FunctionBlock); ok {
			cn, err := mapName(fb.Name, fb.Tok)
			if err != nil {
				return "", err
			}
			fbs[strings.ToUpper(fb.Name)] = &stateInfo{
				stName: fb.Name, cName: cn, stepName: "body",
				blocks: fb.VarBlocks, body: fb.Body,
			}
		}
	}

	var units []unit
	var progs []*stateInfo
	funcs := map[string]*funcInfo{}
	for _, pou := range sf.POUs {
		switch p := pou.(type) {
		case *ast.Program:
			cn, err := mapName(p.Name, p.Tok)
			if err != nil {
				return "", err
			}
			info := &stateInfo{
				stName: p.Name, cName: cn, stepName: "step",
				blocks: p.VarBlocks, body: p.Body,
			}
			if err := info.collectVars(fbs); err != nil {
				return "", err
			}
			units = append(units, unit{state: info})
			progs = append(progs, info)
		case *ast.Function:
			info, err := newFuncInfo(p)
			if err != nil {
				return "", err
			}
			units = append(units, unit{fn: info})
			funcs[strings.ToUpper(p.Name)] = info
		case *ast.FunctionBlock:
			info := fbs[strings.ToUpper(p.Name)]
			if err := info.collectVars(fbs); err != nil {
				return "", err
			}
			units = append(units, unit{state: info})
		}
	}
	if opts.Main && len(progs) == 0 {
		return "", fmt.Errorf("codegen: -main requires a PROGRAM in the source file")
	}

	g := &gen{funcs: funcs}
	g.linef("#include <stdint.h>")
	if opts.Main {
		g.linef("#include <stdio.h>")
	}

	// Два прохода (решение 5): сначала все typedef и прототипы, потом все
	// тела — порядок POU в исходнике и взаимные ссылки перестают иметь
	// значение. Одно исключение: struct владельца содержит struct экземпляра
	// по значению, и typedef вложенного ФБ обязан идти раньше — прототипы
	// этого не решают. declare эмитит зависимости post-order DFS
	// (топологическая сортировка; циклы вложенности отсёк sema на этапе 6).
	declared := map[*stateInfo]bool{}
	var declare func(info *stateInfo) error
	declare = func(info *stateInfo) error {
		if declared[info] {
			return nil
		}
		declared[info] = true
		for _, vi := range info.order {
			if vi.fb != nil {
				if err := declare(vi.fb); err != nil {
					return err
				}
			}
		}
		g.blank()
		if err := g.emitTypedef(info); err != nil {
			return err
		}
		g.linef("void %s_init(%s *self);", info.cName, info.cName)
		g.linef("void %s_%s(%s *self);", info.cName, info.stepName, info.cName)
		return nil
	}
	for _, u := range units {
		if u.fn != nil {
			g.blank()
			sig, err := funcSignature(u.fn)
			if err != nil {
				return "", err
			}
			g.linef("%s;", sig)
			continue
		}
		if err := declare(u.state); err != nil {
			return "", err
		}
	}
	for _, u := range units {
		if u.fn != nil {
			g.blank()
			if err := g.emitFunction(u.fn); err != nil {
				return "", err
			}
			continue
		}
		g.blank()
		if err := g.emitInit(u.state); err != nil {
			return "", err
		}
		g.blank()
		if err := g.emitStep(u.state); err != nil {
			return "", err
		}
	}
	if opts.Main {
		g.blank()
		g.emitDriver(progs[0], opts.Scans)
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
	stName string      // оригинальное написание из объявления (для печати драйвером)
	cName  string      // имя поля в struct
	stType string      // имя ST-типа как объявлено
	fb     *stateInfo  // не nil → экземпляр ФБ: поле-struct, а не скаляр
	tok    lexer.Token
}

// stateInfo — POU с состоянием (PROGRAM или FUNCTION_BLOCK) глазами
// генератора: typedef struct + Name_init + функция шага. Форма у обоих одна
// (решение 2), различается только имя функции шага: _step у PROGRAM,
// _body у ФБ. Ключ vars — strings.ToUpper: ссылка TOTAL на объявленное total
// обязана дать один и тот же C-идентификатор, иначе регистронезависимость
// IEC сломает компиляцию C.
type stateInfo struct {
	stName   string
	cName    string
	stepName string // "step" у PROGRAM, "body" у ФБ
	blocks   []*ast.VarBlock
	body     []ast.Statement
	vars     map[string]*varInfo
	order    []*varInfo // порядок объявления — для struct, _init и драйвера
}

// collectVars строит таблицу переменных POU. Тип объявления, найденный в fbs,
// делает переменную экземпляром ФБ (поле-struct во владельце); остальные
// типы — скаляры, неизвестный скалярный тип отвергнет cType при эмиссии.
func (info *stateInfo) collectVars(fbs map[string]*stateInfo) error {
	info.vars = map[string]*varInfo{}
	used := map[string]string{} // C-имя → ST-имя: ловим склейку после префиксации
	for _, blk := range info.blocks {
		for _, d := range blk.Decls {
			fb := fbs[strings.ToUpper(d.TypeName)]
			for _, name := range d.Names {
				cn, err := mapName(name.Name, name.Tok)
				if err != nil {
					return err
				}
				if prev, dup := used[cn]; dup {
					return fmt.Errorf("line %d:%d: codegen: renamed %q collides with %q (both map to C name %q)",
						name.Tok.Line, name.Tok.Col, name.Name, prev, cn)
				}
				used[cn] = name.Name
				vi := &varInfo{stName: name.Name, cName: cn, stType: d.TypeName, fb: fb, tok: name.Tok}
				info.vars[strings.ToUpper(name.Name)] = vi
				info.order = append(info.order, vi)
			}
		}
	}
	return nil
}

// funcInfo — FUNCTION глазами генератора. Параметры — из блоков VAR_INPUT в
// порядке объявления: порядок задаёт и сигнатуру C-функции, и раскладку
// именованных аргументов в позиционные. Возвратная переменная одноимённа
// функции (возврат по IEC — присваивание её имени: `Add := x + y;`) и в C
// становится локальной, легально затеняющей саму функцию (рекурсию отсекла
// sema). Состояния между вызовами у функции нет — struct не нужен.
type funcInfo struct {
	f      *ast.Function
	cName  string
	retVar *varInfo
	params []*varInfo
	vars   map[string]*varInfo
}

func newFuncInfo(f *ast.Function) (*funcInfo, error) {
	cn, err := mapName(f.Name, f.Tok)
	if err != nil {
		return nil, err
	}
	info := &funcInfo{f: f, cName: cn, vars: map[string]*varInfo{}}
	info.retVar = &varInfo{stName: f.Name, cName: cn, stType: f.ReturnType, tok: f.Tok}
	info.vars[strings.ToUpper(f.Name)] = info.retVar
	used := map[string]string{cn: f.Name}
	for _, blk := range f.VarBlocks {
		switch blk.Kind {
		case ast.VarInput, ast.VarPlain, ast.VarTemp:
		default:
			return nil, fmt.Errorf("line %d:%d: codegen: %s block is not supported in FUNCTION",
				blk.Tok.Line, blk.Tok.Col, blk.Kind)
		}
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
				if blk.Kind == ast.VarInput {
					info.params = append(info.params, vi)
				}
			}
		}
	}
	return info, nil
}

// funcSignature — заголовок C-функции, общий для прототипа и определения.
func funcSignature(info *funcInfo) (string, error) {
	ret, err := cType(info.f.ReturnType, info.f.Tok)
	if err != nil {
		return "", err
	}
	if len(info.params) == 0 {
		return fmt.Sprintf("%s %s(void)", ret, info.cName), nil
	}
	parts := make([]string, len(info.params))
	for i, p := range info.params {
		ct, err := cType(p.stType, p.tok)
		if err != nil {
			return "", err
		}
		parts[i] = ct + " " + p.cName
	}
	return fmt.Sprintf("%s %s(%s)", ret, info.cName, strings.Join(parts, ", ")), nil
}

// ---------------------------------------------------------------------------
// Эмиттер
// ---------------------------------------------------------------------------

// gen — состояние генерации: буфер, отступ, контекст текущего POU (таблица
// имён + признак «переменные живут в struct состояния») и счётчик суффиксов
// временных FOR (уникальность при вложенности).
type gen struct {
	b      strings.Builder
	ind    int
	vars   map[string]*varInfo  // таблица имён текущего POU
	deref  bool                 // true → обращение self->x (PROGRAM, позже ФБ); false → x (FUNCTION)
	funcs  map[string]*funcInfo // функции файла (ключ ToUpper) — для эмиссии вызовов
	forSeq int
}

// ref — C-обращение к переменной с учётом контекста POU: поле struct
// состояния (`self->x`) либо локальная переменная функции (`x`).
func (g *gen) ref(vi *varInfo) string {
	if g.deref {
		return "self->" + vi.cName
	}
	return vi.cName
}

func (g *gen) linef(format string, args ...any) {
	g.b.WriteString(strings.Repeat("    ", g.ind))
	fmt.Fprintf(&g.b, format, args...)
	g.b.WriteByte('\n')
}

func (g *gen) blank() { g.b.WriteByte('\n') }

func (g *gen) emitTypedef(info *stateInfo) error {
	g.linef("typedef struct {")
	g.ind++
	for _, vi := range info.order {
		if vi.fb != nil {
			// Экземпляр ФБ — вложенный struct по значению; его typedef уже
			// эмитнут раньше (топологический порядок в Generate).
			g.linef("%s %s;", vi.fb.cName, vi.cName)
			continue
		}
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
// применяется к каждому имени. Поле-экземпляр ФБ рекурсивно зовёт свой
// _init (инициализатор у экземпляра отвергла sema).
func (g *gen) emitInit(info *stateInfo) error {
	g.vars, g.deref, g.forSeq = info.vars, true, 0
	g.linef("void %s_init(%s *self) {", info.cName, info.cName)
	g.ind++
	for _, blk := range info.blocks {
		for _, d := range blk.Decls {
			for _, name := range d.Names {
				vi := info.vars[strings.ToUpper(name.Name)]
				if vi.fb != nil {
					g.linef("%s_init(&%s);", vi.fb.cName, g.ref(vi))
					continue
				}
				if d.Init == nil {
					g.linef("%s = 0;", g.ref(vi))
					continue
				}
				val, err := g.expr(d.Init)
				if err != nil {
					return err
				}
				g.linef("%s = %s;", g.ref(vi), val)
			}
		}
	}
	g.ind--
	g.linef("}")
	return nil
}

// emitStep — функция шага: тело POU со состоянием (Name_step у PROGRAM,
// Name_body у ФБ); переменные — поля struct, обращение через self->.
func (g *gen) emitStep(info *stateInfo) error {
	g.vars, g.deref, g.forSeq = info.vars, true, 0
	g.linef("void %s_%s(%s *self) {", info.cName, info.stepName, info.cName)
	g.ind++
	if err := g.stmts(info.body); err != nil {
		return err
	}
	g.ind--
	g.linef("}")
	return nil
}

// emitFunction — FUNCTION как обычная C-функция (этап 5). Возвратная
// переменная объявляется первой строкой и возвращается последней; локальные
// из VAR/VAR_TEMP объявляются нулём, затем инициализаторы присваиваются в
// порядке объявления — та же семантика, что у _init (sema разрешает
// `x : INT := y;` до объявления y, инлайн-инициализатор в C тут сломался бы).
// Инициализаторы VAR_INPUT игнорируются: значение параметра приходит от
// вызывающего, все входы функции по sema обязательны (дефолты входов — долг).
func (g *gen) emitFunction(info *funcInfo) error {
	g.vars, g.deref, g.forSeq = info.vars, false, 0
	sig, err := funcSignature(info)
	if err != nil {
		return err
	}
	g.linef("%s {", sig)
	g.ind++
	ret, err := cType(info.retVar.stType, info.retVar.tok)
	if err != nil {
		return err
	}
	g.linef("%s %s = 0;", ret, info.retVar.cName)
	for _, blk := range info.f.VarBlocks {
		if blk.Kind == ast.VarInput {
			continue
		}
		for _, d := range blk.Decls {
			for _, name := range d.Names {
				vi := info.vars[strings.ToUpper(name.Name)]
				ct, err := cType(vi.stType, vi.tok)
				if err != nil {
					return err
				}
				g.linef("%s %s = 0;", ct, vi.cName)
			}
		}
	}
	for _, blk := range info.f.VarBlocks {
		if blk.Kind == ast.VarInput {
			continue
		}
		for _, d := range blk.Decls {
			if d.Init == nil {
				continue
			}
			val, err := g.expr(d.Init)
			if err != nil {
				return err
			}
			for _, name := range d.Names {
				g.linef("%s = %s;", info.vars[strings.ToUpper(name.Name)].cName, val)
			}
		}
	}
	if err := g.stmts(info.f.Body); err != nil {
		return err
	}
	g.linef("return %s;", info.retVar.cName)
	g.ind--
	g.linef("}")
	return nil
}

// emitDriver — main() по решению 4: _init, затем Scans вызовов _step, после
// каждого — печать всех скалярных полей PROGRAM в порядке объявления, по
// строке `имя=значение` с оригинальным ST-именем. Поля-экземпляры ФБ не
// разворачиваются: внутреннее состояние выводится наружу явно (`=>` или
// присваивание из inst.member).
func (g *gen) emitDriver(info *stateInfo, scans int) {
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
		if vi.fb != nil {
			continue
		}
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

	case *ast.CallStatement:
		return g.callStmt(s)

	default:
		return fmt.Errorf("line %d: codegen: unsupported statement %T", s.Line(), s)
	}
}

// callStmt — вызов ФБ как оператор (этап 7): разворот в присваивания входов
// (в порядке аргументов), вызов тела и выгрузку выходов (в порядке
// аргументов). Что callee — экземпляр, аргументы только именованные, имена
// существуют и направление верное — гарантирует sema; дыры здесь внутренние
// ошибки.
func (g *gen) callStmt(s *ast.CallStatement) error {
	call := s.Call
	id, ok := call.Callee.(*ast.Identifier)
	if !ok {
		return fmt.Errorf("line %d: codegen: unsupported call statement target %T", call.Line(), call.Callee)
	}
	vi := g.vars[strings.ToUpper(id.Name)]
	if vi == nil || vi.fb == nil {
		return fmt.Errorf("line %d: codegen: internal: %q is not a function block instance (sema must reject this)",
			call.Line(), id.Name)
	}
	inst := g.ref(vi)
	for _, a := range call.Args {
		if a.Output {
			continue
		}
		m := vi.fb.vars[strings.ToUpper(a.Name)]
		if a.Name == "" || m == nil {
			return fmt.Errorf("line %d: codegen: internal: bad input binding %q in call of %q (sema must reject this)",
				call.Line(), a.Name, id.Name)
		}
		val, err := g.expr(a.Value)
		if err != nil {
			return err
		}
		g.linef("%s.%s = %s;", inst, m.cName, val)
	}
	g.linef("%s_%s(&%s);", vi.fb.cName, vi.fb.stepName, inst)
	for _, a := range call.Args {
		if !a.Output {
			continue
		}
		m := vi.fb.vars[strings.ToUpper(a.Name)]
		if m == nil {
			return fmt.Errorf("line %d: codegen: internal: bad output binding %q in call of %q (sema must reject this)",
				call.Line(), a.Name, id.Name)
		}
		target, err := g.expr(a.Value)
		if err != nil {
			return err
		}
		g.linef("%s = %s.%s;", target, inst, m.cName)
	}
	return nil
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
	vi := g.vars[strings.ToUpper(s.Var.Name)]
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
	g.linef("%s = (%s)%s;", g.ref(vi), narrow, i)
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
		vi := g.vars[strings.ToUpper(e.Name)]
		if vi == nil {
			return "", fmt.Errorf("line %d: codegen: internal: undeclared variable %q (sema must reject this)",
				e.Line(), e.Name)
		}
		return g.ref(vi), nil

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

	case *ast.CallExpr:
		return g.call(e)

	case *ast.MemberExpr:
		// Доступ к члену экземпляра ФБ: чтение inst.Out в выражении и цель
		// присваивания inst.In := … (обе формы идут через expr). Что база —
		// экземпляр, член существует и направление верное — проверила sema.
		id, ok := e.Base.(*ast.Identifier)
		if !ok {
			return "", fmt.Errorf("line %d: codegen: unsupported member access base %T", e.Line(), e.Base)
		}
		vi := g.vars[strings.ToUpper(id.Name)]
		if vi == nil || vi.fb == nil {
			return "", fmt.Errorf("line %d: codegen: internal: %q is not a function block instance (sema must reject this)",
				e.Line(), id.Name)
		}
		m := vi.fb.vars[strings.ToUpper(e.Member)]
		if m == nil {
			return "", fmt.Errorf("line %d: codegen: internal: %q has no member %q (sema must reject this)",
				e.Line(), id.Name, e.Member)
		}
		return g.ref(vi) + "." + m.cName, nil

	default:
		return "", fmt.Errorf("line %d: codegen: unsupported expression %T", e.Line(), e)
	}
}

// call — вызов функции в выражении: именованные аргументы раскладываются в
// позиционные по порядку VAR_INPUT. Полноту, уникальность привязок и порядок
// «позиционные раньше именованных» гарантирует sema — дыры здесь внутренние
// ошибки. Собственных скобок вызову не нужно: это первичное выражение.
func (g *gen) call(e *ast.CallExpr) (string, error) {
	id, ok := e.Callee.(*ast.Identifier)
	if !ok {
		return "", fmt.Errorf("line %d: codegen: unsupported call target %T", e.Line(), e.Callee)
	}
	info, ok := g.funcs[strings.ToUpper(id.Name)]
	if !ok {
		return "", fmt.Errorf("line %d: codegen: internal: call of unknown function %q (sema must reject this)",
			e.Line(), id.Name)
	}
	index := map[string]int{}
	for i, p := range info.params {
		index[strings.ToUpper(p.stName)] = i
	}
	args := make([]string, len(info.params))
	bound := make([]bool, len(info.params))
	for i, a := range e.Args {
		if a.Output {
			return "", fmt.Errorf("line %d: codegen: internal: output binding in call of function %q (sema must reject this)",
				e.Line(), id.Name)
		}
		val, err := g.expr(a.Value)
		if err != nil {
			return "", err
		}
		slot := i
		if a.Name != "" {
			s, known := index[strings.ToUpper(a.Name)]
			if !known {
				return "", fmt.Errorf("line %d: codegen: internal: function %q has no input %q (sema must reject this)",
					e.Line(), id.Name, a.Name)
			}
			slot = s
		}
		if slot >= len(args) || bound[slot] {
			return "", fmt.Errorf("line %d: codegen: internal: bad argument binding in call of %q (sema must reject this)",
				e.Line(), id.Name)
		}
		args[slot] = val
		bound[slot] = true
	}
	for i := range bound {
		if !bound[i] {
			return "", fmt.Errorf("line %d: codegen: internal: input %q of %q not bound (sema must reject this)",
				e.Line(), info.params[i].stName, id.Name)
		}
	}
	return info.cName + "(" + strings.Join(args, ", ") + ")", nil
}
