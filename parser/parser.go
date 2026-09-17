package parser

import (
	"fmt"
	"lin/commons"
	"lin/lexer"
	"math"
	"os"
	"strings"
)

type ExprKind int

const (
	KIND_NONE ExprKind = iota
	KIND_PROGRAM

	KIND_RETURN
	KIND_IF
	KIND_WHILE
	KIND_SWITCH
	KIND_CASE
	KIND_DEFAULT
	KIND_FOR
	KIND_BREAK
	KIND_CONTINUE
	KIND_BODY
	KIND_FREEARENA
	KIND_DUMP
	KIND_LONG
	KIND_INT
	KIND_STRUCT
	KIND_DOUBLE
	KIND_FLOAT
	KIND_STRING
	KIND_BOOL
	KIND_ADD
	KIND_MUL
	KIND_SUB
	KIND_DIV
	KIND_POW
	KIND_MOD
	KIND_CAST
	KIND_CONSTSET
	KIND_VARINIT
	KIND_VARGET
	KIND_VARSLICEGET
	KIND_VARASSIGN
	KIND_VARSLICEASSIGN
	KIND_VARFIELDGET
	KIND_VARFIELDASSIGN
	KIND_STRCAT
	KIND_NEG
	KIND_POSTINC
	KIND_POSTDEC
	KIND_PREINC
	KIND_PREDEC
	KIND_RAW_QBE
	KIND_TYPEDECL

	//Boolean Algebra
	KIND_OR
	KIND_AND
	KIND_EQ
	KIND_NEQ
	KIND_GT
	KIND_GE
	KIND_LT
	KIND_LE
	KIND_NOT
)

func (t ExprKind) String() string {
	switch t {
	case KIND_PROGRAM:
		return "program"
	case KIND_RETURN:
		return "return"
	case KIND_DUMP:
		return "dump"
	case KIND_LONG:
		return "int"
	case KIND_STRING:
		return "string"
	case KIND_BOOL:
		return "bool"
	case KIND_ADD:
		return "add"
	case KIND_MUL:
		return "mul"
	case KIND_SUB:
		return "sub"
	case KIND_DIV:
		return "div"
	case KIND_CAST:
		return "cast"
	case KIND_POW:
		return "pow"
	case KIND_MOD:
		return "mod"
	case KIND_CONSTSET:
		return "const set"
	case KIND_VARINIT:
		return "init var"
	case KIND_VARGET:
		return "get var"
	case KIND_VARSLICEGET:
		return "get slice var"
	case KIND_VARASSIGN:
		return "assign var"
	case KIND_VARSLICEASSIGN:
		return "assign slice var"
	case KIND_VARFIELDGET:
		return "field get"
	case KIND_VARFIELDASSIGN:
		return "field assign"
	case KIND_STRCAT:
		return "string concatenation"
	case KIND_FREEARENA:
		return "free arena"
	case KIND_POSTINC:
		return "postinc"
	case KIND_POSTDEC:
		return "postdec"
	case KIND_PREINC:
		return "preinc"
	case KIND_PREDEC:
		return "predec"
	case KIND_IF:
		return "if"
	case KIND_WHILE:
		return "while"
	case KIND_FOR:
		return "for"
	case KIND_SWITCH:
		return "switch"
	case KIND_CASE:
		return "case"
	case KIND_DEFAULT:
		return "default"
	case KIND_BODY:
		return "body"
	case KIND_BREAK:
		return "break"
	case KIND_CONTINUE:
		return "break"
	case KIND_RAW_QBE:
		return "raw qbe"
	case KIND_OR:
		return "or"
	case KIND_AND:
		return "and"
	case KIND_EQ:
		return "equal (==)"
	case KIND_NEQ:
		return "not equal (!=)"
	case KIND_GT:
		return ">"
	case KIND_GE:
		return ">="
	case KIND_LT:
		return "<"
	case KIND_LE:
		return "<="
	case KIND_NOT:
		return "not (!)"
	case KIND_TYPEDECL:
		return "type declaration"
	case KIND_STRUCT:
		return "struct"
	default:
		return "unknown"
	}
}

type SymbolKind int

const (
	SYMBOL_VAR SymbolKind = iota
	SYMBOL_CONST
	SYMBOL_TYPE
)

type ContextKind int

const (
	CONTEXT_VARSET ContextKind = iota
	CONTEXT_BINOPASSIGN
	CONTEXT_CONSTSET
	CONTEXT_CAST
	CONTEXT_IF
	CONTEXT_CASE
	CONTEXT_DEFAULT
	CONTEXT_SWITCH
	CONTEXT_WHILE
	CONTEXT_FOR
	CONTEXT_CMP
	CONTEXT_BODY
	CONTEXT_ROOT
	CONTEXT_DUMP
	CONTEXT_RETURN
	CONTEXT_SLICE
)

type SwitchExpr struct {
	ID      int
	Cases   []int
	Default int
}

type Expr struct {
	Kind ExprKind
	Type commons.ExprType

	ValueLong    int64
	ValueDouble  float64
	ValueInt     int32
	ValueFloat   float32
	ValueString  string
	ValueKind    ExprKind
	ValueType    commons.ExprType
	ValueSymbol  *Symbol
	ValueContext *Context

	// Switch
	ValueSwitch *SwitchExpr

	// Raw QBE metadata. RawCaptures maps ${name} placeholders to Lin symbols.
	RawResult   string
	RawCaptures map[string]*Symbol

	// Bodies
	ID int
	// IF BODY
	HasElse bool

	Children []*Expr

	Line   int
	Column int

	Parser *Parser
}

type Field struct {
	Name       string
	Type       commons.ExprType
	TypeSymbol *Symbol // for structs when Type == TYPE_STRUCT
	// for array fields (Type == TYPE_ARRAY): describe element
	Element    commons.ExprType
	ElementSym *Symbol
	Size       int
	Init       *Expr
}

type Symbol struct {
	Name     string
	Internal string
	Type     commons.ExprType
	Kind     SymbolKind

	Value *Expr

	//Struct TYPE
	Fields   []*Field
	FieldMap map[string]*Field

	//Array TYPE
	Element    commons.ExprType
	ElementSym *Symbol
	Size       int
	// For variables of non-primitive types (e.g., struct),
	// keep a reference to the type Symbol declaration.
	TypeSymbol *Symbol
}

type Context struct {
	Parser   *Parser
	Parent   *Context
	Children []*Context
	Type     commons.ExprType
	Kind     ContextKind
	BodyKind ContextKind
	Symbols  map[string]*Symbol
	// KIND == CONTEXT_VARSET
	VarName  string
	VarIndex int
	// KIND == CONTEXT_BODY
	ID int
}

func (c *Context) Lookup(name string) (*Symbol, bool) {
	for ctx := c; ctx != nil; ctx = ctx.Parent {
		if sym, ok := ctx.Symbols[name]; ok {
			return sym, true
		}
	}

	return nil, false
}

func (c *Context) Declare(name string, sym *Symbol) (*Symbol, bool) {
	if sym, exists := c.Symbols[name]; exists {
		return sym, false
	}
	sym.Name = name
	sym.Internal = c.Parser.newSymbolName(name)
	c.Symbols[name] = sym
	return sym, true
}

type Parser struct {
	lexer          *lexer.Lexer
	Strings        map[int]string
	StringsLookup  map[string]int
	HasDump        bool
	HasReturn      bool
	HasFreeArena   bool
	StmtContexts   []*Context
	CurrentContext *Context
	Root           *Expr

	NextSymbolID int
	NextIfID     int
	NextWhileID  int
	NextForID    int
	NextSwitchID int
	// >0 while parsing a sub-expression (rvalue). Assignments are forbidden
	// when rvalueDepth > 0.
	rvalueDepth int
}

func New(l *lexer.Lexer) *Parser {
	rootContext := &Context{
		Kind:    CONTEXT_ROOT,
		Type:    commons.TYPE_UNDEFINED,
		Symbols: make(map[string]*Symbol),
	}

	p := &Parser{
		lexer:          l,
		HasDump:        false,
		HasReturn:      false,
		HasFreeArena:   false,
		Strings:        make(map[int]string),
		StringsLookup:  make(map[string]int),
		StmtContexts:   []*Context{rootContext},
		CurrentContext: rootContext,
	}

	rootContext.Parser = p

	return p
}

func (p *Parser) newExpr(
	kind ExprKind,
	typ commons.ExprType,
	rootExpr any,
	children ...*Expr,
) *Expr {
	var line, column int

	switch root := rootExpr.(type) {
	case *Expr:
		line = root.Line
		column = root.Column

	case lexer.Token:
		line = root.Line
		column = root.Column

	default:
		commons.CrashOut(fmt.Sprintf("invalid rootExpr type: %q", rootExpr), p.lexer.File_path, line, column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	expr := &Expr{
		Kind:     kind,
		Type:     typ,
		Children: children,
		Parser:   p,
		Line:     line,
		Column:   column,
	}

	return expr
}

func (p *Parser) parseRValue() *Expr {
	p.rvalueDepth++
	e := p.parseExpression()
	p.rvalueDepth--
	return e
}

func (p *Parser) assignAllowed() bool { return p.rvalueDepth == 0 }

func (p *Parser) newContext(
	parent *Context,
	context *Context,
) *Context {
	context.Parser = p
	context.Parent = parent
	context.Symbols = make(map[string]*Symbol)
	parent.Children = append(parent.Children, context)
	p.CurrentContext = context
	return context
}

func (p *Parser) endContext() *Context {
	if p.CurrentContext == nil {
		return nil
	}

	p.CurrentContext = p.CurrentContext.Parent
	return p.CurrentContext
}

func (p *Parser) newSymbolName(name string) string {
	gen := commons.NewHashName(name, p.NextSymbolID)
	p.NextSymbolID++

	return gen
}

func (p *Parser) newIfID() int {
	id := p.NextIfID
	p.NextIfID++
	return id
}

func (p *Parser) newWhileID() int {
	id := p.NextWhileID
	p.NextWhileID++
	return id
}

func (p *Parser) newForID() int {
	id := p.NextForID
	p.NextForID++
	return id
}

func (p *Parser) newSwitchID() int {
	id := p.NextSwitchID
	p.NextSwitchID++
	return id
}

func (e *Expr) checkKind(kind ExprKind) {
	if e.Kind != kind {
		commons.CrashOut(fmt.Sprintf(
			"expected %q got %q",
			kind.String(),
			e.Kind.String(),
		), e.Parser.lexer.File_path, e.Line, e.Column, commons.CRASH_ERROR)
		os.Exit(1)
	}
}

func (e *Expr) checkType(types ...commons.ExprType) {
	for _, typ := range types {
		if e.Type == typ {
			return
		}
	}

	expected := ""

	for i, typ := range types {
		if i > 0 {
			expected += ", "
		}
		expected += "\"" + typ.String() + "\""
	}

	commons.CrashOut(fmt.Sprintf(
		"expected %s, got %q",
		expected,
		e.Type.String(),
	), e.Parser.lexer.File_path, e.Line, e.Column, commons.CRASH_ERROR)

	os.Exit(1)
}

func (e *Expr) IsConstant() bool {
	switch e.Kind {
	case KIND_INT,
		KIND_LONG,
		KIND_FLOAT,
		KIND_DOUBLE,
		KIND_STRING,
		KIND_BOOL:

		return true

	case KIND_ADD,
		KIND_SUB,
		KIND_MUL,
		KIND_DIV,
		KIND_MOD,
		KIND_POW,
		KIND_CAST,
		KIND_STRCAT,
		KIND_GT, KIND_GE, KIND_LT, KIND_LE,
		KIND_AND, KIND_OR, KIND_NOT,
		KIND_EQ, KIND_NEQ:

		for _, child := range e.Children {
			if !child.IsConstant() {
				return false
			}
		}

		return true

	default:
		return false
	}
}

func (p *Parser) expectKind(kind lexer.TokenKind) lexer.Token {
	tok := p.lexer.NextToken()

	if tok.Kind != kind {
		commons.CrashOut(fmt.Sprintf(
			"expected %q got %q",
			kind.String(),
			tok.Kind.String(),
		), p.lexer.File_path, tok.Line, tok.Column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	return tok
}

func (p *Parser) optionalExpectKind(kind lexer.TokenKind) (tok lexer.Token, found bool) {
	found = false
	if p.lexer.PeekToken().Kind == kind {
		tok = p.expectKind(kind)
		found = true
	}
	return
}

func (p *Parser) expectType(expr *Expr, t commons.ExprType) {
	if expr.Type != t {
		commons.CrashOut(
			fmt.Sprintf("expected %q got %q", t, expr.Type),
			p.lexer.File_path,
			expr.Line,
			expr.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}
}

func (p *Parser) getValueType(value any, Type commons.ExprType, root any) *Expr {
	expr := p.newExpr(KIND_INT, Type, root)
	switch Type {
	case commons.TYPE_I32:
		expr.Kind = KIND_INT
		expr.ValueInt = int32(value.(int))
	case commons.TYPE_I64:
		expr.Kind = KIND_LONG
		expr.ValueLong = int64(value.(int))
	case commons.TYPE_F32:
		expr.Kind = KIND_FLOAT
		expr.ValueFloat = float32(value.(int))
	case commons.TYPE_F64:
		expr.Kind = KIND_DOUBLE
		expr.ValueDouble = float64(value.(int))
	default:
		panic(fmt.Sprintf("can't generate value for type %q", Type.String()))
	}
	return expr
}

func (p *Parser) parseString() *Expr {
	tok := p.expectKind(lexer.TOKEN_STRING)

	var id int
	if str, exists := p.StringsLookup[tok.Val_string]; exists {
		id = str
	} else {
		id = len(p.Strings)

		p.Strings[id] = tok.Val_string
		p.StringsLookup[tok.Val_string] = id
	}

	expr := p.newExpr(KIND_STRING, commons.TYPE_STRING, tok)
	expr.ValueString = tok.Val_string
	expr.ValueLong = int64(id)
	return expr
}

func (p *Parser) parseLong() *Expr {
	tok := p.expectKind(lexer.TOKEN_I64)

	expr := p.newExpr(KIND_LONG, commons.TYPE_I64, tok)
	expr.ValueLong = tok.Val_long
	return expr
}

func (p *Parser) parseInt() *Expr {
	tok := p.expectKind(lexer.TOKEN_I32)
	expr := p.newExpr(KIND_INT, commons.TYPE_I32, tok)
	expr.ValueInt = tok.Val_int
	return expr
}

func (p *Parser) parseDouble() *Expr {
	tok := p.expectKind(lexer.TOKEN_F64)
	expr := p.newExpr(KIND_DOUBLE, commons.TYPE_F64, tok)
	expr.ValueDouble = tok.Val_double
	return expr
}

func (p *Parser) ParseFloat() *Expr {
	tok := p.expectKind(lexer.TOKEN_F32)
	expr := p.newExpr(KIND_FLOAT, commons.TYPE_F32, tok)
	expr.ValueFloat = tok.Val_float
	return expr
}

func (p *Parser) parseAssign(id lexer.Token) *Expr {
	symbol, exists := p.CurrentContext.Lookup(id.Val_string)

	if !exists {
		commons.CrashOut(fmt.Sprintf("variable %q does not exists", id.Val_string), p.lexer.File_path, id.Line, id.Column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_EQUAL)

	var expr *Expr

	switch symbol.Type {
	case commons.TYPE_ARRAY:
		// parse the array literal/value as an rvalue and then build a var assign
		val := p.parseSetArrayValues(id, symbol.Size, false, symbol.Element, symbol.ElementSym)
		left := p.newExpr(KIND_VARGET, symbol.Type, id)
		left.ValueString = symbol.Name
		left.ValueSymbol = symbol
		expr = p.newExpr(KIND_VARASSIGN, symbol.Type, id, left, val)
	case commons.TYPE_STRUCT:
		var structVal *Expr
		if p.lexer.PeekToken().Kind == lexer.TOKEN_ID {
			structId := p.expectKind(lexer.TOKEN_ID)
			structVal = p.parseStructValue(structId)
		} else if p.lexer.PeekToken().Kind == lexer.TOKEN_LCPAREN {
			// Infer struct type from the variable's declared type symbol
			if symbol.TypeSymbol == nil {
				commons.CrashOut(
					"missing struct type information for assignment",
					p.lexer.File_path,
					p.lexer.PeekToken().Line,
					p.lexer.PeekToken().Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			structVal = p.parseStructLiteralBody(symbol.TypeSymbol, p.lexer.PeekToken())
		}
		if structVal == nil {
			commons.CrashOut(
				"expected struct literal or struct type identifier",
				p.lexer.File_path,
				id.Line,
				id.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		// Wrap struct literal into a var assignment node
		p.newContext(p.CurrentContext, &Context{
			Kind:    CONTEXT_VARSET,
			Type:    symbol.Type,
			VarName: symbol.Name,
		})
		p.endContext()
		left := p.newExpr(KIND_VARGET, symbol.Type, id)
		left.ValueString = symbol.Name
		left.ValueSymbol = symbol
		expr = p.newExpr(KIND_VARASSIGN, symbol.Type, id, left, structVal)
		expr.ValueString = symbol.Name
	default:
		p.newContext(p.CurrentContext, &Context{
			Kind:    CONTEXT_VARSET,
			Type:    symbol.Type,
			VarName: symbol.Name,
		})

		value := p.parseRValue()

		value = p.implicitCast(value, symbol.Type)
		value.checkType(symbol.Type)

		p.endContext()
		left := p.newExpr(KIND_VARGET, symbol.Type, id)
		left.ValueString = symbol.Name
		left.ValueSymbol = symbol
		expr = p.newExpr(KIND_VARASSIGN, symbol.Type, id, left, value)
		expr.ValueString = symbol.Name
	}
	expr.ValueSymbol = symbol
	return expr
}

func (p *Parser) parseAssignmentBINOP(id lexer.Token) *Expr {
	symbol, exists := p.CurrentContext.Lookup(id.Val_string)

	if !exists || symbol.Kind != SYMBOL_VAR {
		commons.CrashOut(
			fmt.Sprintf("symbol %q is not a variable", id.Val_string),
			p.lexer.File_path,
			id.Line,
			id.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	tk := p.lexer.NextToken()

	var binopKind ExprKind

	switch tk.Kind {
	case lexer.TOKEN_PLUS_EQUAL:
		binopKind = KIND_ADD
	case lexer.TOKEN_MINUS_EQUAL:
		binopKind = KIND_SUB
	case lexer.TOKEN_MUL_EQUAL:
		binopKind = KIND_MUL
	case lexer.TOKEN_DIV_EQUAL:
		binopKind = KIND_DIV
	default:
		commons.CrashOut(
			fmt.Sprintf("unsupported assignment operator %q", tk.Kind.String()),
			p.lexer.File_path,
			id.Line,
			id.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	p.newContext(p.CurrentContext, &Context{
		Kind:    CONTEXT_BINOPASSIGN,
		Type:    symbol.Type,
		VarName: symbol.Name,
	})

	value := p.implicitCast(p.parseRValue(), symbol.Type)
	value.checkType(symbol.Type)

	p.endContext()

	left := p.newExpr(
		KIND_VARGET,
		symbol.Type,
		id,
	)
	left.ValueString = symbol.Name
	left.ValueSymbol = symbol

	var right *Expr
	switch symbol.Type {
	case commons.TYPE_STRING:
		if binopKind == KIND_ADD {
			right = p.newExpr(KIND_STRCAT, commons.TYPE_STRING, id, left, value)

		} else {
			commons.CrashOut(
				fmt.Sprintf(
					"operator %q cannot be applied to type %q",
					binopKind.String(),
					symbol.Type.String(),
				),
				p.lexer.File_path,
				id.Line,
				id.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
	default:
		right = p.newExpr(
			binopKind,
			symbol.Type,
			id,
			left,
			value,
		)
	}
	right.ValueSymbol = symbol

	expr_assign := p.newExpr(
		KIND_VARASSIGN,
		symbol.Type,
		id,
		left,
		right,
	)
	expr_assign.ValueString = symbol.Name
	expr_assign.ValueSymbol = symbol

	return expr_assign
}

func (p *Parser) checkIncDecType(
	Type commons.ExprType,
	line int,
	column int,
) {
	switch Type {
	case commons.TYPE_I8,
		commons.TYPE_U8,
		commons.TYPE_I16,
		commons.TYPE_U16,
		commons.TYPE_I32,
		commons.TYPE_I64,
		commons.TYPE_F32,
		commons.TYPE_F64:
		return

	default:
		commons.CrashOut(
			fmt.Sprintf(
				"can't increment or decrement value of type %q",
				Type.String(),
			),
			p.lexer.File_path,
			line,
			column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}
}

func (p *Parser) buildIncDecAssign(target *Expr, incdecKind ExprKind) *Expr {
	p.checkIncDecType(target.Type, target.Line, target.Column)

	one := p.getValueType(1, target.Type, target)
	binop := p.newExpr(incdecKind, target.Type, target, target, one)
	binop.ValueSymbol = target.ValueSymbol

	return p.buildAssign(target, binop)
}

func (p *Parser) assignKindFor(target *Expr) ExprKind {
	switch target.Kind {
	case KIND_VARGET:
		return KIND_VARASSIGN
	case KIND_VARSLICEGET:
		return KIND_VARSLICEASSIGN
	case KIND_VARFIELDGET:
		return KIND_VARFIELDASSIGN
	default:
		return KIND_NONE
	}
}

func (p *Parser) buildAssign(target, value *Expr) *Expr {
	kind := p.assignKindFor(target)
	if kind == KIND_NONE {
		commons.CrashOut(
			fmt.Sprintf("cannot assign to %q", target.Kind.String()),
			p.lexer.File_path,
			target.Line,
			target.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	e := p.newExpr(kind, target.Type, target, target, value)
	e.ValueString = target.ValueString
	e.ValueSymbol = target.ValueSymbol
	e.ValueInt = target.ValueInt
	return e
}

func (p *Parser) parsePostINCDEC(id any) *Expr {
	var line, column int
	var valueString string
	var target *Expr
	var symbol *Symbol
	var targetType commons.ExprType
	isID := false

	switch root := id.(type) {
	case *Expr:
		line = root.Line
		column = root.Column
		valueString = root.ValueString

		switch root.Kind {
		case KIND_VARGET, KIND_VARSLICEGET:
			target = root
			targetType = root.Type
			symbol = root.ValueSymbol
			isID = true
		}

	case lexer.Token:
		line = root.Line
		column = root.Column
		valueString = root.Val_string

		if root.Kind == lexer.TOKEN_ID {
			isID = true
		}

	default:
		commons.CrashOut(fmt.Sprintf("invalid id type: %q", id), p.lexer.File_path, line, column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	if !isID {
		commons.CrashOut("invalid id kind", p.lexer.File_path, line, column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	if symbol == nil {
		var exists bool
		symbol, exists = p.CurrentContext.Lookup(valueString)
		if !exists || symbol.Kind != SYMBOL_VAR {
			commons.CrashOut(
				fmt.Sprintf("symbol %q is not a variable", valueString),
				p.lexer.File_path,
				line,
				column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		targetType = symbol.Type
		target = p.newExpr(KIND_VARGET, symbol.Type, id)
		target.ValueString = symbol.Name
		target.ValueSymbol = symbol
	}

	var incdecKind ExprKind
	var postKind ExprKind
	switch p.lexer.NextToken().Kind {
	case lexer.TOKEN_INC:
		incdecKind = KIND_ADD
		postKind = KIND_POSTINC
	case lexer.TOKEN_DEC:
		incdecKind = KIND_SUB
		postKind = KIND_POSTDEC
	default:
		commons.CrashOut(
			"expected ++ or --",
			p.lexer.File_path,
			line,
			column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	exprAssign := p.buildIncDecAssign(target, incdecKind)
	exprPost := p.newExpr(postKind, targetType, target, target, exprAssign)
	exprPost.ValueSymbol = symbol
	return exprPost
}

func (p *Parser) parsePreINCDEC() *Expr {
	var incdecKind ExprKind
	var preKind ExprKind

	switch tk := p.lexer.NextToken(); tk.Kind {
	case lexer.TOKEN_INC:
		incdecKind = KIND_ADD
		preKind = KIND_PREINC

	case lexer.TOKEN_DEC:
		incdecKind = KIND_SUB
		preKind = KIND_PREDEC

	default:
		commons.CrashOut(
			"expected ++ or --",
			p.lexer.File_path,
			tk.Line,
			tk.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	target := p.parsePrimary()
	switch target.Kind {
	case KIND_VARGET, KIND_VARSLICEGET:
		// valid lvalue expressions
	default:
		commons.CrashOut(
			fmt.Sprintf("invalid lvalue kind %q for ++/--", target.Kind.String()),
			p.lexer.File_path,
			target.Line,
			target.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	exprAssign := p.buildIncDecAssign(target, incdecKind)
	exprPre := p.newExpr(preKind, target.Type, target, exprAssign)
	exprPre.ValueSymbol = target.ValueSymbol
	return exprPre
}

func (p *Parser) parseId() *Expr {
	id := p.expectKind(lexer.TOKEN_ID)

	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_EQUAL:
		return p.parseAssign(id)
	case lexer.TOKEN_DOUBLECOLON:
		return p.parseConstSet(id)
	case lexer.TOKEN_PLUS_EQUAL, lexer.TOKEN_MINUS_EQUAL, lexer.TOKEN_DIV_EQUAL, lexer.TOKEN_MUL_EQUAL:
		return p.parseAssignmentBINOP(id)
	case lexer.TOKEN_INC, lexer.TOKEN_DEC:
		return p.parsePostINCDEC(id)
	case lexer.TOKEN_LRPAREN:
		return p.parseCast(id)
	case lexer.TOKEN_LCPAREN:
		return p.parseStructValue(id)
	default:
		symbol, exists := p.CurrentContext.Lookup(id.Val_string)

		if !exists {
			commons.CrashOut(fmt.Sprintf("no symbol exists with name %q", id.Val_string), p.lexer.File_path, id.Line, id.Column, commons.CRASH_ERROR)
			os.Exit(1)
		}

		if symbol.Kind == SYMBOL_CONST {
			return symbol.Value
		}

		expr := p.newExpr(KIND_VARGET, symbol.Type, id)
		expr.ValueString = symbol.Name
		expr.ValueSymbol = symbol
		return expr
	}
}

func (p *Parser) parseBool() *Expr {
	tok := p.lexer.NextToken()

	value := int32(0)

	if tok.Kind == lexer.TOKEN_TRUE {
		value = 1
	} else if tok.Kind != lexer.TOKEN_FALSE {
		commons.CrashOut(fmt.Sprintf("expected \"true\" or \"false\" got %q", tok.Kind.String()), p.lexer.File_path, tok.Line, tok.Column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	return &Expr{
		Kind:     KIND_BOOL,
		Type:     commons.TYPE_BOOL,
		ValueInt: value,
		Line:     tok.Line,
		Column:   tok.Column,
		Parser:   p,
	}
}

func (p *Parser) parsePrimary() *Expr {
	switch p.lexer.PeekToken().Kind {

	case lexer.TOKEN_TYPE:
		Type := p.expectKind(lexer.TOKEN_TYPE)
		return p.parseCast(Type)

	case lexer.TOKEN_I64:
		return p.parseLong()

	case lexer.TOKEN_F64:
		return p.parseDouble()

	case lexer.TOKEN_I32:
		return p.parseInt()

	case lexer.TOKEN_F32:
		return p.ParseFloat()

	case lexer.TOKEN_STRING:
		return p.parseString()

	case lexer.TOKEN_LRPAREN:
		return p.parseParen()

	case lexer.TOKEN_LSPAREN:
		// array literal
		return p.parseArrayLiteral()

	case lexer.TOKEN_ID:
		return p.parseId()

	//case lexer.TOKEN_LCPAREN:
	//	return p.parseId()

	case lexer.TOKEN_INC, lexer.TOKEN_DEC:
		return p.parsePreINCDEC()

	case lexer.TOKEN_TRUE, lexer.TOKEN_FALSE:
		return p.parseBool()

	case lexer.TOKEN_QBE:
		return p.parseRawQBE()

	default:
		tok := p.lexer.NextToken()
		commons.CrashOut(
			fmt.Sprintf("primary: unexpected %q", tok.Kind.String()),
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	panic("UNREACHABLE")
}

func (p *Parser) parseRawQBE() *Expr {
	qbeTok := p.expectKind(lexer.TOKEN_QBE)
	resultType := commons.TYPE_UNDEFINED

	if p.lexer.PeekToken().Kind == lexer.TOKEN_TYPE {
		resultType = p.expectKind(lexer.TOKEN_TYPE).Val_type
	}

	p.expectKind(lexer.TOKEN_LSHIFT)
	raw := p.lexer.ReadRawQBE()
	if strings.Contains(raw.Val_string, "__lin_") {
		commons.CrashOut("raw QBE expression can't use namespace '__lin_' in any place", p.lexer.File_path, raw.Line, raw.Column, commons.CRASH_ERROR)
		os.Exit(1)
	}
	expr := p.newExpr(KIND_RAW_QBE, resultType, qbeTok)
	expr.ValueString = raw.Val_string
	expr.RawCaptures = make(map[string]*Symbol)

	if resultType != commons.TYPE_UNDEFINED {
		lines := strings.SplitAfter(raw.Val_string, "\n")
		found := false
		var body strings.Builder
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "yield ") {
				if found {
					commons.CrashOut("raw QBE expression has multiple yield statements", p.lexer.File_path, raw.Line, raw.Column, commons.CRASH_ERROR)
					os.Exit(1)
				}
				expr.RawResult = strings.TrimSpace(strings.TrimPrefix(trimmed, "yield "))
				found = true
				continue
			}
			body.WriteString(line)
		}
		if !found || expr.RawResult == "" {
			commons.CrashOut("raw QBE expression must contain one `yield <value>` line", p.lexer.File_path, raw.Line, raw.Column, commons.CRASH_ERROR)
			os.Exit(1)
		}
		expr.ValueString = body.String()
	}

	// Resolve captures while the parser still has the correct lexical context.
	for _, source := range []string{expr.ValueString, expr.RawResult} {
		for pos := 0; pos < len(source); {
			start := strings.Index(source[pos:], "${")
			if start < 0 {
				break
			}
			start += pos
			end := strings.IndexByte(source[start+2:], '}')
			if end < 0 {
				commons.CrashOut("unterminated raw QBE capture, expected ${name}", p.lexer.File_path, raw.Line, raw.Column, commons.CRASH_ERROR)
				os.Exit(1)
			}
			end += start + 2
			name := source[start+2 : end]
			symbol, exists := p.CurrentContext.Lookup(name)
			if !exists {
				commons.CrashOut(fmt.Sprintf("raw QBE capture refers to unknown symbol %q", name), p.lexer.File_path, raw.Line, raw.Column, commons.CRASH_ERROR)
				os.Exit(1)
			}
			expr.RawCaptures[name] = symbol
			pos = end + 1
		}
	}

	return expr
}

func (p *Parser) parseUnary() *Expr {
	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_MINUS:
		tok := p.expectKind(lexer.TOKEN_MINUS)

		value := p.parseUnary()

		return p.newExpr(
			KIND_NEG,
			value.Type,
			tok,
			value,
		)
	case lexer.TOKEN_NOT:
		tok := p.expectKind(lexer.TOKEN_NOT)

		value := p.parseUnary()
		value = p.implicitCast(value, commons.TYPE_BOOL)
		value.checkType(commons.TYPE_BOOL)

		return p.newExpr(
			KIND_NOT,
			commons.TYPE_BOOL,
			tok,
			value,
		)
	}

	return p.parsePow()
}

func (p *Parser) parseOr() *Expr {
	left := p.parseAnd()

	for p.lexer.PeekToken().Kind == lexer.TOKEN_OR {
		tok := p.expectKind(lexer.TOKEN_OR)
		right := p.parseAnd()

		left = p.implicitCast(left, commons.TYPE_BOOL)
		right = p.implicitCast(right, commons.TYPE_BOOL)

		left = p.newExpr(
			KIND_OR,
			commons.TYPE_BOOL,
			tok,
			left,
			right,
		)
	}

	return left
}

func (p *Parser) parseAnd() *Expr {
	left := p.parseComparison()

	for p.lexer.PeekToken().Kind == lexer.TOKEN_AND {
		tok := p.expectKind(lexer.TOKEN_AND)
		right := p.parseComparison()

		left = p.implicitCast(left, commons.TYPE_BOOL)
		right = p.implicitCast(right, commons.TYPE_BOOL)

		left = p.newExpr(
			KIND_AND,
			commons.TYPE_BOOL,
			tok,
			left,
			right,
		)
	}

	return left
}

func (p *Parser) getEqualityComparisonType(
	left *Expr,
	right *Expr,
) commons.ExprType {

	if left.Type == commons.TYPE_BOOL ||
		right.Type == commons.TYPE_BOOL {

		if left.Type != commons.TYPE_BOOL ||
			right.Type != commons.TYPE_BOOL {
			commons.CrashOut(
				"boolean can only be compared with boolean",
				p.lexer.File_path,
				left.Line,
				left.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		return commons.TYPE_BOOL
	}

	if left.Type == commons.TYPE_STRING ||
		right.Type == commons.TYPE_STRING {

		if left.Type != commons.TYPE_STRING ||
			right.Type != commons.TYPE_STRING {
			commons.CrashOut(
				"string can only be compared with string",
				p.lexer.File_path,
				left.Line,
				left.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		return commons.TYPE_STRING
	}

	return p.getFinalBinopType(left, right)
}

func (p *Parser) getNumericComparisonType(left, right *Expr) commons.ExprType {
	switch left.Type {
	case commons.TYPE_I8,
		commons.TYPE_U8,
		commons.TYPE_I16,
		commons.TYPE_U16,
		commons.TYPE_I32,
		commons.TYPE_I64,
		commons.TYPE_F32,
		commons.TYPE_F64:
		// valid
	default:
		commons.CrashOut(
			fmt.Sprintf(
				"operator cannot be applied to type %q",
				left.Type.String(),
			),
			p.lexer.File_path,
			left.Line,
			left.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	switch right.Type {
	case commons.TYPE_I8,
		commons.TYPE_U8,
		commons.TYPE_I16,
		commons.TYPE_U16,
		commons.TYPE_I32,
		commons.TYPE_I64,
		commons.TYPE_F32,
		commons.TYPE_F64:
		// valid
	default:
		commons.CrashOut(
			fmt.Sprintf(
				"operator cannot be applied to type %q",
				right.Type.String(),
			),
			p.lexer.File_path,
			right.Line,
			right.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	// For string i arleady check before
	return p.getFinalNumericType(left, right)
}

func (p *Parser) parseComparison() *Expr {
	left := p.parseAddSub()

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_CMP,
		Type: commons.TYPE_UNDEFINED,
	})

	for {
		switch PkTok := p.lexer.PeekToken().Kind; PkTok {
		case lexer.TOKEN_EQ:
			tok := p.expectKind(lexer.TOKEN_EQ)
			right := p.parseAddSub()

			finalType := p.getEqualityComparisonType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			left.checkType(finalType)
			right.checkType(finalType)

			left = p.newExpr(
				KIND_EQ,
				commons.TYPE_BOOL,
				tok,
				left,
				right,
			)
		case lexer.TOKEN_NEQ:
			tok := p.expectKind(lexer.TOKEN_NEQ)
			right := p.parseAddSub()

			finalType := p.getEqualityComparisonType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			left.checkType(finalType)
			right.checkType(finalType)

			left = p.newExpr(
				KIND_NEQ,
				commons.TYPE_BOOL,
				tok,
				left,
				right,
			)

		case lexer.TOKEN_GT, lexer.TOKEN_GE, lexer.TOKEN_LT, lexer.TOKEN_LE:
			tok := p.expectKind(PkTok)
			right := p.parseAddSub()

			finalType := p.getNumericComparisonType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			left.checkType(finalType)
			right.checkType(finalType)

			var kind ExprKind
			switch PkTok {
			case lexer.TOKEN_GT:
				kind = KIND_GT
			case lexer.TOKEN_GE:
				kind = KIND_GE
			case lexer.TOKEN_LT:
				kind = KIND_LT
			case lexer.TOKEN_LE:
				kind = KIND_LE
			}

			left = p.newExpr(
				kind,
				commons.TYPE_BOOL,
				tok,
				left,
				right,
			)

		default:
			p.endContext()
			return left
		}
	}

}

func (p *Parser) parseExpression() *Expr {
	return p.parseOr() //GENERAL BINOP or String
}

func (p *Parser) getFinalBinopType(left *Expr, right *Expr) commons.ExprType {
	if left.Type == commons.TYPE_STRING || right.Type == commons.TYPE_STRING {
		return commons.TYPE_STRING
	}

	if p.CurrentContext.Type != commons.TYPE_UNDEFINED {
		return p.CurrentContext.Type
	}

	return p.getFinalNumericType(left, right)
}

func (p *Parser) getFinalNumericType(left *Expr, right *Expr) commons.ExprType {
	rank := map[commons.ExprType]int{
		commons.TYPE_I32: 1,
		commons.TYPE_I64: 2,
		commons.TYPE_F32: 3,
		commons.TYPE_F64: 4,
	}

	if rank[left.Type] >= rank[right.Type] {
		return left.Type
	}

	return right.Type
}

func (p *Parser) implicitCast(expr *Expr, finalType commons.ExprType) *Expr {
	expr.checkType(
		commons.TYPE_I32, commons.TYPE_I64, // Full Number
		commons.TYPE_F64, commons.TYPE_F32, // Decimal
		commons.TYPE_I16, commons.TYPE_I8, // Smaller
		commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
		commons.TYPE_STRING, commons.TYPE_BOOL, commons.TYPE_STRUCT, commons.TYPE_ARRAY, // "Aliases"
	)

	if expr.Type == finalType {
		return expr
	}

	switch finalType {
	case commons.TYPE_I32:
		if expr.Type == commons.TYPE_I16 ||
			expr.Type == commons.TYPE_I8 ||
			expr.Type == commons.TYPE_U16 ||
			expr.Type == commons.TYPE_U8 {
			return p.newExpr(
				KIND_CAST,
				commons.TYPE_I32,
				expr,
				expr,
			)
		}

	case commons.TYPE_I64:
		if expr.Type == commons.TYPE_I32 ||
			expr.Type == commons.TYPE_I16 ||
			expr.Type == commons.TYPE_I8 ||
			expr.Type == commons.TYPE_U16 ||
			expr.Type == commons.TYPE_U8 {
			return p.newExpr(
				KIND_CAST,
				commons.TYPE_I64,
				expr,
				expr,
			)
		}

	case commons.TYPE_F32:
		if expr.Type == commons.TYPE_I32 ||
			expr.Type == commons.TYPE_I64 ||
			expr.Type == commons.TYPE_I16 ||
			expr.Type == commons.TYPE_I8 ||
			expr.Type == commons.TYPE_U16 ||
			expr.Type == commons.TYPE_U8 {
			return p.newExpr(
				KIND_CAST,
				commons.TYPE_F32,
				expr,
				expr,
			)
		}

	case commons.TYPE_F64:
		if expr.Type == commons.TYPE_I32 ||
			expr.Type == commons.TYPE_I64 ||
			expr.Type == commons.TYPE_F32 ||
			expr.Type == commons.TYPE_I16 ||
			expr.Type == commons.TYPE_I8 ||
			expr.Type == commons.TYPE_U16 ||
			expr.Type == commons.TYPE_U8 {
			return p.newExpr(
				KIND_CAST,
				commons.TYPE_F64,
				expr,
				expr,
			)
		}
	}

	// Allow conversion between string and array (u8[]) via explicit cast node
	if finalType == commons.TYPE_ARRAY && expr.Type == commons.TYPE_STRING {
		return p.newExpr(
			KIND_CAST,
			commons.TYPE_ARRAY,
			expr,
			expr,
		)
	}

	if finalType == commons.TYPE_STRING && expr.Type == commons.TYPE_ARRAY {
		return p.newExpr(
			KIND_CAST,
			commons.TYPE_STRING,
			expr,
			expr,
		)
	}

	commons.CrashOut(
		fmt.Sprintf(
			"cannot implicitly convert %q to %q",
			expr.Type.String(),
			finalType.String(),
		),
		p.lexer.File_path,
		expr.Line,
		expr.Column,
		commons.CRASH_ERROR,
	)

	os.Exit(1)
	return &Expr{}
}

func (p *Parser) parsePostfix() *Expr {
	expr := p.parsePrimary()

	for {
		switch p.lexer.PeekToken().Kind {
		case lexer.TOKEN_LSPAREN:
			expr = p.parseSliceExpr(expr)
		case lexer.TOKEN_DOT:
			expr = p.parseFieldExpr(expr)
		case lexer.TOKEN_INC, lexer.TOKEN_DEC:
			return p.parsePostINCDEC(expr)
		default:
			return expr
		}
	}
}

func (p *Parser) parseSliceExpr(base *Expr) *Expr {
	if base.Type != commons.TYPE_ARRAY && base.Type != commons.TYPE_STRING {
		commons.CrashOut(
			fmt.Sprintf("can't slice type %q", base.Type.String()),
			p.lexer.File_path,
			base.Line,
			base.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_LSPAREN)
	indexExpr := p.parseSliceIndex()

	if indexExpr.IsConstant() {
		indexExpr = p.evalConst(indexExpr)
		index := indexExpr.ValueInt
		if base.ValueSymbol != nil {
			size := base.ValueSymbol.Size
			if index < 0 || int(index) >= size {
				commons.CrashOut(
					fmt.Sprintf("array index %d out of bounds for array %q of size %d", index, base.ValueSymbol.Name, size),
					p.lexer.File_path,
					indexExpr.Line,
					indexExpr.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
		}
	}
	p.expectKind(lexer.TOKEN_RSPAREN)

	// Prefer per-element initializer info when the index is constant.
	// For strings, the element type is u8 and there is no symbol.
	var resultType commons.ExprType
	var resultSymbol *Symbol
	if base.Type == commons.TYPE_STRING {
		resultType = commons.TYPE_U8
		resultSymbol = nil
	} else {
		resultType, resultSymbol = p.arrayElementInfo(base)
	}
	if indexExpr.IsConstant() && base.ValueSymbol != nil && base.ValueSymbol.Value != nil {
		idx := int(indexExpr.ValueInt)
		if idx >= 0 && idx < len(base.ValueSymbol.Value.Children) {
			child := base.ValueSymbol.Value.Children[idx]
			if child != nil {
				// If the stored child has its own symbol (e.g., a nested array),
				// prefer that symbol/type for subsequent chained indexing.
				if child.ValueSymbol != nil {
					resultSymbol = child.ValueSymbol
					resultType = child.Type
				} else {
					resultType = child.Type
				}
			}
		}
	}

	getExpr := p.newExpr(KIND_VARSLICEGET, resultType, base, base, indexExpr)
	getExpr.ValueString = base.ValueString
	getExpr.ValueSymbol = resultSymbol
	getExpr.ValueType = resultType

	// Disallow assignment to string elements (strings are dynamic/immutable)
	if _, found := p.optionalExpectKind(lexer.TOKEN_EQUAL); found {
		if base.Type == commons.TYPE_STRING {
			commons.CrashOut(
				"cannot assign to string element",
				p.lexer.File_path,
				base.Line,
				base.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		if !p.assignAllowed() {
			commons.CrashOut("assignment is not allowed in this context",
				p.lexer.File_path, base.Line, base.Column, commons.CRASH_ERROR)
			os.Exit(1)
		}
		ctx := &Context{Kind: CONTEXT_VARSET, Type: resultType, VarName: getExpr.ValueString}
		// If index is a constant, record it in the context as a hint for
		// downstream processing (e.g., when assigning to nested fields).
		if indexExpr.IsConstant() {
			ctx.VarIndex = int(indexExpr.ValueInt)
		}
		p.newContext(p.CurrentContext, ctx)
		value := p.parseRValue()
		value = p.implicitCast(value, resultType)
		value.checkType(resultType)
		p.endContext()

		assign := p.newExpr(KIND_VARSLICEASSIGN, resultType, base, getExpr, value)
		assign.ValueString = getExpr.ValueString
		assign.ValueSymbol = getExpr.ValueSymbol
		return assign
	}

	return getExpr
}

func (p *Parser) arrayElementInfo(base *Expr) (commons.ExprType, *Symbol) {
	if base.ValueSymbol == nil {
		return base.ValueType, nil
	}
	return base.ValueSymbol.Element, base.ValueSymbol.ElementSym
}

func (p *Parser) fieldTypeDescriptor(field *Field) *Symbol {
	switch field.Type {
	case commons.TYPE_STRUCT:
		return field.TypeSymbol
	case commons.TYPE_ARRAY:
		return &Symbol{
			Type:       commons.TYPE_ARRAY,
			Element:    field.Element,
			ElementSym: field.ElementSym,
			Size:       field.Size,
			Kind:       SYMBOL_TYPE,
		}
	default:
		return nil
	}
}

func (p *Parser) newZeroValueArray(field *Field, root lexer.Token) *Expr {
	// create a unique anonymous name for the temp array
	name := fmt.Sprintf("__anon_arr_%d", p.NextSymbolID)
	token := lexer.Token{Val_string: name, Line: root.Line, Column: root.Column}

	values := make([]*Expr, field.Size)
	for i := 0; i < field.Size; i++ {
		values[i] = p.newZeroValue(field.Element, field.ElementSym, root)
	}

	return p.makeArray(token, field.Size, field.Element, field.ElementSym, values...)
}

func (p *Parser) parseFieldExpr(base *Expr) *Expr {
	p.expectKind(lexer.TOKEN_DOT)
	fieldName := p.expectKind(lexer.TOKEN_ID)

	if base.Type != commons.TYPE_STRUCT {
		commons.CrashOut(
			fmt.Sprintf("type %q is not a struct", base.Type.String()),
			p.lexer.File_path,
			base.Line,
			base.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	baseTypeSym := base.ValueSymbol
	if baseTypeSym == nil {
		if base.ValueContext != nil {
			baseTypeSym = base.ValueContext.Symbols[base.ValueString]
		}
	}
	if baseTypeSym == nil && base.Children != nil && len(base.Children) > 0 {
		if inner := base.Children[0]; inner != nil && inner.ValueSymbol != nil {
			baseTypeSym = inner.ValueSymbol
		}
	}
	if baseTypeSym == nil || (baseTypeSym.Type != commons.TYPE_STRUCT && baseTypeSym.Type != commons.TYPE_ARRAY) {
		commons.CrashOut(
			"missing struct type information for field access",
			p.lexer.File_path,
			base.Line,
			base.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	curExpr := base
	typeSym := baseTypeSym.TypeSymbol
	if typeSym == nil && baseTypeSym.Type == commons.TYPE_STRUCT {
		typeSym = baseTypeSym
	}
	for {
		idx, field := p.findField(typeSym, fieldName.Val_string)
		if field == nil {
			commons.CrashOut(
				fmt.Sprintf("struct %q has no field %q", typeSym.Name, fieldName.Val_string),
				p.lexer.File_path,
				fieldName.Line,
				fieldName.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		node := p.newExpr(KIND_VARFIELDGET, field.Type, base, curExpr)
		node.ValueInt = int32(idx)
		node.ValueSymbol = p.fieldTypeDescriptor(field)
		node.ValueString = field.Name
		curExpr = node

		if p.lexer.PeekToken().Kind == lexer.TOKEN_DOT {
			p.expectKind(lexer.TOKEN_DOT)
			fieldName = p.expectKind(lexer.TOKEN_ID)
			if field.Type != commons.TYPE_STRUCT {
				commons.CrashOut(
					fmt.Sprintf("field %q of struct %q is not a struct", field.Name, typeSym.Name),
					p.lexer.File_path,
					fieldName.Line,
					fieldName.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			typeSym = field.TypeSymbol
			if typeSym == nil {
				commons.CrashOut(
					"missing nested struct type information",
					p.lexer.File_path,
					fieldName.Line,
					fieldName.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			continue
		}
		break
	}

	if _, found := p.optionalExpectKind(lexer.TOKEN_EQUAL); found {
		if !p.assignAllowed() {
			commons.CrashOut("assignment is not allowed in this context", p.lexer.File_path, base.Line, base.Column, commons.CRASH_ERROR)
			os.Exit(1)
		}

		// Use the root base variable name as the VarName for the assignment
		// context so downstream code has a consistent hint about which
		// variable is being written. Also propagate any constant index if
		// the base is a slice access.
		varName := base.ValueString
		varIndex := -1
		if base != nil && base.Kind == KIND_VARSLICEGET && len(base.Children) >= 3 {
			if base.Children[2].IsConstant() {
				varIndex = int(base.Children[2].ValueInt)
			}
		}

		ctx := &Context{Kind: CONTEXT_VARSET, Type: curExpr.Type, VarName: varName}
		if varIndex >= 0 {
			ctx.VarIndex = varIndex
		}
		p.newContext(p.CurrentContext, ctx)
		value := p.parseRValue()
		value = p.implicitCast(value, curExpr.Type)
		value.checkType(curExpr.Type)
		p.endContext()

		assign := p.newExpr(KIND_VARFIELDASSIGN, curExpr.Type, base, curExpr, value)
		assign.ValueString = curExpr.ValueString
		assign.ValueSymbol = curExpr.ValueSymbol
		return assign
	}

	return curExpr
}

func (p *Parser) parsePow() *Expr {
	left := p.parsePostfix()

	if p.lexer.PeekToken().Kind != lexer.TOKEN_POW {
		return left
	}

	tok := p.expectKind(lexer.TOKEN_POW)
	right := p.parseUnary()

	finalType := commons.TYPE_F64

	left = p.implicitCast(left, finalType)
	right = p.implicitCast(right, finalType)

	return p.newExpr(
		KIND_POW,
		finalType,
		tok,
		left,
		right,
	)
}

func (p *Parser) parseMulDivMod() *Expr {
	left := p.parseUnary()

	for {
		switch p.lexer.PeekToken().Kind {

		case lexer.TOKEN_MUL:
			p.expectKind(lexer.TOKEN_MUL)

			right := p.parseUnary()
			finalType := p.getFinalBinopType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			p.expectType(left, finalType)
			p.expectType(right, finalType)

			left = p.newExpr(
				KIND_MUL,
				finalType,
				left,
				left,
				right,
			)

		case lexer.TOKEN_DIV:
			p.expectKind(lexer.TOKEN_DIV)

			right := p.parseUnary()

			finalType := p.getFinalBinopType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			p.expectType(left, finalType)
			p.expectType(right, finalType)

			left = p.newExpr(
				KIND_DIV,
				finalType,
				left,
				left,
				right,
			)

		case lexer.TOKEN_MOD:
			p.expectKind(lexer.TOKEN_MOD)

			right := p.parseUnary()
			finalType := p.getFinalBinopType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			p.expectType(left, finalType)
			p.expectType(right, finalType)

			left = p.newExpr(
				KIND_MOD,
				finalType,
				left,
				left,
				right,
			)

		default:
			return left
		}
	}
}

func (p *Parser) parseAddSub() *Expr {
	left := p.parseMulDivMod()

	for {
		switch p.lexer.PeekToken().Kind {

		case lexer.TOKEN_PLUS:
			p.expectKind(lexer.TOKEN_PLUS)

			right := p.parseMulDivMod()

			finalType := p.getFinalBinopType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			p.expectType(left, finalType)
			p.expectType(right, finalType)

			if left.Type == commons.TYPE_STRING && right.Type == commons.TYPE_STRING {
				left = p.newExpr(
					KIND_STRCAT,
					commons.TYPE_STRING,
					left,
					left,
					right,
				)
			} else {
				left = p.newExpr(
					KIND_ADD,
					finalType,
					left,
					left,
					right,
				)
			}

		case lexer.TOKEN_MINUS:
			p.expectKind(lexer.TOKEN_MINUS)

			right := p.parseMulDivMod()
			finalType := p.getFinalBinopType(left, right)

			left = p.implicitCast(left, finalType)
			right = p.implicitCast(right, finalType)

			p.expectType(left, finalType)
			p.expectType(right, finalType)

			left = p.newExpr(
				KIND_SUB,
				finalType,
				left,
				left,
				right,
			)

		default:
			return left
		}
	}
}

func (p *Parser) parseParen() *Expr {
	p.expectKind(lexer.TOKEN_LRPAREN)

	expr := p.parseRValue()

	p.expectKind(lexer.TOKEN_RRPAREN)

	return expr
}

func (p *Parser) parseReturn() *Expr {
	// freeArena nodes are emitted by the code generator when needed; remove
	// ad-hoc insertion here to avoid double-free and keep parser pure.

	p.expectKind(lexer.TOKEN_RETURN)

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_RETURN,
		Type: commons.TYPE_UNDEFINED,
	})

	value := p.parseRValue()

	value.checkType(commons.TYPE_I64, commons.TYPE_I32, commons.TYPE_I16, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_U8, commons.TYPE_BOOL)

	p.expectKind(lexer.TOKEN_ENDLINE)

	p.endContext()
	return p.newExpr(KIND_RETURN, commons.TYPE_U0, value, value)
}

func (p *Parser) implicitReturn() *Expr {
	tok := p.expectKind(lexer.TOKEN_EOF)

	value := p.newExpr(KIND_LONG, commons.TYPE_I64, tok)
	value.ValueLong = 0

	return p.newExpr(KIND_RETURN, commons.TYPE_U0, value, value)
}

func (p *Parser) freeArena() *Expr {
	p.HasFreeArena = true
	return p.newExpr(KIND_FREEARENA, commons.TYPE_U0, &Expr{})
}

func (p *Parser) parseDump() *Expr {
	p.expectKind(lexer.TOKEN_DUMP)

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_DUMP,
		Type: commons.TYPE_UNDEFINED,
	})

	value := p.parseRValue()

	if value.IsConstant() {
		value = p.evalConst(value)
	}

	value.checkType(
		commons.TYPE_I32, commons.TYPE_I64, // Full Number
		commons.TYPE_F64, commons.TYPE_F32, // Decimal
		commons.TYPE_I16, commons.TYPE_I8, // Smaller
		commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
		commons.TYPE_STRING, commons.TYPE_BOOL, // "Aliases"
	)

	p.expectKind(lexer.TOKEN_ENDLINE)

	p.HasDump = true

	p.endContext()
	return p.newExpr(KIND_DUMP, commons.TYPE_U0, value, value)
}

func (p *Parser) newZeroValue(
	typ commons.ExprType,
	typeSym *Symbol,
	root lexer.Token,
) *Expr {
	expr := &Expr{
		Type:   typ,
		Line:   root.Line,
		Column: root.Column,
		Parser: p,
	}

	switch typ {
	case commons.TYPE_I32:
		expr.Kind = KIND_INT
		expr.ValueInt = 0

	case commons.TYPE_I64:
		expr.Kind = KIND_LONG
		expr.ValueLong = 0

	case commons.TYPE_F32:
		expr.Kind = KIND_FLOAT
		expr.ValueFloat = 0

	case commons.TYPE_F64:
		expr.Kind = KIND_DOUBLE
		expr.ValueDouble = 0

	case commons.TYPE_I8,
		commons.TYPE_U8,
		commons.TYPE_I16,
		commons.TYPE_U16:

		expr.Kind = KIND_INT
		expr.ValueInt = 0

	case commons.TYPE_STRING:
		expr.Kind = KIND_STRING
		expr.ValueLong = -1
		expr.ValueString = ""

	case commons.TYPE_STRUCT:
		if typeSym == nil {
			commons.CrashOut(
				"missing struct type information for zero-init",
				p.lexer.File_path, root.Line, root.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		expr.Kind = KIND_STRUCT
		expr.ValueSymbol = typeSym

		for _, field := range typeSym.Fields {
			if field.Init != nil {
				expr.Children = append(expr.Children, field.Init)
				continue
			}
			if field.Type == commons.TYPE_ARRAY {
				expr.Children = append(expr.Children, p.newZeroValueArray(field, root))
				continue
			}
			expr.Children = append(
				expr.Children,
				p.newZeroValue(field.Type, field.TypeSymbol, root),
			)
		}

	case commons.TYPE_ARRAY:
		// For anonymous zero-init arrays, create an anonymous array variable
		// in the current context and return its initializer expression.
		// This case is hit only when callers pass TYPE_ARRAY directly; prefer
		// using newZeroValueArray for field-specific array zero-init.
		commons.CrashOut(
			"zero-init for standalone array without element info is not supported",
			p.lexer.File_path,
			root.Line,
			root.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)

	default:
		commons.CrashOut(
			fmt.Sprintf(
				"cannot zero-initialize type %q",
				typ.String(),
			),
			p.lexer.File_path,
			root.Line,
			root.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	return expr
}

func (p *Parser) makeArray(name lexer.Token, size int, Type commons.ExprType, elemSym *Symbol, values ...*Expr) *Expr {
	sym, _ := p.CurrentContext.Declare(name.Val_string, &Symbol{
		Type:       commons.TYPE_ARRAY,
		Element:    Type,
		ElementSym: elemSym,
		Size:       size,
		Kind:       SYMBOL_VAR,
	})

	expr := p.newExpr(
		KIND_VARINIT,
		commons.TYPE_ARRAY,
		name,
		values...,
	)

	expr.ValueType = Type
	expr.ValueInt = int32(size)
	expr.ValueString = name.Val_string
	expr.ValueSymbol = sym
	// keep a reference from the symbol to its initializer expression
	sym.Value = expr
	return expr
}

func (p *Parser) parseSetArrayValues(name lexer.Token, size int, inferSize bool, finalType commons.ExprType, elemSym *Symbol) *Expr {

	// [
	p.expectKind(lexer.TOKEN_LSPAREN)

	var values []*Expr
	// Parse explicit elements
	for {
		if p.lexer.PeekToken().Kind == lexer.TOKEN_RSPAREN {
			break
		}

		if len(values) >= size && !inferSize {
			commons.CrashOut(
				fmt.Sprintf(
					"can't assign more than %d elements to this array",
					size,
				),
				p.lexer.File_path,
				name.Line,
				name.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		p.newContext(p.CurrentContext, &Context{Kind: CONTEXT_VARSET, Type: finalType, VarName: name.Val_string, VarIndex: len(values)})

		value := p.parseRValue()

		value.checkType(
			commons.TYPE_I32, commons.TYPE_I64, // Full Number
			commons.TYPE_F64, commons.TYPE_F32, // Decimal
			commons.TYPE_I16, commons.TYPE_I8, // Smaller
			commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
			commons.TYPE_STRING, commons.TYPE_BOOL, commons.TYPE_STRUCT, commons.TYPE_ARRAY, // "Aliases"
		)

		p.endContext()

		// First element determines element type. Keep the symbol of the nested
		// array/struct element so later slice/field access can recover the proper
		// type metadata.
		if finalType == commons.TYPE_UNDEFINED {
			finalType = value.Type
			if value.ValueSymbol != nil && (value.Type == commons.TYPE_STRUCT || value.Type == commons.TYPE_ARRAY) {
				elemSym = value.ValueSymbol
			}
		} else {
			value = p.implicitCast(value, finalType)
			value.checkType(finalType)
		}

		values = append(values, value)

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
	}

	if inferSize {
		size = len(values)
	}

	p.expectKind(lexer.TOKEN_RSPAREN)

	// Empty array
	if finalType == commons.TYPE_UNDEFINED {
		commons.CrashOut(
			"cannot infer element type of empty array",
			p.lexer.File_path,
			name.Line,
			name.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	// Fill remaining elements
	for len(values) < size {
		values = append(values, p.newZeroValue(finalType, elemSym, name))
	}

	expr := p.makeArray(name, size, finalType, elemSym, values...)

	return expr
}

func (p *Parser) parseArrayLiteral() *Expr {
	// use the generic set-array values parser with anonymous name
	anon := lexer.Token{Val_string: fmt.Sprintf("__anon_arr_%d", p.NextSymbolID), Line: p.lexer.PeekToken().Line, Column: p.lexer.PeekToken().Column}
	p.NextSymbolID++
	expr := p.parseSetArrayValues(anon, 0, true, commons.TYPE_UNDEFINED, nil)
	return expr
}

func (p *Parser) parseConstSlice() *Expr {
	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_SLICE,
		Type: commons.TYPE_I32,
	})

	idxExpr := p.parseRValue()

	if !idxExpr.IsConstant() {
		commons.CrashOut(
			"array size must be compile-time evaluable",
			p.lexer.File_path,
			idxExpr.Line,
			idxExpr.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	idxExpr = p.evalConst(idxExpr)
	idxExpr.checkType(commons.TYPE_I32)

	p.endContext()
	return idxExpr
}

func (p *Parser) parseSliceIndex() *Expr {
	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_SLICE,
		Type: commons.TYPE_I32,
	})

	indexExpr := p.parseRValue()
	indexExpr.checkType(commons.TYPE_I32)

	p.endContext()
	return indexExpr
}

func (p *Parser) parseSetArray(name lexer.Token) []*Expr {
	// x[SIZE]
	p.expectKind(lexer.TOKEN_LSPAREN)

	var size int
	inferSize := false
	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_RSPAREN:
		inferSize = true
		size = 0
	default:
		sizeExpr := p.parseConstSlice()

		size = int(sizeExpr.ValueInt)

		if size < 0 {
			commons.CrashOut(
				"array size cannot be negative",
				p.lexer.File_path,
				sizeExpr.Line,
				sizeExpr.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
	}

	p.expectKind(lexer.TOKEN_RSPAREN)
	p.expectKind(lexer.TOKEN_EQUAL)

	expr := p.parseSetArrayValues(name, size, inferSize, commons.TYPE_UNDEFINED, nil)
	return []*Expr{expr}
}

func (p *Parser) parseType() (Type commons.ExprType, isArray bool, arraySize int, typeSym *Symbol) {
	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_ID:
		id := p.expectKind(lexer.TOKEN_ID)
		sym, ok := p.CurrentContext.Lookup(id.Val_string)
		if !ok || sym.Kind != SYMBOL_TYPE {
			commons.CrashOut(
				fmt.Sprintf("unknown type: %q", id.Val_string),
				p.lexer.File_path,
				id.Line,
				id.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		if sym.Type == commons.TYPE_ARRAY {
			isArray = true
			Type = sym.Element
			arraySize = sym.Size
			typeSym = sym.ElementSym
		} else {
			Type = sym.Type
			typeSym = sym
		}

	case lexer.TOKEN_TYPE:
		Type = p.expectKind(lexer.TOKEN_TYPE).Val_type

		var dims []int
		for {
			if _, found := p.optionalExpectKind(lexer.TOKEN_LSPAREN); !found {
				break
			}
			sizeExpr := p.parseConstSlice()
			p.expectKind(lexer.TOKEN_RSPAREN)
			dims = append(dims, int(sizeExpr.ValueInt))
		}

		if len(dims) > 0 {
			isArray = true
			arraySize = dims[0]

			// build nested symbols from inner to outer
			var innerSym *Symbol
			elemType := Type
			for d := len(dims) - 1; d >= 1; d-- {
				innerSym = &Symbol{
					Type:       commons.TYPE_ARRAY,
					Element:    elemType,
					ElementSym: innerSym,
					Size:       dims[d],
					Kind:       SYMBOL_TYPE,
				}
				elemType = commons.TYPE_ARRAY
			}
			typeSym = innerSym
			Type = elemType
		}
	}
	return
}

func (p *Parser) parseSet() []*Expr {
	p.expectKind(lexer.TOKEN_SET)

	var names []lexer.Token
	for {
		name := p.expectKind(lexer.TOKEN_ID)

		_, exists := p.CurrentContext.Lookup(name.Val_string)
		if exists {
			commons.CrashOut(fmt.Sprintf("symbol with name %q already exists", name.Val_string), p.lexer.File_path, name.Line, name.Column, commons.CRASH_ERROR)
			os.Exit(1)
		}

		if p.lexer.PeekToken().Kind == lexer.TOKEN_LSPAREN {
			if len(names) == 0 {
				expr := p.parseSetArray(name)
				return expr
			} else {
				commons.CrashOut("array declarations cannot be combined with multiple variables", p.lexer.File_path, name.Line, name.Column, commons.CRASH_ERROR)
				os.Exit(1)
			}
		}

		names = append(names, name)

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
	}

	if p.lexer.PeekToken().Kind == lexer.TOKEN_TYPE || p.lexer.PeekToken().Kind == lexer.TOKEN_ID {
		result := make([]*Expr, 0, len(names))

		Type, isArray, Size, typeSym := p.parseType()
		for _, name := range names {
			var expr *Expr
			if isArray {
				values := make([]*Expr, Size)

				for i := range values {
					values[i] = p.newZeroValue(Type, typeSym, name)
				}

				expr = p.makeArray(name, Size, Type, typeSym, values...)
			} else {
				sym, _ := p.CurrentContext.Declare(name.Val_string, &Symbol{
					Type:       Type,
					Kind:       SYMBOL_VAR,
					TypeSymbol: typeSym,
				})

				expr = p.newExpr(
					KIND_VARINIT,
					Type,
					name,
					p.newZeroValue(Type, typeSym, name),
				)
				expr.ValueString = name.Val_string
				expr.ValueSymbol = sym
			}

			result = append(result, expr)
		}

		return result
	}

	p.expectKind(lexer.TOKEN_EQUAL)

	var values []*Expr
	for {
		i := len(values)
		if i >= len(names) {
			commons.CrashOut(
				fmt.Sprintf(
					"expected %d values, got %d",
					len(names),
					i+1,
				),
				p.lexer.File_path,
				names[0].Line,
				names[0].Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		if p.lexer.PeekToken().Kind == lexer.TOKEN_LSPAREN {
			values = append(values, p.parseSetArrayValues(names[i], 0, true, commons.TYPE_UNDEFINED, nil))
		} else {
			p.newContext(p.CurrentContext, &Context{
				Kind:    CONTEXT_VARSET,
				Type:    commons.TYPE_UNDEFINED,
				VarName: names[i].Val_string,
			})

			value := p.parseRValue()

			value.checkType(
				commons.TYPE_I32, commons.TYPE_I64, // Full Number
				commons.TYPE_F64, commons.TYPE_F32, // Decimal
				commons.TYPE_I16, commons.TYPE_I8, // Smaller
				commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
				commons.TYPE_STRING, commons.TYPE_BOOL, commons.TYPE_STRUCT, commons.TYPE_ARRAY, // "Aliases"
			)

			p.endContext()

			values = append(values, value)
		}

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
	}

	if len(names) != len(values) {
		commons.CrashOut(
			fmt.Sprintf(
				"expected %d values, got %d",
				len(names),
				len(values),
			),
			p.lexer.File_path,
			names[0].Line,
			names[0].Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	result := make([]*Expr, 0, len(names))

	for i, name := range names {
		value := values[i]

		if value.Type != commons.TYPE_ARRAY {
			var typeSym *Symbol
			if value.Type == commons.TYPE_STRUCT {
				typeSym = value.ValueSymbol
			}

			sym, _ := p.CurrentContext.Declare(name.Val_string, &Symbol{
				Type:       value.Type,
				Kind:       SYMBOL_VAR,
				TypeSymbol: typeSym,
			})

			if value != nil && value.Type == commons.TYPE_STRING {
				sym.Size = len(value.ValueString)
			}

			value = p.newExpr(
				KIND_VARINIT,
				value.Type,
				value,
				value,
			)
			value.ValueString = name.Val_string
			value.ValueSymbol = sym
		}

		result = append(result, value)
	}

	return result
}

// #region const-parsing
func (p *Parser) newConstU8(value int32, root *Expr) *Expr {
	return &Expr{
		Kind:     KIND_INT,
		Type:     commons.TYPE_U8,
		ValueInt: value,
		Line:     root.Line,
		Column:   root.Column,
		Parser:   p,
	}
}

func (p *Parser) newConstU16(value int32, root *Expr) *Expr {
	return &Expr{
		Kind:     KIND_INT,
		Type:     commons.TYPE_U16,
		ValueInt: value,
		Line:     root.Line,
		Column:   root.Column,
		Parser:   p,
	}
}

func (p *Parser) newConstI8(value int32, root *Expr) *Expr {
	return &Expr{
		Kind:     KIND_INT,
		Type:     commons.TYPE_I8,
		ValueInt: value,
		Line:     root.Line,
		Column:   root.Column,
		Parser:   p,
	}
}

func (p *Parser) newConstI16(value int32, root *Expr) *Expr {
	return &Expr{
		Kind:     KIND_INT,
		Type:     commons.TYPE_I16,
		ValueInt: value,
		Line:     root.Line,
		Column:   root.Column,
		Parser:   p,
	}
}

func (p *Parser) newConstInt(value int32, root *Expr) *Expr {
	return &Expr{
		Kind:     KIND_INT,
		Type:     commons.TYPE_I32,
		ValueInt: value,
		Line:     root.Line,
		Column:   root.Column,
		Parser:   p,
	}
}

func (p *Parser) newConstLong(value int64, root *Expr) *Expr {
	return &Expr{
		Kind:      KIND_LONG,
		Type:      commons.TYPE_I64,
		ValueLong: value,
		Line:      root.Line,
		Column:    root.Column,
		Parser:    p,
	}
}

func (p *Parser) newConstFloat(value float32, root *Expr) *Expr {
	return &Expr{
		Kind:       KIND_FLOAT,
		Type:       commons.TYPE_F32,
		ValueFloat: value,
		Line:       root.Line,
		Column:     root.Column,
		Parser:     p,
	}
}

func (p *Parser) newConstDouble(value float64, root *Expr) *Expr {
	return &Expr{
		Kind:        KIND_DOUBLE,
		Type:        commons.TYPE_F64,
		ValueDouble: value,
		Line:        root.Line,
		Column:      root.Column,
		Parser:      p,
	}
}

func (p *Parser) newConstString(value string, root *Expr) *Expr {
	var id int
	if str, exists := p.StringsLookup[value]; exists {
		id = str
	} else {
		id = len(p.Strings)

		p.Strings[id] = value
		p.StringsLookup[value] = id
	}

	return &Expr{
		Kind:        KIND_STRING,
		Type:        commons.TYPE_STRING,
		ValueString: value,
		ValueLong:   int64(id),
		Line:        root.Line,
		Column:      root.Column,
		Parser:      p,
	}
}

func (p *Parser) newConstBool(value int32, root *Expr) *Expr {
	return &Expr{
		Kind:     KIND_BOOL,
		Type:     commons.TYPE_BOOL,
		ValueInt: value,
		Line:     root.Line,
		Column:   root.Column,
		Parser:   p,
	}
}

func (p *Parser) Btoi(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func (p *Parser) evalConst(expr *Expr) *Expr {
	switch expr.Kind {
	case KIND_INT:
		return expr

	case KIND_LONG:
		return expr

	case KIND_FLOAT:
		return expr

	case KIND_DOUBLE:
		return expr

	case KIND_STRING:
		return expr

	case KIND_BOOL:
		return expr

	case KIND_ADD:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch expr.Type {
		case commons.TYPE_I32:
			return &Expr{
				Kind:     KIND_INT,
				Type:     commons.TYPE_I32,
				ValueInt: left.ValueInt + right.ValueInt,
				Line:     expr.Line,
				Column:   expr.Column,
				Parser:   p,
			}

		case commons.TYPE_I64:
			return &Expr{
				Kind:      KIND_LONG,
				Type:      commons.TYPE_I64,
				ValueLong: left.ValueLong + right.ValueLong,
				Line:      expr.Line,
				Column:    expr.Column,
				Parser:    p,
			}

		case commons.TYPE_F32:
			return &Expr{
				Kind:       KIND_FLOAT,
				Type:       commons.TYPE_F32,
				ValueFloat: left.ValueFloat + right.ValueFloat,
				Line:       expr.Line,
				Column:     expr.Column,
				Parser:     p,
			}

		case commons.TYPE_F64:
			return &Expr{
				Kind:        KIND_DOUBLE,
				Type:        commons.TYPE_F64,
				ValueDouble: left.ValueDouble + right.ValueDouble,
				Line:        expr.Line,
				Column:      expr.Column,
				Parser:      p,
			}
		}

	case KIND_SUB:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch expr.Type {
		case commons.TYPE_I32:
			return p.newConstInt(
				left.ValueInt-right.ValueInt,
				expr,
			)

		case commons.TYPE_I64:
			return p.newConstLong(
				left.ValueLong-right.ValueLong,
				expr,
			)

		case commons.TYPE_F32:
			return p.newConstFloat(
				left.ValueFloat-right.ValueFloat,
				expr,
			)

		case commons.TYPE_F64:
			return p.newConstDouble(
				left.ValueDouble-right.ValueDouble,
				expr,
			)
		}

	case KIND_MUL:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch expr.Type {
		case commons.TYPE_I32:
			return p.newConstInt(
				left.ValueInt*right.ValueInt,
				expr,
			)

		case commons.TYPE_I64:
			return p.newConstLong(
				left.ValueLong*right.ValueLong,
				expr,
			)

		case commons.TYPE_F32:
			return p.newConstFloat(
				left.ValueFloat*right.ValueFloat,
				expr,
			)

		case commons.TYPE_F64:
			return p.newConstDouble(
				left.ValueDouble*right.ValueDouble,
				expr,
			)
		}

	case KIND_DIV:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch expr.Type {
		case commons.TYPE_I32:
			if right.ValueInt == 0 {
				commons.CrashOut(
					"division by zero",
					p.lexer.File_path,
					expr.Line,
					expr.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}

			return p.newConstInt(
				left.ValueInt/right.ValueInt,
				expr,
			)

		case commons.TYPE_I64:
			if right.ValueLong == 0 {
				commons.CrashOut(
					"division by zero",
					p.lexer.File_path,
					expr.Line,
					expr.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}

			return p.newConstLong(
				left.ValueLong/right.ValueLong,
				expr,
			)

		case commons.TYPE_F32:
			return p.newConstFloat(
				left.ValueFloat/right.ValueFloat,
				expr,
			)

		case commons.TYPE_F64:
			return p.newConstDouble(
				left.ValueDouble/right.ValueDouble,
				expr,
			)
		}

	case KIND_MOD:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch expr.Type {
		case commons.TYPE_I32:
			if right.ValueInt == 0 {
				commons.CrashOut(
					"division by zero",
					p.lexer.File_path,
					expr.Line,
					expr.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			return p.newConstInt(
				left.ValueInt%right.ValueInt,
				expr,
			)

		case commons.TYPE_I64:
			if right.ValueLong == 0 {
				commons.CrashOut(
					"division by zero",
					p.lexer.File_path,
					expr.Line,
					expr.Column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			return p.newConstLong(
				left.ValueLong%right.ValueLong,
				expr,
			)
		}

	case KIND_POW:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch expr.Type {
		case commons.TYPE_F32:
			return p.newConstFloat(
				float32(math.Pow(
					float64(left.ValueFloat),
					float64(right.ValueFloat),
				)),
				expr,
			)

		case commons.TYPE_F64:
			return p.newConstDouble(
				math.Pow(left.ValueDouble, right.ValueDouble),
				expr,
			)

		default:
			panic(fmt.Sprintf(
				"invalid constant pow type %q",
				expr.Type.String(),
			))
		}

	case KIND_STRCAT:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		if left.Type != commons.TYPE_STRING || right.Type != commons.TYPE_STRING {
			panic("invalid constant string concatenation")
		}

		return p.newConstString(
			left.ValueString+right.ValueString,
			expr,
		)

	case KIND_GT:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch left.Type {
		case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16:
			return p.newConstBool(p.Btoi(left.ValueInt > right.ValueInt), expr)
		case commons.TYPE_I64:
			return p.newConstBool(p.Btoi(left.ValueLong > right.ValueLong), expr)
		case commons.TYPE_F32:
			return p.newConstBool(p.Btoi(left.ValueFloat > right.ValueFloat), expr)
		case commons.TYPE_F64:
			return p.newConstBool(p.Btoi(left.ValueDouble > right.ValueDouble), expr)
		}

	case KIND_GE:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch left.Type {
		case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16:
			return p.newConstBool(p.Btoi(left.ValueInt >= right.ValueInt), expr)
		case commons.TYPE_I64:
			return p.newConstBool(p.Btoi(left.ValueLong >= right.ValueLong), expr)
		case commons.TYPE_F32:
			return p.newConstBool(p.Btoi(left.ValueFloat >= right.ValueFloat), expr)
		case commons.TYPE_F64:
			return p.newConstBool(p.Btoi(left.ValueDouble >= right.ValueDouble), expr)
		}

	case KIND_LT:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch left.Type {
		case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16:
			return p.newConstBool(p.Btoi(left.ValueInt < right.ValueInt), expr)
		case commons.TYPE_I64:
			return p.newConstBool(p.Btoi(left.ValueLong < right.ValueLong), expr)
		case commons.TYPE_F32:
			return p.newConstBool(p.Btoi(left.ValueFloat < right.ValueFloat), expr)
		case commons.TYPE_F64:
			return p.newConstBool(p.Btoi(left.ValueDouble < right.ValueDouble), expr)
		}

	case KIND_LE:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch left.Type {
		case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16:
			return p.newConstBool(p.Btoi(left.ValueInt <= right.ValueInt), expr)
		case commons.TYPE_I64:
			return p.newConstBool(p.Btoi(left.ValueLong <= right.ValueLong), expr)
		case commons.TYPE_F32:
			return p.newConstBool(p.Btoi(left.ValueFloat <= right.ValueFloat), expr)
		case commons.TYPE_F64:
			return p.newConstBool(p.Btoi(left.ValueDouble <= right.ValueDouble), expr)
		}

	case KIND_EQ:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch left.Type {
		case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16, commons.TYPE_BOOL:
			return p.newConstBool(p.Btoi(left.ValueInt == right.ValueInt), expr)
		case commons.TYPE_I64:
			return p.newConstBool(p.Btoi(left.ValueLong == right.ValueLong), expr)
		case commons.TYPE_F32:
			return p.newConstBool(p.Btoi(left.ValueFloat == right.ValueFloat), expr)
		case commons.TYPE_F64:
			return p.newConstBool(p.Btoi(left.ValueDouble == right.ValueDouble), expr)
		}

	case KIND_NEQ:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		switch left.Type {
		case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16, commons.TYPE_BOOL:
			return p.newConstBool(p.Btoi(left.ValueInt != right.ValueInt), expr)
		case commons.TYPE_I64:
			return p.newConstBool(p.Btoi(left.ValueLong != right.ValueLong), expr)
		case commons.TYPE_F32:
			return p.newConstBool(p.Btoi(left.ValueFloat != right.ValueFloat), expr)
		case commons.TYPE_F64:
			return p.newConstBool(p.Btoi(left.ValueDouble != right.ValueDouble), expr)
		}

	case KIND_OR:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		return p.newConstBool(left.ValueInt|right.ValueInt, expr)

	case KIND_AND:
		left := p.evalConst(expr.Children[0])
		right := p.evalConst(expr.Children[1])

		return p.newConstBool(left.ValueInt&right.ValueInt, expr)

	case KIND_NOT:
		left := p.evalConst(expr.Children[0])

		return p.newConstBool(left.ValueInt^1, expr)

	case KIND_CAST:
		value := p.evalConst(expr.Children[0])

		if expr.Type == value.Type {
			return value
		}

		switch expr.Type {
		case commons.TYPE_U8:
			switch value.Type {
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstU8(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstU8(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstU8(int32(value.ValueDouble), expr)
			}
		case commons.TYPE_I8:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstI8(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstI8(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstI8(int32(value.ValueDouble), expr)
			}
		case commons.TYPE_U16:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstU16(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstU16(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstU16(int32(value.ValueDouble), expr)
			}
		case commons.TYPE_I16:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstI16(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstI16(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstI16(int32(value.ValueDouble), expr)
			}
		case commons.TYPE_I32:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstInt(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstInt(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstInt(int32(value.ValueDouble), expr)
			}

		case commons.TYPE_I64:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstLong(int64(value.ValueInt), expr)
			case commons.TYPE_F32:
				return p.newConstLong(int64(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstLong(int64(value.ValueDouble), expr)
			}

		case commons.TYPE_F32:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstFloat(float32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstFloat(float32(value.ValueLong), expr)
			case commons.TYPE_F64:
				return p.newConstFloat(float32(value.ValueDouble), expr)
			}

		case commons.TYPE_F64:
			switch value.Type {
			case commons.TYPE_U8:
				return p.newConstU8(int32(value.ValueInt), expr)
			case commons.TYPE_I8:
				return p.newConstI8(int32(value.ValueInt), expr)
			case commons.TYPE_U16:
				return p.newConstU16(int32(value.ValueInt), expr)
			case commons.TYPE_I16:
				return p.newConstI16(int32(value.ValueInt), expr)
			case commons.TYPE_I32:
				return p.newConstDouble(float64(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstDouble(float64(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstDouble(float64(value.ValueFloat), expr)
			}

		case commons.TYPE_STRING:
			switch value.Type {
			case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16:
				return p.newConstString(fmt.Sprintf("%d", value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstString(fmt.Sprintf("%d", value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstString(fmt.Sprintf("%f", value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstString(fmt.Sprintf("%f", value.ValueDouble), expr)
			}

		case commons.TYPE_BOOL:
			switch value.Type {
			case commons.TYPE_I32, commons.TYPE_U8, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_I16:
				return p.newConstBool(int32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstBool(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstBool(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstBool(int32(value.ValueDouble), expr)
			}
		}
		commons.CrashOut(fmt.Sprintf(
			"compile-time cast not supported from type %q to type %q",
			value.Type.String(),
			expr.Type.String(),
		), p.lexer.File_path, expr.Line, expr.Column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	commons.CrashOut(fmt.Sprintf(
		"cannot evaluate compile-time expression %q",
		expr.Kind.String(),
	), p.lexer.File_path, expr.Line, expr.Column, commons.CRASH_ERROR)
	os.Exit(1)

	panic("UNREACHABLE")
}

func (p *Parser) parseConstSet(name lexer.Token) *Expr {
	p.expectKind(lexer.TOKEN_DOUBLECOLON)

	if _, exists := p.CurrentContext.Lookup(name.Val_string); exists {
		commons.CrashOut(
			fmt.Sprintf(
				"constant symbol %q already exists",
				name.Val_string,
			),
			p.lexer.File_path,
			name.Line,
			name.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	p.newContext(p.CurrentContext, &Context{
		Kind:    CONTEXT_CONSTSET,
		Type:    commons.TYPE_UNDEFINED,
		VarName: name.Val_string,
	})

	value := p.parseRValue()

	if !value.IsConstant() {
		commons.CrashOut(
			"constant expression is not compile-time evaluable",
			p.lexer.File_path,
			value.Line,
			value.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	value = p.evalConst(value)

	p.endContext()

	value.checkType(
		commons.TYPE_I32, commons.TYPE_I64, // Full Number
		commons.TYPE_F64, commons.TYPE_F32, // Decimal
		commons.TYPE_I16, commons.TYPE_I8, // Smaller
		commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
		commons.TYPE_STRING, commons.TYPE_BOOL, // "Aliases"
	)

	sym, _ := p.CurrentContext.Declare(name.Val_string, &Symbol{
		Type:  value.Type,
		Kind:  SYMBOL_CONST,
		Value: value,
	})

	expr := p.newExpr(
		KIND_CONSTSET,
		value.Type,
		value,
		value,
	)

	expr.ValueString = name.Val_string
	expr.ValueSymbol = sym

	return expr
}

//#endregion

func (p *Parser) resolveStructType(id lexer.Token) *Symbol {
	sym, exists := p.CurrentContext.Lookup(id.Val_string)
	if !exists || sym.Kind != SYMBOL_TYPE || sym.Type != commons.TYPE_STRUCT {
		commons.CrashOut(
			fmt.Sprintf("%q is not a struct type", id.Val_string),
			p.lexer.File_path, id.Line, id.Column, commons.CRASH_ERROR,
		)
		os.Exit(1)
	}
	return sym
}

func (p *Parser) parseStructValue(id lexer.Token) *Expr {
	typeSym := p.resolveStructType(id)
	return p.parseStructLiteralBody(typeSym, id)
}

func (p *Parser) isNamedField() bool {
	return p.lexer.PeekToken().Kind == lexer.TOKEN_ID &&
		p.lexer.PeekTokenAt(1).Kind == lexer.TOKEN_COLON
}

func (p *Parser) findField(typeSym *Symbol, name string) (int, *Field) {
	for i, f := range typeSym.Fields {
		if f.Name == name {
			return i, f
		}
	}
	return -1, nil
}

func (p *Parser) parseFieldValue(field *Field, root lexer.Token) *Expr {
	_ = root // Maybe i'll use it
	if field.Type == commons.TYPE_STRUCT {
		return p.parseStructLiteralBody(field.TypeSymbol, p.lexer.PeekToken())
	}

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_VARSET,
		Type: field.Type,
	})

	value := p.parseRValue()
	value = p.implicitCast(value, field.Type)
	value.checkType(field.Type)

	p.endContext()

	// If the field is a fixed-size array and the provided value was an
	// array literal with fewer elements, extend it with zero-values so the
	// initializer matches the declared array size.
	if field.Type == commons.TYPE_ARRAY && value != nil && value.Type == commons.TYPE_ARRAY {
		if value.ValueSymbol != nil {
			// Ensure the anonymous symbol reflects the declared size
			value.ValueSymbol.Size = field.Size
		}
		// Fill missing elements with zero values for the element type
		for len(value.Children) < field.Size {
			value.Children = append(value.Children, p.newZeroValue(field.Element, field.ElementSym, root))
		}
		value.ValueInt = int32(field.Size)
	}
	return value
}

func (p *Parser) parseStructLiteralBody(typeSym *Symbol, root lexer.Token) *Expr {
	p.expectKind(lexer.TOKEN_LCPAREN)

	values := make([]*Expr, len(typeSym.Fields))
	set := make([]bool, len(typeSym.Fields))

	named := p.lexer.PeekToken().Kind != lexer.TOKEN_RCPAREN && p.isNamedField()
	posIndex := 0

	for p.lexer.PeekToken().Kind != lexer.TOKEN_RCPAREN {
		if p.isNamedField() != named {
			commons.CrashOut(
				"cannot mix named and positional fields in struct literal",
				p.lexer.File_path, root.Line, root.Column, commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		var idx int
		var field *Field

		if named {
			nameTok := p.expectKind(lexer.TOKEN_ID)
			p.expectKind(lexer.TOKEN_COLON)

			idx, field = p.findField(typeSym, nameTok.Val_string)
			if field == nil {
				commons.CrashOut(
					fmt.Sprintf("struct %q has no field %q", typeSym.Name, nameTok.Val_string),
					p.lexer.File_path, nameTok.Line, nameTok.Column, commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			if set[idx] {
				commons.CrashOut(
					fmt.Sprintf("field %q already initialized", nameTok.Val_string),
					p.lexer.File_path, nameTok.Line, nameTok.Column, commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
		} else {
			if posIndex >= len(typeSym.Fields) {
				commons.CrashOut(
					fmt.Sprintf("too many values for struct %q", typeSym.Name),
					p.lexer.File_path, root.Line, root.Column, commons.CRASH_ERROR,
				)
				os.Exit(1)
			}
			idx = posIndex
			field = typeSym.Fields[idx]
			posIndex++
		}

		values[idx] = p.parseFieldValue(field, root)
		set[idx] = true

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}
		p.expectKind(lexer.TOKEN_COMMA)
	}

	p.expectKind(lexer.TOKEN_RCPAREN)

	for idx, field := range typeSym.Fields {
		if !set[idx] {
			values[idx] = p.newZeroValue(field.Type, field.TypeSymbol, root)
		}
	}

	expr := p.newExpr(KIND_STRUCT, commons.TYPE_STRUCT, root, values...)
	expr.ValueSymbol = typeSym
	return expr
}

func (p *Parser) parseCast(tok lexer.Token) *Expr {
	var Type commons.ExprType
	switch tok.Kind {
	case lexer.TOKEN_TYPE:
		Type = tok.Val_type
	case lexer.TOKEN_ID:
		sym, ok := p.CurrentContext.Lookup(tok.Val_string)
		if !ok || sym.Kind != SYMBOL_TYPE {
			commons.CrashOut(
				fmt.Sprintf("unknown type: %q", tok.Val_string),
				p.lexer.File_path,
				tok.Line,
				tok.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		if sym.Type == commons.TYPE_STRUCT {
			commons.CrashOut(
				fmt.Sprintf("cannot cast to non-primitive type: %q (%s)", tok.Val_string, sym.Type.String()),
				p.lexer.File_path,
				tok.Line,
				tok.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
		Type = sym.Type
	}

	p.expectKind(lexer.TOKEN_LRPAREN)

	var ctxType commons.ExprType
	if Type != commons.TYPE_STRING {
		ctxType = Type
	} else {
		ctxType = commons.TYPE_UNDEFINED
	}

	if Type == commons.TYPE_BOOL {
		commons.CrashOut(
			"casting to boolean may produce unexpected behavior",
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_WARN,
		)
	}

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_CAST,
		Type: ctxType,
	})

	Value := p.parseRValue()

	p.endContext()

	p.expectKind(lexer.TOKEN_RRPAREN)

	return p.newExpr(
		KIND_CAST,
		Type,
		Value,
		Value,
	)
}

func (p *Parser) parseBlock(endKinds ...lexer.TokenKind) *Expr {
	start := p.lexer.PeekToken()

	body := &Expr{
		Kind:     KIND_BODY,
		Type:     commons.TYPE_UNDEFINED,
		Parser:   p,
		Line:     start.Line,
		Column:   start.Column,
		Children: make([]*Expr, 0),
	}

	for {
		tok := p.lexer.PeekToken()

		// Check if this token terminates the block.
		for _, endKind := range endKinds {
			if tok.Kind == endKind {
				return body
			}
		}

		// EOF before the expected terminator.
		if tok.Kind == lexer.TOKEN_EOF {
			commons.CrashOut(
				fmt.Sprintf(
					"unexpected EOF, expected %q",
					endKinds[0].String(),
				),
				p.lexer.File_path,
				tok.Line,
				tok.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		body.Children = append(
			body.Children,
			p.parseStatement()...,
		)
	}
}

func (p *Parser) parseIfKind(kind lexer.TokenKind) *Expr {
	tok := p.expectKind(kind)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	// condition
	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_IF,
		Type: commons.TYPE_UNDEFINED,
	})

	condExpr := p.parseRValue()

	if condExpr.IsConstant() {
		condExpr = p.evalConst(condExpr)
	}

	cond := p.implicitCast(
		condExpr,
		commons.TYPE_BOOL,
	)

	cond.checkType(commons.TYPE_BOOL)

	p.endContext()

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	p.expectKind(lexer.TOKEN_THEN)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	// then
	p.newContext(p.CurrentContext, &Context{
		Kind:     CONTEXT_BODY,
		BodyKind: CONTEXT_IF,
	})

	body := p.parseBlock(
		lexer.TOKEN_END,
		lexer.TOKEN_ELSE,
		lexer.TOKEN_ELIF,
	)

	p.endContext()

	var elseExpr *Expr

	switch p.lexer.PeekToken().Kind {

	case lexer.TOKEN_ELIF:
		elseExpr = p.parseIfKind(lexer.TOKEN_ELIF)

	case lexer.TOKEN_ELSE:
		p.expectKind(lexer.TOKEN_ELSE)

		p.optionalExpectKind(lexer.TOKEN_ENDLINE)

		p.newContext(p.CurrentContext, &Context{
			Kind:     CONTEXT_BODY,
			BodyKind: CONTEXT_IF,
		})

		elseExpr = p.parseBlock(lexer.TOKEN_END)

		p.endContext()

		p.expectKind(lexer.TOKEN_END)

	case lexer.TOKEN_END:
		p.expectKind(lexer.TOKEN_END)
	}

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	id := p.newIfID()
	body.ID = id

	ifExpr := p.newExpr(
		KIND_IF,
		commons.TYPE_UNDEFINED,
		tok,
		cond,
		body,
	)

	if elseExpr != nil {
		ifExpr.HasElse = true
		ifExpr.Children = append(ifExpr.Children, elseExpr)
	}

	ifExpr.ID = id

	return ifExpr
}

func (p *Parser) parseIf() *Expr {
	return p.parseIfKind(lexer.TOKEN_IF)
}

func (p *Parser) parseStruct(id lexer.Token) (commons.ExprType, *Symbol) {
	p.expectKind(lexer.TOKEN_STRUCT)
	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	if sym, exists := p.CurrentContext.Lookup(id.Val_string); exists {
		commons.CrashOut(
			fmt.Sprintf("symbol named %q already exists in this context", sym.Name),
			p.lexer.File_path,
			id.Line,
			id.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	sym := &Symbol{
		Type:     commons.TYPE_STRUCT,
		Kind:     SYMBOL_TYPE,
		Fields:   []*Field{},
		FieldMap: make(map[string]*Field),
	}

	for {
		if p.lexer.PeekToken().Kind == lexer.TOKEN_END {
			p.expectKind(lexer.TOKEN_END)
			break
		}

		name := p.expectKind(lexer.TOKEN_ID)

		typ, isArray, arraySize, typeSym := p.parseType()

		field := &Field{
			Name: name.Val_string,
		}
		if isArray {
			field.Type = commons.TYPE_ARRAY
			field.Element = typ
			field.ElementSym = typeSym
			field.Size = arraySize
		} else {
			field.Type = typ
			field.TypeSymbol = typeSym
		}

		if _, exists := sym.FieldMap[field.Name]; exists {
			commons.CrashOut(
				fmt.Sprintf("duplicate field %q in struct %q", field.Name, id.Val_string),
				p.lexer.File_path,
				name.Line,
				name.Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		sym.Fields = append(sym.Fields, field)
		sym.FieldMap[field.Name] = field

		if p.lexer.PeekToken().Kind == lexer.TOKEN_COMMA {
			p.expectKind(lexer.TOKEN_COMMA)
		}

		p.optionalExpectKind(lexer.TOKEN_ENDLINE)
	}

	p.CurrentContext.Declare(id.Val_string, sym)
	return commons.TYPE_STRUCT, sym
}

func (p *Parser) parseTypeDecl() *Expr {
	tok := p.expectKind(lexer.TOKEN_TYPEKW)

	id := p.expectKind(lexer.TOKEN_ID)

	if sym, exists := p.CurrentContext.Lookup(id.Val_string); exists {
		commons.CrashOut(
			fmt.Sprintf("symbol named %q already exists in this context", sym.Name),
			p.lexer.File_path,
			id.Line,
			id.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	var Type commons.ExprType
	var ValueSymbol *Symbol
	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_TYPE, lexer.TOKEN_ID:
		Typ, isArray, arraySize, _ := p.parseType()
		if isArray {
			ValueSymbol = &Symbol{
				Type:    commons.TYPE_ARRAY,
				Kind:    SYMBOL_TYPE,
				Element: Typ,
				Size:    arraySize,
			}
			p.CurrentContext.Declare(id.Val_string, ValueSymbol)
			Type = commons.TYPE_ARRAY
		} else {
			ValueSymbol = &Symbol{
				Type: Typ,
				Kind: SYMBOL_TYPE,
			}
			p.CurrentContext.Declare(id.Val_string, ValueSymbol)
			Type = Typ
		}
	case lexer.TOKEN_STRUCT:
		Type, ValueSymbol = p.parseStruct(id)
	default:
		commons.CrashOut(
			fmt.Sprintf("expected type, type identifier or struct. got %q", p.lexer.PeekToken().Kind.String()),
			p.lexer.File_path,
			p.lexer.PeekToken().Line,
			p.lexer.PeekToken().Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_ENDLINE)

	expr := p.newExpr(KIND_TYPEDECL, Type, tok)
	expr.ValueSymbol = ValueSymbol
	return expr
}

func (p *Parser) parseSwitchCase(id int, caseID int, switchType commons.ExprType, compExpr *Expr) *Expr {
	tok := p.expectKind(lexer.TOKEN_CASE)

	caseExpr := p.newExpr(
		KIND_CASE,
		commons.TYPE_UNDEFINED,
		tok,
	)
	caseExpr.ID = caseID

	for {
		p.newContext(p.CurrentContext, &Context{
			Kind: CONTEXT_CASE,
			Type: switchType,
		})

		value := p.parseRValue()

		value = p.implicitCast(value, switchType)
		value.ID = len(caseExpr.Children)
		p.endContext()

		checkExpr := p.newExpr(KIND_EQ, switchType, value, value, compExpr)
		checkExpr.ID = len(caseExpr.Children)

		caseExpr.Children = append(
			caseExpr.Children,
			checkExpr,
		)

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
	}

	p.expectKind(lexer.TOKEN_COLON)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	p.newContext(p.CurrentContext, &Context{
		Kind:     CONTEXT_BODY,
		BodyKind: CONTEXT_CASE,
		ID:       id,
	})

	body := p.parseBlock(
		lexer.TOKEN_END,
		lexer.TOKEN_CASE,
		lexer.TOKEN_DEFAULT,
	)
	body.ID = id

	p.endContext()

	caseExpr.Children = append(caseExpr.Children, body)

	return caseExpr
}

func (p *Parser) parseSwitchDefault(id int) *Expr {
	tok := p.expectKind(lexer.TOKEN_DEFAULT)

	p.expectKind(lexer.TOKEN_COLON)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	p.newContext(p.CurrentContext, &Context{
		Kind:     CONTEXT_BODY,
		BodyKind: CONTEXT_DEFAULT,
		ID:       id,
	})

	body := p.parseBlock(
		lexer.TOKEN_END,
		lexer.TOKEN_CASE,
	)
	body.ID = id

	p.endContext()

	expr := p.newExpr(
		KIND_DEFAULT,
		commons.TYPE_UNDEFINED,
		tok,
		body,
	)
	expr.ID = id

	return expr
}

func (p *Parser) parseSwitch() *Expr {
	tok := p.expectKind(lexer.TOKEN_SWITCH)

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_SWITCH,
		Type: commons.TYPE_UNDEFINED,
	})

	value := p.parsePrimary()

	p.endContext()

	expr := p.newExpr(
		KIND_SWITCH,
		commons.TYPE_UNDEFINED,
		tok,
		value,
	)
	expr.ValueSwitch = &SwitchExpr{
		Cases:   make([]int, 0),
		Default: -1,
	}
	id := p.newSwitchID()
	expr.ID = id

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	for {
		switch p.lexer.PeekToken().Kind {
		case lexer.TOKEN_CASE:
			caseIndex := len(expr.Children)

			expr.Children = append(
				expr.Children,
				p.parseSwitchCase(
					id,
					caseIndex,
					value.Type,
					value,
				),
			)

			expr.ValueSwitch.Cases = append(
				expr.ValueSwitch.Cases,
				caseIndex,
			)

		case lexer.TOKEN_DEFAULT:
			defId := len(expr.Children)
			expr.Children = append(expr.Children, p.parseSwitchDefault(id))
			expr.ValueSwitch.Default = defId
		case lexer.TOKEN_END:
			p.expectKind(lexer.TOKEN_END)
			p.expectKind(lexer.TOKEN_ENDLINE)
			return expr
		default:
			commons.CrashOut(
				"expected case, default, or end",
				p.lexer.File_path,
				p.lexer.PeekToken().Line,
				p.lexer.PeekToken().Column,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}
	}
}

func (p *Parser) checkSupportedBodyContext() (*Context, bool) {
	ctx := p.CurrentContext

	for ctx != nil {
		if ctx.BodyKind == CONTEXT_FOR || ctx.BodyKind == CONTEXT_WHILE || ctx.BodyKind == CONTEXT_CASE {
			return ctx, true
		}

		ctx = ctx.Parent
	}

	return nil, false
}

func (p *Parser) parseBreak() *Expr {
	tok := p.expectKind(lexer.TOKEN_BREAK)

	if p.CurrentContext.Kind != CONTEXT_BODY {
		commons.CrashOut(
			"break can't be used in this context",
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	ctx, ok := p.checkSupportedBodyContext()

	if !ok {
		commons.CrashOut(
			"break can't be used in this context",
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_ENDLINE)

	expr := p.newExpr(KIND_BREAK, commons.TYPE_UNDEFINED, tok)
	expr.ValueContext = ctx

	return expr
}

func (p *Parser) parseContinue() *Expr {
	tok := p.expectKind(lexer.TOKEN_CONTINUE)

	if p.CurrentContext.Kind != CONTEXT_BODY {
		commons.CrashOut(
			"continue can't be used in this context",
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	ctx, ok := p.checkSupportedBodyContext()

	if !ok {
		commons.CrashOut(
			"continue can't be used in this context",
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_ENDLINE)

	expr := p.newExpr(KIND_CONTINUE, commons.TYPE_UNDEFINED, tok)
	expr.ValueContext = ctx

	return expr
}

func (p *Parser) parseWhile() *Expr {
	tok := p.expectKind(lexer.TOKEN_WHILE)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	// condition
	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_WHILE,
		Type: commons.TYPE_UNDEFINED,
	})

	condExpr := p.parseRValue()

	if condExpr.IsConstant() {
		condExpr = p.evalConst(condExpr)
	}

	cond := p.implicitCast(
		condExpr,
		commons.TYPE_BOOL,
	)

	cond.checkType(commons.TYPE_BOOL)

	p.endContext()

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	p.expectKind(lexer.TOKEN_DO)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	// do
	id := p.newWhileID()

	p.newContext(p.CurrentContext, &Context{
		Kind:     CONTEXT_BODY,
		BodyKind: CONTEXT_WHILE,
		ID:       id,
	})

	body := p.parseBlock(
		lexer.TOKEN_END,
	)

	p.endContext()

	p.expectKind(lexer.TOKEN_END)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	body.ID = id

	whileExpr := p.newExpr(
		KIND_WHILE,
		commons.TYPE_UNDEFINED,
		tok,
		cond,
		body,
	)
	whileExpr.ID = id

	return whileExpr
}

func (p *Parser) parseFor() *Expr {
	tok := p.expectKind(lexer.TOKEN_FOR)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	parenDepth := 0

	for p.lexer.PeekToken().Kind == lexer.TOKEN_LRPAREN {
		p.expectKind(lexer.TOKEN_LRPAREN)
		parenDepth++
	}

	// begin
	beginCtx := p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_FOR,
		Type: commons.TYPE_UNDEFINED,
	})

	beginExpr := p.newExpr(KIND_BODY, commons.TYPE_UNDEFINED, tok)
	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_SET:
		beginExpr.Children = append(beginExpr.Children, p.parseSet()...)
	case lexer.TOKEN_SEMICOLON:
		// Nothing
	default:
		expr := p.parseRValue()
		if expr.IsConstant() {
			expr = p.evalConst(expr)
		}
		beginExpr.Children = append(beginExpr.Children, expr)
	}

	p.endContext()

	p.expectKind(lexer.TOKEN_SEMICOLON)

	//cond
	condCtx := p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_FOR,
		Type: commons.TYPE_UNDEFINED,
	})

	condCtx.Symbols = beginCtx.Symbols

	condExpr := p.newExpr(KIND_BOOL, commons.TYPE_BOOL, tok)
	condExpr.ValueInt = 1 // true
	if p.lexer.PeekToken().Kind != lexer.TOKEN_SEMICOLON {
		condExpr = p.parseRValue()

		if condExpr.IsConstant() {
			condExpr = p.evalConst(condExpr)
		}

		condExpr = p.implicitCast(
			condExpr,
			commons.TYPE_BOOL,
		)

		condExpr.checkType(commons.TYPE_BOOL)
	}

	p.endContext()

	p.expectKind(lexer.TOKEN_SEMICOLON)

	//update
	updateCtx := p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_FOR,
		Type: commons.TYPE_UNDEFINED,
	})

	updateCtx.Symbols = beginCtx.Symbols

	updateExpr := p.newExpr(KIND_NONE, commons.TYPE_UNDEFINED, tok)
	if p.lexer.PeekToken().Kind != lexer.TOKEN_DO {
		updateExpr = p.parseRValue()

		if updateExpr.IsConstant() {
			updateExpr = p.evalConst(updateExpr)
		}

		updateExpr.checkType(
			commons.TYPE_I32, commons.TYPE_I64, // Full Number
			commons.TYPE_F64, commons.TYPE_F32, // Decimal
			commons.TYPE_I16, commons.TYPE_I8, // Smaller
			commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
			commons.TYPE_STRING, commons.TYPE_BOOL, commons.TYPE_STRUCT, // "Aliases"
		)
	}

	for i := 0; i < parenDepth; i++ {
		p.expectKind(lexer.TOKEN_RRPAREN)
	}

	p.expectKind(lexer.TOKEN_DO)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	// do
	id := p.newForID()
	bodyCtx := p.newContext(p.CurrentContext, &Context{
		Kind:     CONTEXT_BODY,
		BodyKind: CONTEXT_FOR,
		ID:       id,
	})
	bodyCtx.Symbols = beginCtx.Symbols

	body := p.parseBlock(
		lexer.TOKEN_END,
	)

	p.endContext()

	p.expectKind(lexer.TOKEN_END)

	p.optionalExpectKind(lexer.TOKEN_ENDLINE)

	body.ID = id

	whileExpr := p.newExpr(
		KIND_FOR,
		commons.TYPE_UNDEFINED,
		tok,
		beginExpr,
		condExpr,
		updateExpr,
		body,
	)
	whileExpr.ID = id

	return whileExpr
}

func (p *Parser) parseStatement() []*Expr {
	switch p.lexer.PeekToken().Kind {

	case lexer.TOKEN_RETURN:
		if p.CurrentContext.Kind == CONTEXT_ROOT {
			p.HasReturn = true
		}

		return []*Expr{p.parseReturn()}

	case lexer.TOKEN_DUMP:
		return []*Expr{p.parseDump()}

	case lexer.TOKEN_SET:
		expr := p.parseSet()
		p.expectKind(lexer.TOKEN_ENDLINE)
		return expr

	case lexer.TOKEN_TYPEKW:
		return []*Expr{p.parseTypeDecl()}

	case lexer.TOKEN_IF:
		return []*Expr{p.parseIf()}

	case lexer.TOKEN_WHILE:
		return []*Expr{p.parseWhile()}

	case lexer.TOKEN_FOR:
		return []*Expr{p.parseFor()}

	case lexer.TOKEN_QBE:
		expr := p.parseExpression()
		p.expectKind(lexer.TOKEN_ENDLINE)
		return []*Expr{expr}

	case lexer.TOKEN_SWITCH:
		return []*Expr{p.parseSwitch()}

	case lexer.TOKEN_BREAK:
		return []*Expr{p.parseBreak()}

	case lexer.TOKEN_CONTINUE:
		return []*Expr{p.parseContinue()}

	case lexer.TOKEN_ID:
		expr := p.parseExpression()
		p.expectKind(lexer.TOKEN_ENDLINE)
		return []*Expr{expr}

	default:
		tok := p.lexer.NextToken()

		commons.CrashOut(
			fmt.Sprintf("statement: unexpected %q", tok.Kind.String()),
			p.lexer.File_path,
			tok.Line,
			tok.Column,
			commons.CRASH_ERROR,
		)
		os.Exit(1)
	}

	return nil
}

func (p *Parser) Parse() *Expr {
	root := &Expr{
		Kind:   KIND_PROGRAM,
		Parser: p,
	}
	p.Root = root

	for {
		if p.lexer.PeekToken().Kind == lexer.TOKEN_EOF {

			if !p.HasReturn {
				root.Children = append(
					root.Children,
					p.implicitReturn(),
				)
			}

			return root
		}

		root.Children = append(
			root.Children,
			p.parseStatement()...,
		)
	}
}
