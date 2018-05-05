// Copyright (c) 2016, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package syntax

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

// Indent sets the number of spaces used for indentation. If set to 0,
// tabs will be used instead.
func Indent(spaces uint) func(*Printer) {
	return func(p *Printer) { p.indentSpaces = spaces }
}

// BinaryNextLine will make binary operators appear on the next line
// when a binary command, such as a pipe, spans multiple lines. A
// backslash will be used.
func BinaryNextLine(p *Printer) { p.binNextLine = true }

// SwitchCaseIndent will make switch cases be indented. As such, switch
// case bodies will be two levels deeper than the switch itself.
func SwitchCaseIndent(p *Printer) { p.swtCaseIndent = true }

// KeepPadding will keep most nodes and tokens in the same column that
// they were in the original source. This allows the user to decide how
// to align and pad their code with spaces.
//
// Note that this feature is best-effort and will only keep the
// alignment stable, so it may need some human help the first time it is
// run.
func KeepPadding(p *Printer) {
	p.keepPadding = true
	p.cols.Writer = p.bufWriter.(*bufio.Writer)
	p.bufWriter = &p.cols
}

// Minify will print programs in a way to save the most bytes possible.
// For example, indentation and comments are skipped, and extra
// whitespace is avoided when possible.
func Minify(p *Printer) { p.minify = true }

// NewPrinter allocates a new Printer and applies any number of options.
func NewPrinter(options ...func(*Printer)) *Printer {
	p := &Printer{
		bufWriter:  bufio.NewWriter(nil),
		lenPrinter: new(Printer),
	}
	for _, opt := range options {
		opt(p)
	}
	return p
}

// Print "pretty-prints" the given AST file to the given writer. Writes
// to w are buffered.
func (p *Printer) Print(w io.Writer, f *File) error {
	p.reset()
	p.bufWriter.Reset(w)
	p.stmts(f.StmtList)
	p.newline(Pos{})
	return p.bufWriter.Flush()
}

type bufWriter interface {
	WriteByte(byte) error
	WriteString(string) (int, error)
	Reset(io.Writer)
	Flush() error
}

type colCounter struct {
	*bufio.Writer
	column int
}

func (c *colCounter) WriteByte(b byte) error {
	if b == '\n' {
		c.column = 1
	} else {
		c.column++
	}
	return c.Writer.WriteByte(b)
}

func (c *colCounter) WriteString(s string) (int, error) {
	for _, r := range s {
		if r == '\n' {
			c.column = 1
		} else {
			c.column++
		}
	}
	return c.Writer.WriteString(s)
}

func (c *colCounter) Reset(w io.Writer) {
	c.column = 1
	c.Writer.Reset(w)
}

// Printer holds the internal state of the printing mechanism of a
// program.
type Printer struct {
	bufWriter
	cols colCounter

	indentSpaces  uint
	binNextLine   bool
	swtCaseIndent bool
	keepPadding   bool
	minify        bool

	wantSpace   bool
	wantNewline bool
	wroteSemi   bool

	commentPadding uint

	// line is the current line number
	line uint

	// lastLevel is the last level of indentation that was used.
	lastLevel uint
	// level is the current level of indentation.
	level uint
	// levelIncs records which indentation level increments actually
	// took place, to revert them once their section ends.
	levelIncs []bool

	nestedBinary bool

	// pendingHdocs is the list of pending heredocs to write.
	pendingHdocs []*Redirect

	// used in stmtCols to align comments
	lenPrinter *Printer
	lenCounter byteCounter
}

func (p *Printer) reset() {
	p.wantSpace, p.wantNewline = false, false
	p.commentPadding = 0
	p.line = 0
	p.lastLevel, p.level = 0, 0
	p.levelIncs = p.levelIncs[:0]
	p.nestedBinary = false
	p.pendingHdocs = p.pendingHdocs[:0]
}

func (p *Printer) spaces(n uint) {
	for i := uint(0); i < n; i++ {
		p.WriteByte(' ')
	}
}

func (p *Printer) space() {
	p.WriteByte(' ')
	p.wantSpace = false
}

func (p *Printer) spacePad(pos Pos) {
	if p.wantSpace {
		p.WriteByte(' ')
		p.wantSpace = false
	}
	for p.cols.column > 0 && p.cols.column < int(pos.col) {
		p.WriteByte(' ')
	}
}

func (p *Printer) bslashNewl() {
	if p.wantSpace {
		p.space()
	}
	p.WriteString("\\\n")
	p.line++
	p.indent()
}

func (p *Printer) spacedString(s string, pos Pos) {
	p.spacePad(pos)
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) spacedToken(s string, pos Pos) {
	if p.minify {
		p.WriteString(s)
		p.wantSpace = false
		return
	}
	p.spacePad(pos)
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) semiOrNewl(s string, pos Pos) {
	if p.wantNewline {
		p.newline(pos)
		p.indent()
	} else {
		if !p.wroteSemi {
			p.WriteByte(';')
		}
		if !p.minify {
			p.space()
		}
		p.line = pos.Line()
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) incLevel() {
	inc := false
	if p.level <= p.lastLevel || len(p.levelIncs) == 0 {
		p.level++
		inc = true
	} else if last := &p.levelIncs[len(p.levelIncs)-1]; *last {
		*last = false
		inc = true
	}
	p.levelIncs = append(p.levelIncs, inc)
}

func (p *Printer) decLevel() {
	if p.levelIncs[len(p.levelIncs)-1] {
		p.level--
	}
	p.levelIncs = p.levelIncs[:len(p.levelIncs)-1]
}

func (p *Printer) indent() {
	if p.minify {
		return
	}
	p.lastLevel = p.level
	switch {
	case p.level == 0:
	case p.indentSpaces == 0:
		for i := uint(0); i < p.level; i++ {
			p.WriteByte('\t')
		}
	default:
		p.spaces(p.indentSpaces * p.level)
	}
}

func (p *Printer) newline(pos Pos) {
	p.wantNewline, p.wantSpace = false, false
	p.WriteByte('\n')
	if p.line < pos.Line() {
		p.line++
	}
	hdocs := p.pendingHdocs
	p.pendingHdocs = p.pendingHdocs[:0]
	for _, r := range hdocs {
		if r.Hdoc != nil {
			p.word(r.Hdoc)
			p.line = r.Hdoc.End().Line()
		}
		p.unquotedWord(r.Word)
		p.line++
		p.WriteByte('\n')
		p.wantSpace = false
	}
}

func (p *Printer) newlines(pos Pos) {
	p.newline(pos)
	if pos.Line() > p.line {
		if !p.minify {
			// preserve single empty lines
			p.WriteByte('\n')
		}
		p.line++
	}
	p.indent()
}

func (p *Printer) rightParen(pos Pos) {
	if p.minify {
	} else if p.wantNewline || pos.Line() > p.line {
		p.newlines(pos)
	}
	p.WriteByte(')')
	p.wantSpace = true
}

func (p *Printer) semiRsrv(s string, pos Pos, fallback bool) {
	if p.wantNewline || pos.Line() > p.line {
		p.newlines(pos)
	} else {
		if fallback && !p.wroteSemi {
			p.WriteByte(';')
		}
		if !p.minify {
			p.spacePad(pos)
		}
	}
	p.WriteString(s)
	p.wantSpace = true
}

func (p *Printer) comment(c Comment) {
	if p.minify {
		return
	}
	switch {
	case p.line == 0:
	case c.Hash.Line() > p.line:
		p.newlines(c.Hash)
	case p.wantSpace:
		if p.keepPadding {
			p.spacePad(c.Pos())
		} else {
			p.spaces(p.commentPadding + 1)
		}
	}
	p.line = c.Hash.Line()
	p.WriteByte('#')
	p.WriteString(strings.TrimRightFunc(c.Text, unicode.IsSpace))
}

func (p *Printer) comments(cs []Comment) {
	for _, c := range cs {
		p.comment(c)
	}
}

func (p *Printer) wordParts(wps []WordPart) {
	for i, n := range wps {
		var next WordPart
		if i+1 < len(wps) {
			next = wps[i+1]
		}
		p.wordPart(n, next)
	}
}

func (p *Printer) wordPart(wp, next WordPart) {
	switch x := wp.(type) {
	case *Lit:
		p.WriteString(x.Value)
	case *SglQuoted:
		if x.Dollar {
			p.WriteByte('$')
		}
		p.WriteByte('\'')
		p.WriteString(x.Value)
		p.WriteByte('\'')
		p.line = x.End().Line()
	case *DblQuoted:
		p.dblQuoted(x)
	case *CmdSubst:
		p.line = x.Pos().Line()
		switch {
		case x.TempFile:
			p.WriteString("${")
			p.wantSpace = true
			p.nestedStmts(x.StmtList, x.Right)
			p.wantSpace = false
			p.semiRsrv("}", x.Right, true)
		case x.ReplyVar:
			p.WriteString("${|")
			p.nestedStmts(x.StmtList, x.Right)
			p.wantSpace = false
			p.semiRsrv("}", x.Right, true)
		default:
			p.WriteString("$(")
			p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
			p.nestedStmts(x.StmtList, x.Right)
			p.rightParen(x.Right)
		}
	case *ParamExp:
		nextLit, ok := next.(*Lit)
		litCont := ";"
		if ok {
			litCont = nextLit.Value[:1]
		}
		if p.minify && !x.Excl && !x.Length && !x.Width &&
			x.Index == nil && x.Slice == nil && x.Repl == nil &&
			x.Exp == nil && !ValidName(x.Param.Value+litCont) {
			x2 := *x
			x2.Short = true
			p.paramExp(&x2)
			return
		}
		p.paramExp(x)
	case *ArithmExp:
		p.WriteString("$((")
		if x.Unsigned {
			p.WriteString("# ")
		}
		p.arithmExpr(x.X, false, false)
		p.WriteString("))")
	case *ExtGlob:
		p.WriteString(x.Op.String())
		p.WriteString(x.Pattern.Value)
		p.WriteByte(')')
	case *ProcSubst:
		// avoid conflict with << and others
		if p.wantSpace {
			p.space()
		}
		p.WriteString(x.Op.String())
		p.nestedStmts(x.StmtList, Pos{})
		p.WriteByte(')')
	}
}

func (p *Printer) dblQuoted(dq *DblQuoted) {
	if dq.Dollar {
		p.WriteByte('$')
	}
	p.WriteByte('"')
	if len(dq.Parts) > 0 {
		p.wordParts(dq.Parts)
		p.line = dq.Parts[len(dq.Parts)-1].End().Line()
	}
	p.WriteByte('"')
}

func (p *Printer) wroteIndex(index ArithmExpr) bool {
	if index == nil {
		return false
	}
	p.WriteByte('[')
	p.arithmExpr(index, false, false)
	p.WriteByte(']')
	return true
}

func (p *Printer) paramExp(pe *ParamExp) {
	if pe.nakedIndex() { // arr[x]
		p.WriteString(pe.Param.Value)
		p.wroteIndex(pe.Index)
		return
	}
	if pe.Short { // $var
		p.WriteByte('$')
		p.WriteString(pe.Param.Value)
		return
	}
	// ${var...}
	p.WriteString("${")
	switch {
	case pe.Length:
		p.WriteByte('#')
	case pe.Width:
		p.WriteByte('%')
	case pe.Excl:
		p.WriteByte('!')
	}
	p.WriteString(pe.Param.Value)
	p.wroteIndex(pe.Index)
	if pe.Slice != nil {
		p.WriteByte(':')
		p.arithmExpr(pe.Slice.Offset, true, true)
		if pe.Slice.Length != nil {
			p.WriteByte(':')
			p.arithmExpr(pe.Slice.Length, true, false)
		}
	} else if pe.Repl != nil {
		if pe.Repl.All {
			p.WriteByte('/')
		}
		p.WriteByte('/')
		if pe.Repl.Orig != nil {
			p.word(pe.Repl.Orig)
		}
		p.WriteByte('/')
		if pe.Repl.With != nil {
			p.word(pe.Repl.With)
		}
	} else if pe.Names != 0 {
		p.WriteString(pe.Names.String())
	} else if pe.Exp != nil {
		p.WriteString(pe.Exp.Op.String())
		if pe.Exp.Word != nil {
			p.word(pe.Exp.Word)
		}
	}
	p.WriteByte('}')
}

func (p *Printer) loop(loop Loop) {
	switch x := loop.(type) {
	case *WordIter:
		p.WriteString(x.Name.Value)
		if len(x.Items) > 0 {
			p.spacedString(" in", Pos{})
			p.wordJoin(x.Items)
		}
	case *CStyleLoop:
		p.WriteString("((")
		if x.Init == nil {
			p.space()
		}
		p.arithmExpr(x.Init, false, false)
		p.WriteString("; ")
		p.arithmExpr(x.Cond, false, false)
		p.WriteString("; ")
		p.arithmExpr(x.Post, false, false)
		p.WriteString("))")
	}
}

func (p *Printer) arithmExpr(expr ArithmExpr, compact, spacePlusMinus bool) {
	if p.minify {
		compact = true
	}
	switch x := expr.(type) {
	case *Word:
		p.word(x)
	case *BinaryArithm:
		if compact {
			p.arithmExpr(x.X, compact, spacePlusMinus)
			p.WriteString(x.Op.String())
			p.arithmExpr(x.Y, compact, false)
		} else {
			p.arithmExpr(x.X, compact, spacePlusMinus)
			if x.Op != Comma {
				p.space()
			}
			p.WriteString(x.Op.String())
			p.space()
			p.arithmExpr(x.Y, compact, false)
		}
	case *UnaryArithm:
		if x.Post {
			p.arithmExpr(x.X, compact, spacePlusMinus)
			p.WriteString(x.Op.String())
		} else {
			if spacePlusMinus {
				switch x.Op {
				case Plus, Minus:
					p.space()
				}
			}
			p.WriteString(x.Op.String())
			p.arithmExpr(x.X, compact, false)
		}
	case *ParenArithm:
		p.WriteByte('(')
		p.arithmExpr(x.X, false, false)
		p.WriteByte(')')
	}
}

func (p *Printer) testExpr(expr TestExpr) {
	switch x := expr.(type) {
	case *Word:
		p.word(x)
	case *BinaryTest:
		p.testExpr(x.X)
		p.space()
		p.WriteString(x.Op.String())
		p.space()
		p.testExpr(x.Y)
	case *UnaryTest:
		p.WriteString(x.Op.String())
		p.space()
		p.testExpr(x.X)
	case *ParenTest:
		p.WriteByte('(')
		p.testExpr(x.X)
		p.WriteByte(')')
	}
}

func (p *Printer) word(w *Word) {
	p.wordParts(w.Parts)
	p.wantSpace = true
}

func (p *Printer) unquotedWord(w *Word) {
	for _, wp := range w.Parts {
		switch x := wp.(type) {
		case *SglQuoted:
			p.WriteString(x.Value)
		case *DblQuoted:
			p.wordParts(x.Parts)
		case *Lit:
			for i := 0; i < len(x.Value); i++ {
				if b := x.Value[i]; b == '\\' {
					if i++; i < len(x.Value) {
						p.WriteByte(x.Value[i])
					}
				} else {
					p.WriteByte(b)
				}
			}
		}
	}
}

func (p *Printer) wordJoin(ws []*Word) {
	anyNewline := false
	for _, w := range ws {
		if pos := w.Pos(); pos.Line() > p.line {
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.bslashNewl()
		} else {
			p.spacePad(w.Pos())
		}
		p.word(w)
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) casePatternJoin(pats []*Word) {
	anyNewline := false
	for i, w := range pats {
		if i > 0 {
			p.spacedToken("|", Pos{})
		}
		if pos := w.Pos(); pos.Line() > p.line {
			if !anyNewline {
				p.incLevel()
				anyNewline = true
			}
			p.bslashNewl()
		} else {
			p.spacePad(w.Pos())
		}
		p.word(w)
	}
	if anyNewline {
		p.decLevel()
	}
}

func (p *Printer) elemJoin(elems []*ArrayElem, last []Comment) {
	p.incLevel()
	for _, el := range elems {
		var left *Comment
		for _, c := range el.Comments {
			if c.Pos().After(el.Pos()) {
				left = &c
				break
			}
			p.comment(c)
		}
		if el.Pos().Line() > p.line {
			p.newline(el.Pos())
			p.indent()
		} else if p.wantSpace {
			p.space()
		}
		if p.wroteIndex(el.Index) {
			p.WriteByte('=')
		}
		p.word(el.Value)
		if left != nil {
			p.comment(*left)
		}
	}
	if len(last) > 0 {
		p.comments(last)
	}
	p.decLevel()
}

func (p *Printer) stmt(s *Stmt) {
	if s.Negated {
		p.spacedString("!", s.Pos())
	}
	var startRedirs int
	if s.Cmd != nil {
		startRedirs = p.command(s.Cmd, s.Redirs)
	}
	p.incLevel()
	for _, r := range s.Redirs[startRedirs:] {
		if r.OpPos.Line() > p.line {
			p.bslashNewl()
		}
		if p.minify && r.N == nil {
		} else if p.wantSpace {
			p.spacePad(r.Pos())
		}
		if r.N != nil {
			p.WriteString(r.N.Value)
		}
		p.WriteString(r.Op.String())
		p.wantSpace = true
		p.word(r.Word)
		if r.Op == Hdoc || r.Op == DashHdoc {
			p.pendingHdocs = append(p.pendingHdocs, r)
		}
	}
	p.wroteSemi = false
	switch {
	case s.Semicolon.IsValid() && s.Semicolon.Line() > p.line:
		p.bslashNewl()
		p.WriteByte(';')
		p.wroteSemi = true
	case s.Background:
		if !p.minify {
			p.space()
		}
		p.WriteString("&")
	case s.Coprocess:
		if !p.minify {
			p.space()
		}
		p.WriteString("|&")
	}
	p.decLevel()
}

func (p *Printer) command(cmd Command, redirs []*Redirect) (startRedirs int) {
	p.spacePad(cmd.Pos())
	switch x := cmd.(type) {
	case *CallExpr:
		p.assigns(x.Assigns, true)
		if len(x.Args) <= 1 {
			p.wordJoin(x.Args)
			return 0
		}
		p.wordJoin(x.Args[:1])
		for _, r := range redirs {
			if r.Pos().After(x.Args[1].Pos()) || r.Op == Hdoc || r.Op == DashHdoc {
				break
			}
			if p.minify && r.N == nil {
			} else if p.wantSpace {
				p.spacePad(r.Pos())
			}
			if r.N != nil {
				p.WriteString(r.N.Value)
			}
			p.WriteString(r.Op.String())
			p.wantSpace = true
			p.word(r.Word)
			startRedirs++
		}
		p.wordJoin(x.Args[1:])
	case *Block:
		p.WriteByte('{')
		p.wantSpace = true
		p.nestedStmts(x.StmtList, x.Rbrace)
		p.semiRsrv("}", x.Rbrace, true)
	case *IfClause:
		p.ifClause(x, false)
	case *Subshell:
		p.WriteByte('(')
		p.wantSpace = len(x.Stmts) > 0 && startsWithLparen(x.Stmts[0])
		p.spacePad(x.StmtList.pos())
		p.nestedStmts(x.StmtList, x.Rparen)
		p.wantSpace = false
		p.spacePad(x.Rparen)
		p.rightParen(x.Rparen)
	case *WhileClause:
		if x.Until {
			p.spacedString("until", x.Pos())
		} else {
			p.spacedString("while", x.Pos())
		}
		p.nestedStmts(x.Cond, Pos{})
		p.semiOrNewl("do", x.DoPos)
		p.nestedStmts(x.Do, Pos{})
		p.semiRsrv("done", x.DonePos, true)
	case *ForClause:
		if x.Select {
			p.WriteString("select ")
		} else {
			p.WriteString("for ")
		}
		p.loop(x.Loop)
		p.semiOrNewl("do", x.DoPos)
		p.nestedStmts(x.Do, Pos{})
		p.semiRsrv("done", x.DonePos, true)
	case *BinaryCmd:
		p.stmt(x.X)
		if p.minify || x.Y.Pos().Line() <= p.line {
			// leave p.nestedBinary untouched
			p.spacedToken(x.Op.String(), x.OpPos)
			p.line = x.Y.Pos().Line()
			p.stmt(x.Y)
			break
		}
		indent := !p.nestedBinary
		if indent {
			p.incLevel()
		}
		if p.binNextLine {
			if len(p.pendingHdocs) == 0 {
				p.bslashNewl()
			}
			p.spacedToken(x.Op.String(), x.OpPos)
			if len(x.Y.Comments) > 0 {
				p.wantSpace = false
				p.WriteByte('\n')
				p.indent()
				p.comments(x.Y.Comments)
				p.WriteByte('\n')
				p.indent()
			}
		} else {
			p.spacedToken(x.Op.String(), x.OpPos)
			p.line = x.OpPos.Line()
			p.comments(x.Y.Comments)
			p.newline(Pos{})
			p.indent()
		}
		p.line = x.Y.Pos().Line()
		_, p.nestedBinary = x.Y.Cmd.(*BinaryCmd)
		p.stmt(x.Y)
		if indent {
			p.decLevel()
		}
		p.nestedBinary = false
	case *FuncDecl:
		if x.RsrvWord {
			p.WriteString("function ")
		}
		p.WriteString(x.Name.Value)
		p.WriteString("()")
		if !p.minify {
			p.space()
		}
		p.line = x.Body.Pos().Line()
		p.stmt(x.Body)
	case *CaseClause:
		p.WriteString("case ")
		p.word(x.Word)
		p.WriteString(" in")
		if p.swtCaseIndent {
			p.incLevel()
		}
		for i, ci := range x.Items {
			var inlineCom *Comment
			for _, c := range ci.Comments {
				if c.Pos().After(ci.Patterns[0].Pos()) {
					inlineCom = &c
					break
				}
				p.comment(c)
			}
			if pos := ci.Patterns[0].Pos(); pos.Line() > p.line {
				p.newlines(pos)
			}
			p.casePatternJoin(ci.Patterns)
			p.WriteByte(')')
			p.wantSpace = !p.minify
			sep := len(ci.Stmts) > 1 || ci.StmtList.pos().Line() > p.line
			if ci.OpPos != x.Esac && !ci.StmtList.empty() &&
				ci.OpPos.Line() > ci.StmtList.end().Line() {
				sep = true
			}
			sl := ci.StmtList
			p.nestedStmts(sl, Pos{})
			if !p.minify || i != len(x.Items)-1 {
				p.level++
				if sep {
					p.newlines(ci.OpPos)
					p.wantNewline = true
				}
				p.spacedToken(ci.Op.String(), ci.OpPos)
				if inlineCom != nil {
					p.comment(*inlineCom)
				}
				p.level--
			}
		}
		p.comments(x.Last)
		if p.swtCaseIndent {
			p.decLevel()
		}
		p.semiRsrv("esac", x.Esac, len(x.Items) == 0)
	case *ArithmCmd:
		p.WriteString("((")
		if x.Unsigned {
			p.WriteString("# ")
		}
		p.arithmExpr(x.X, false, false)
		p.WriteString("))")
	case *TestClause:
		p.WriteString("[[ ")
		p.testExpr(x.X)
		p.spacedString("]]", x.Right)
	case *DeclClause:
		p.spacedString(x.Variant.Value, x.Pos())
		for _, w := range x.Opts {
			p.space()
			p.word(w)
		}
		p.assigns(x.Assigns, false)
	case *TimeClause:
		p.spacedString("time", x.Pos())
		if x.PosixFormat {
			p.spacedString("-p", x.Pos())
		}
		if x.Stmt != nil {
			p.stmt(x.Stmt)
		}
	case *CoprocClause:
		p.spacedString("coproc", x.Pos())
		if x.Name != nil {
			p.space()
			p.WriteString(x.Name.Value)
		}
		p.space()
		p.stmt(x.Stmt)
	case *LetClause:
		p.spacedString("let", x.Pos())
		for _, n := range x.Exprs {
			p.space()
			p.arithmExpr(n, true, false)
		}
	}
	return startRedirs
}

func (p *Printer) ifClause(ic *IfClause, elif bool) {
	if !elif {
		p.spacedString("if", ic.Pos())
	}
	p.nestedStmts(ic.Cond, Pos{})
	p.semiOrNewl("then", ic.ThenPos)
	p.nestedStmts(ic.Then, Pos{})
	if ic.FollowedByElif() {
		p.semiRsrv("elif", ic.ElsePos, true)
		p.ifClause(ic.Else.Stmts[0].Cmd.(*IfClause), true)
		return
	}
	if !ic.Else.empty() {
		p.semiRsrv("else", ic.ElsePos, true)
		p.nestedStmts(ic.Else, Pos{})
	} else if ic.ElsePos.IsValid() {
		p.line = ic.ElsePos.Line()
	}
	p.semiRsrv("fi", ic.FiPos, true)
}

func startsWithLparen(s *Stmt) bool {
	switch x := s.Cmd.(type) {
	case *Subshell:
		return true
	case *BinaryCmd:
		return startsWithLparen(x.X)
	}
	return false
}

func (p *Printer) hasInline(s *Stmt) bool {
	for _, c := range s.Comments {
		if c.Pos().Line() == s.End().Line() {
			return true
		}
	}
	return false
}

func (p *Printer) stmts(sl StmtList) {
	switch len(sl.Stmts) {
	case 0:
		p.comments(sl.Last)
		return
	case 1:
		s := sl.Stmts[0]
		pos := s.Pos()
		var inlineCom *Comment
		for _, c := range s.Comments {
			if c.Pos().After(s.Pos()) {
				inlineCom = &c
				break
			}
			p.comment(c)
		}
		if pos.Line() <= p.line || (p.minify && !p.wantSpace) {
			p.line = pos.Line()
			p.stmt(s)
		} else {
			if p.line > 0 {
				p.newlines(pos)
			}
			p.line = pos.Line()
			p.stmt(s)
			p.wantNewline = true
		}
		if inlineCom != nil {
			p.comment(*inlineCom)
		}
		p.comments(sl.Last)
		return
	}
	inlineIndent := 0
	lastIndentedLine := uint(0)
	for i, s := range sl.Stmts {
		pos := s.Pos()
		var inlineCom *Comment
		for _, c := range s.Comments {
			if c.Pos().After(s.Pos()) {
				inlineCom = &c
				break
			}
			p.comment(c)
		}
		if p.minify && i == 0 && !p.wantSpace {
		} else if p.line > 0 {
			p.newlines(pos)
		}
		p.line = pos.Line()
		if !p.hasInline(s) {
			inlineIndent = 0
			p.commentPadding = 0
			p.stmt(s)
			continue
		}
		p.stmt(s)
		if s.Pos().Line() > lastIndentedLine+1 {
			inlineIndent = 0
		}
		if inlineIndent == 0 {
			for _, s2 := range sl.Stmts[i:] {
				if !p.hasInline(s2) {
					break
				}
				if l := p.stmtCols(s2); l > inlineIndent {
					inlineIndent = l
				}
			}
		}
		if inlineIndent > 0 {
			if l := p.stmtCols(s); l > 0 {
				p.commentPadding = uint(inlineIndent - l)
			}
			lastIndentedLine = p.line
		}
		if inlineCom != nil {
			p.comment(*inlineCom)
		}
	}
	p.wantNewline = true
	p.comments(sl.Last)
}

type byteCounter int

func (c *byteCounter) WriteByte(b byte) error {
	switch {
	case *c < 0:
	case b == '\n':
		*c = -1
	default:
		*c++
	}
	return nil
}
func (c *byteCounter) WriteString(s string) (int, error) {
	switch {
	case *c < 0:
	case strings.Contains(s, "\n"):
		*c = -1
	default:
		*c += byteCounter(len(s))
	}
	return 0, nil
}
func (c *byteCounter) Reset(io.Writer) { *c = 0 }
func (c *byteCounter) Flush() error    { return nil }

// stmtCols reports the length that s will take when formatted in a
// single line. If it will span multiple lines, stmtCols will return -1.
func (p *Printer) stmtCols(s *Stmt) int {
	if p.lenPrinter == nil {
		return -1 // stmtCols call within stmtCols, bail
	}
	*p.lenPrinter = Printer{
		bufWriter: &p.lenCounter,
	}
	p.lenPrinter.bufWriter.Reset(nil)
	p.lenPrinter.line = s.Pos().Line()
	p.lenPrinter.stmt(s)
	return int(p.lenCounter)
}

func (p *Printer) nestedStmts(sl StmtList, closing Pos) {
	p.incLevel()
	if len(sl.Stmts) == 1 && closing.Line() > p.line && sl.Stmts[0].End().Line() <= p.line {
		p.newline(Pos{})
		p.indent()
	}
	p.stmts(sl)
	p.decLevel()
}

func (p *Printer) assigns(assigns []*Assign, alwaysEqual bool) {
	p.incLevel()
	for _, a := range assigns {
		if a.Pos().Line() > p.line {
			p.bslashNewl()
		} else {
			p.spacePad(a.Pos())
		}
		if a.Name != nil {
			p.WriteString(a.Name.Value)
			p.wroteIndex(a.Index)
			if a.Append {
				p.WriteByte('+')
			}
			if alwaysEqual || a.Value != nil || a.Array != nil {
				p.WriteByte('=')
			}
		}
		if a.Value != nil {
			p.word(a.Value)
		} else if a.Array != nil {
			p.wantSpace = false
			p.WriteByte('(')
			p.elemJoin(a.Array.Elems, a.Array.Last)
			p.rightParen(a.Array.Rparen)
		}
		p.wantSpace = true
	}
	p.decLevel()
}
