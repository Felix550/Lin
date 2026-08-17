package parser

import (
	"fmt"
	"lin/commons"
	"lin/lexer"
	"math"
	"os"
)

type ExprKind int

const (
	KIND_PROGRAM ExprKind = iota

	KIND_RETURN
	KIND_FREEARENA
	KIND_DUMP
	KIND_LONG
	KIND_INT
	KIND_DOUBLE
	KIND_FLOAT
	KIND_STRING
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
	KIND_STRCAT
	KIND_NEG
	KIND_POSTINC
	KIND_POSTDEC
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
	case KIND_STRCAT:
		return "string concatenation"
	case KIND_FREEARENA:
		return "free arena"
	case KIND_POSTINC:
		return "postinc"
	case KIND_POSTDEC:
		return "postdec"
	default:
		return "unknown"
	}
}

type SymbolKind int

const (
	SYMBOL_VAR SymbolKind = iota
	SYMBOL_CONST
)

type ContextKind int

const (
	CONTEXT_VARSET ContextKind = iota
	CONTEXT_BINOPASSIGN
	CONTEXT_CONSTSET
	CONTEXT_CAST
	CONTEXT_ROOT
	CONTEXT_DUMP
	CONTEXT_RETURN
	CONTEXT_SLICE
)

type Expr struct {
	Kind ExprKind
	Type commons.ExprType

	ValueLong   int64
	ValueDouble float64
	ValueInt    int32
	ValueFloat  float32
	ValueString string
	ValueKind   ExprKind
	ValueType   commons.ExprType

	Children []*Expr

	Line   int
	Column int

	Parser *Parser
}

type Symbol struct {
	Name string
	Type commons.ExprType
	Kind SymbolKind

	Value *Expr

	//Array TYPE
	Element commons.ExprType
	Size    int
}

type Context struct {
	Parent   *Context
	Children []*Context
	Type     commons.ExprType
	Kind     ContextKind
	// KIND == CONTEXT_VARSET
	VarName  string
	VarIndex int
}

type Parser struct {
	lexer          *lexer.Lexer
	Symbols        map[string]*Symbol
	Strings        map[int]string
	StringsLookup  map[string]int
	HasDump        bool
	HasReturn      bool
	HasFreeArena   bool
	StmtContexts   []*Context
	CurrentContext *Context
	Root           *Expr
}

func New(l *lexer.Lexer) *Parser {
	rootContext := &Context{
		Kind: CONTEXT_ROOT,
		Type: commons.TYPE_UNDEFINED,
	}

	return &Parser{
		lexer:          l,
		HasDump:        false,
		HasReturn:      false,
		HasFreeArena:   false,
		Symbols:        make(map[string]*Symbol),
		Strings:        make(map[int]string),
		StringsLookup:  make(map[string]int),
		StmtContexts:   []*Context{rootContext},
		CurrentContext: rootContext,
	}
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
		commons.CrashOut(fmt.Sprintf("invalid rootExpr type: %q", rootExpr), p.lexer.File_path, line, line)
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

func (p *Parser) newContext(
	parent *Context,
	context *Context,
) *Context {
	context.Parent = parent
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

func (e *Expr) checkKind(kind ExprKind) {
	if e.Kind != kind {
		commons.CrashOut(fmt.Sprintf(
			"expected %q got %q",
			kind.String(),
			e.Kind.String(),
		), e.Parser.lexer.File_path, e.Line, e.Column)
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
	), e.Parser.lexer.File_path, e.Line, e.Column)

	os.Exit(1)
}

func (e *Expr) IsConstant() bool {
	switch e.Kind {
	case KIND_INT,
		KIND_LONG,
		KIND_FLOAT,
		KIND_DOUBLE,
		KIND_STRING:

		return true

	case KIND_ADD,
		KIND_SUB,
		KIND_MUL,
		KIND_DIV,
		KIND_MOD,
		KIND_POW,
		KIND_CAST,
		KIND_STRCAT:

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
		), p.lexer.File_path, tok.Line, tok.Column)
		os.Exit(1)
	}

	return tok
}

func (p *Parser) expectType(expr *Expr, t commons.ExprType) {
	if expr.Type != t {
		commons.CrashOut(
			fmt.Sprintf("expected %q got %q", t, expr.Type),
			p.lexer.File_path,
			expr.Line,
			expr.Column,
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
	symbol, exists := p.Symbols[id.Val_string]

	if !exists {
		commons.CrashOut(fmt.Sprintf("variable %q does not exists", id.Val_string), p.lexer.File_path, id.Line, id.Column)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_EQUAL)

	var expr *Expr

	if symbol.Type == commons.TYPE_ARRAY {
		expr = p.parseSetArrayValues(id, symbol.Size, false, symbol.Element)
	} else {
		p.newContext(p.CurrentContext, &Context{
			Kind:    CONTEXT_VARSET,
			Type:    symbol.Type,
			VarName: symbol.Name,
		})

		value := p.parseExpression()

		value = p.implicitCast(value, symbol.Type)
		value.checkType(symbol.Type)

		p.endContext()
		expr = p.newExpr(KIND_VARASSIGN, symbol.Type, id, value)
		expr.ValueString = symbol.Name
	}
	return expr
}

func (p *Parser) parseAssignmentBINOP(id lexer.Token) *Expr {
	symbol, exists := p.Symbols[id.Val_string]

	if !exists || symbol.Kind != SYMBOL_VAR {
		commons.CrashOut(
			fmt.Sprintf("symbol %q is not a variable", id.Val_string),
			p.lexer.File_path,
			id.Line,
			id.Column,
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
		)
		os.Exit(1)
	}

	p.newContext(p.CurrentContext, &Context{
		Kind:    CONTEXT_BINOPASSIGN,
		Type:    symbol.Type,
		VarName: symbol.Name,
	})

	value := p.implicitCast(p.parseExpression(), symbol.Type)
	value.checkType(symbol.Type)

	p.endContext()

	left := p.newExpr(
		KIND_VARGET,
		symbol.Type,
		id,
	)
	left.ValueString = symbol.Name

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

	expr_assign := p.newExpr(
		KIND_VARASSIGN,
		symbol.Type,
		id,
		right,
	)
	expr_assign.ValueString = symbol.Name

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
		)
		os.Exit(1)
	}
}

func (p *Parser) parsePostINCDEC(id lexer.Token) *Expr {
	symbol, exists := p.Symbols[id.Val_string]
	if !exists || symbol.Kind != SYMBOL_VAR {
		commons.CrashOut(
			fmt.Sprintf("symbol %q is not a variable", id.Val_string),
			p.lexer.File_path,
			id.Line,
			id.Column,
		)
		os.Exit(1)
	}

	p.checkIncDecType(symbol.Type, id.Line, id.Column)

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
			id.Line,
			id.Column,
		)
		os.Exit(1)
	}

	value := p.getValueType(1, symbol.Type, id)

	left := p.newExpr(KIND_VARGET, symbol.Type, id)
	left.ValueString = symbol.Name

	binop := p.newExpr(incdecKind, symbol.Type, id, left, value)

	exprAssign := p.newExpr(KIND_VARASSIGN, symbol.Type, id, binop)
	exprAssign.ValueString = symbol.Name

	return p.newExpr(postKind, symbol.Type, id, left, exprAssign)
}

func (p *Parser) parseSlice(id lexer.Token) *Expr {
	symbol, exists := p.Symbols[id.Val_string]

	if !exists {
		commons.CrashOut(fmt.Sprintf("no symbol exists with name %q", id.Val_string), p.lexer.File_path, id.Line, id.Column)
		os.Exit(1)
	}

	if symbol.Type != commons.TYPE_ARRAY {
		commons.CrashOut(fmt.Sprintf("can't slice type %q", symbol.Type.String()), p.lexer.File_path, id.Line, id.Column)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_LSPAREN)

	indexExpr := p.parseConstSlice()

	index := indexExpr.ValueInt
	if int(index) >= symbol.Size {
		commons.CrashOut(
			fmt.Sprintf("index %d out of bounds for array %q of size %d", index, symbol.Name, symbol.Size),
			p.lexer.File_path,
			indexExpr.Line,
			indexExpr.Column,
		)
		os.Exit(1)
	}

	p.expectKind(lexer.TOKEN_RSPAREN)

	if p.lexer.PeekToken().Kind == lexer.TOKEN_EQUAL {
		if p.CurrentContext.Kind != CONTEXT_ROOT {
			commons.CrashOut(
				"assignment is not allowed in this context",
				p.lexer.File_path,
				id.Line,
				id.Column,
			)
			os.Exit(1)
		}
		p.expectKind(lexer.TOKEN_EQUAL)
		p.newContext(p.CurrentContext, &Context{
			Kind:     CONTEXT_VARSET,
			Type:     symbol.Element,
			VarName:  symbol.Name,
			VarIndex: int(index),
		})

		value := p.parseExpression()

		value = p.implicitCast(value, symbol.Element)
		value.checkType(symbol.Element)

		p.endContext()
		expr := p.newExpr(KIND_VARSLICEASSIGN, symbol.Element, id, value)
		expr.ValueString = symbol.Name
		expr.ValueInt = index
		return expr
	}

	expr := p.newExpr(KIND_VARSLICEGET, symbol.Element, id)
	expr.ValueString = symbol.Name
	expr.ValueInt = index
	return expr
}

func (p *Parser) parseId() *Expr {
	id := p.expectKind(lexer.TOKEN_ID)

	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_EQUAL:
		if p.CurrentContext.Kind != CONTEXT_ROOT {
			commons.CrashOut(
				"assignment is not allowed in this context",
				p.lexer.File_path,
				id.Line,
				id.Column,
			)
			os.Exit(1)
		}
		return p.parseAssign(id)
	case lexer.TOKEN_DOUBLECOLON:
		if p.CurrentContext.Kind != CONTEXT_ROOT {
			commons.CrashOut(
				"constant declaration is not allowed in this context",
				p.lexer.File_path,
				id.Line,
				id.Column,
			)
			os.Exit(1)
		}
		return p.parseConstSet(id)
	case lexer.TOKEN_PLUS_EQUAL, lexer.TOKEN_MINUS_EQUAL, lexer.TOKEN_DIV_EQUAL, lexer.TOKEN_MUL_EQUAL:
		if p.CurrentContext.Kind != CONTEXT_ROOT {
			commons.CrashOut(
				"assignment is not allowed in this context",
				p.lexer.File_path,
				id.Line,
				id.Column,
			)
			os.Exit(1)
		}
		return p.parseAssignmentBINOP(id)
	case lexer.TOKEN_INC, lexer.TOKEN_DEC:
		return p.parsePostINCDEC(id)
	case lexer.TOKEN_LSPAREN:
		return p.parseSlice(id)
	default:
		symbol, exists := p.Symbols[id.Val_string]

		if !exists {
			commons.CrashOut(fmt.Sprintf("no symbol exists with name %q", id.Val_string), p.lexer.File_path, id.Line, id.Column)
			os.Exit(1)
		}

		if symbol.Kind == SYMBOL_CONST {
			return symbol.Value
		}

		expr := p.newExpr(KIND_VARGET, symbol.Type, id)
		expr.ValueString = symbol.Name
		return expr
	}
}

func (p *Parser) parsePrimary() *Expr {
	switch p.lexer.PeekToken().Kind {

	case lexer.TOKEN_TYPE:
		return p.parseCast()

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

	case lexer.TOKEN_ID:
		return p.parseId()

	default:
		tok := p.lexer.NextToken()
		commons.CrashOut(
			fmt.Sprintf("primary: unexpected %q", tok.Kind.String()),
			p.lexer.File_path,
			tok.Line,
			tok.Column,
		)
		os.Exit(1)
	}

	panic("UNREACHABLE")
}

func (p *Parser) parseUnary() *Expr {
	if p.lexer.PeekToken().Kind == lexer.TOKEN_MINUS {
		tok := p.expectKind(lexer.TOKEN_MINUS)

		value := p.parseUnary()

		return p.newExpr(
			KIND_NEG,
			value.Type,
			tok,
			value,
		)
	}

	return p.parsePow()
}

func (p *Parser) parseExpression() *Expr {
	return p.parseAddSub() //GENERAL BINOP or String
}

func (p *Parser) getFinalBinopType(left *Expr, right *Expr) commons.ExprType {
	if left.Type == commons.TYPE_STRING || right.Type == commons.TYPE_STRING {
		return commons.TYPE_STRING
	}

	if p.CurrentContext.Type != commons.TYPE_UNDEFINED {
		return p.CurrentContext.Type
	}

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
		commons.TYPE_I16, commons.TYPE_I8, // Smaller Signed
		commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
		commons.TYPE_STRING, // String
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

	commons.CrashOut(
		fmt.Sprintf(
			"cannot implicitly convert %q to %q",
			expr.Type.String(),
			finalType.String(),
		),
		p.lexer.File_path,
		expr.Line,
		expr.Column,
	)

	os.Exit(1)
	return &Expr{}
}

func (p *Parser) parsePostfix() *Expr {
	expr := p.parsePrimary()

	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_INC:
		tok := p.expectKind(lexer.TOKEN_INC)

		if expr.Kind != KIND_VARGET {
			commons.CrashOut(
				"operand of ++ must be a variable",
				p.lexer.File_path,
				tok.Line,
				tok.Column,
			)
			os.Exit(1)
		}

		p.checkIncDecType(expr.Type, tok.Line, tok.Column)

		one := p.getValueType(1, expr.Type, tok)

		increment := p.newExpr(
			KIND_ADD,
			expr.Type,
			tok,
			expr,
			one,
		)

		assign := p.newExpr(
			KIND_VARASSIGN,
			expr.Type,
			tok,
			increment,
		)
		assign.ValueString = expr.ValueString

		return p.newExpr(
			KIND_POSTINC,
			expr.Type,
			tok,
			expr,
			assign,
		)

	case lexer.TOKEN_DEC:
		tok := p.expectKind(lexer.TOKEN_DEC)

		if expr.Kind != KIND_VARGET {
			commons.CrashOut(
				"operand of -- must be a variable",
				p.lexer.File_path,
				tok.Line,
				tok.Column,
			)
			os.Exit(1)
		}

		p.checkIncDecType(expr.Type, tok.Line, tok.Column)

		one := p.getValueType(1, expr.Type, tok)

		decrement := p.newExpr(
			KIND_SUB,
			expr.Type,
			tok,
			expr,
			one,
		)

		assign := p.newExpr(
			KIND_VARASSIGN,
			expr.Type,
			tok,
			decrement,
		)
		assign.ValueString = expr.ValueString

		return p.newExpr(
			KIND_POSTDEC,
			expr.Type,
			tok,
			expr,
			assign,
		)
	}

	return expr
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

	expr := p.parseExpression()

	p.expectKind(lexer.TOKEN_RRPAREN)

	return expr
}

func (p *Parser) parseReturn() *Expr {
	if !p.HasFreeArena && p.CurrentContext.Kind == CONTEXT_ROOT {
		p.Root.Children = append(
			p.Root.Children,
			p.freeArena(),
		)
	}

	p.expectKind(lexer.TOKEN_RETURN)

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_RETURN,
		Type: commons.TYPE_UNDEFINED,
	})

	value := p.parseExpression()

	value.checkType(commons.TYPE_I64, commons.TYPE_I32, commons.TYPE_I16, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_U8)

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

	value := p.parseExpression()

	if value.IsConstant() {
		value = p.evalConst(value)
	}

	value.checkType(
		commons.TYPE_I32, commons.TYPE_I64, // Full Number
		commons.TYPE_F64, commons.TYPE_F32, // Decimal
		commons.TYPE_I16, commons.TYPE_I8, // Smaller
		commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
		commons.TYPE_STRING, // String
	)

	p.expectKind(lexer.TOKEN_ENDLINE)

	p.HasDump = true

	p.endContext()
	return p.newExpr(KIND_DUMP, commons.TYPE_U0, value, value)
}

func (p *Parser) newZeroValue(
	typ commons.ExprType,
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

	default:
		commons.CrashOut(
			fmt.Sprintf(
				"cannot zero-initialize type %q",
				typ.String(),
			),
			p.lexer.File_path,
			root.Line,
			root.Column,
		)
		os.Exit(1)
	}

	return expr
}

func (p *Parser) makeArray(name lexer.Token, size int, Type commons.ExprType, values ...*Expr) *Expr {
	p.Symbols[name.Val_string] = &Symbol{
		Name:    name.Val_string,
		Type:    commons.TYPE_ARRAY,
		Element: Type,
		Size:    size,
		Kind:    SYMBOL_VAR,
	}

	expr := p.newExpr(
		KIND_VARINIT,
		commons.TYPE_ARRAY,
		name,
		values...,
	)

	expr.ValueType = Type
	expr.ValueInt = int32(size)
	expr.ValueString = name.Val_string
	return expr
}

func (p *Parser) parseSetArrayValues(name lexer.Token, size int, inferSize bool, finalType commons.ExprType) *Expr {

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
			)
			os.Exit(1)
		}

		p.newContext(p.CurrentContext, &Context{Kind: CONTEXT_VARSET, Type: finalType, VarName: name.Val_string, VarIndex: len(values)})

		value := p.parseExpression()

		value.checkType(
			commons.TYPE_I32,
			commons.TYPE_I64,
			commons.TYPE_F64,
			commons.TYPE_F32,
			commons.TYPE_I16,
			commons.TYPE_I8,
			commons.TYPE_U16,
			commons.TYPE_U8,
			commons.TYPE_STRING,
		)

		p.endContext()

		// First element determines element type
		if finalType == commons.TYPE_UNDEFINED {
			finalType = value.Type
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
		)
		os.Exit(1)
	}

	// Fill remaining elements
	for len(values) < size {
		values = append(values, p.newZeroValue(finalType, name))
	}

	expr := p.makeArray(name, size, finalType, values...)

	return expr
}

func (p *Parser) parseConstSlice() *Expr {
	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_SLICE,
		Type: commons.TYPE_I32,
	})

	idxExpr := p.parseExpression()

	if !idxExpr.IsConstant() {
		commons.CrashOut(
			"array size must be compile-time evaluable",
			p.lexer.File_path,
			idxExpr.Line,
			idxExpr.Column,
		)
		os.Exit(1)
	}

	idxExpr = p.evalConst(idxExpr)
	idxExpr.checkType(commons.TYPE_I32)

	p.endContext()

	return idxExpr
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
			)
			os.Exit(1)
		}
	}

	p.expectKind(lexer.TOKEN_RSPAREN)
	p.expectKind(lexer.TOKEN_EQUAL)

	expr := p.parseSetArrayValues(name, size, inferSize, commons.TYPE_UNDEFINED)
	return []*Expr{expr}
}

func (p *Parser) parseType() (Type commons.ExprType, isArray bool, arraySize int) {
	isArray = false
	arraySize = 0
	Type = p.expectKind(lexer.TOKEN_TYPE).Val_type
	if p.lexer.PeekToken().Kind == lexer.TOKEN_LSPAREN {
		p.expectKind(lexer.TOKEN_LSPAREN)
		sizeExpr := p.parseConstSlice()
		p.expectKind(lexer.TOKEN_RSPAREN)
		isArray = true
		arraySize = int(sizeExpr.ValueInt)
	}
	return
}

func (p *Parser) parseSet() []*Expr {
	p.expectKind(lexer.TOKEN_SET)

	var names []lexer.Token
	for {
		name := p.expectKind(lexer.TOKEN_ID)

		_, exists := p.Symbols[name.Val_string]
		if exists {
			commons.CrashOut(fmt.Sprintf("symbol with name %q already exists", name.Val_string), p.lexer.File_path, name.Line, name.Column)
			os.Exit(1)
		}

		if p.lexer.PeekToken().Kind == lexer.TOKEN_LSPAREN {
			if len(names) == 0 {
				return p.parseSetArray(name)
			} else {
				commons.CrashOut("array declarations cannot be combined with multiple variables", p.lexer.File_path, name.Line, name.Column)
				os.Exit(1)
			}
		}

		names = append(names, name)

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
	}

	if p.lexer.PeekToken().Kind == lexer.TOKEN_TYPE {
		result := make([]*Expr, 0, len(names))

		Type, isArray, Size := p.parseType()
		for _, name := range names {
			var expr *Expr
			if isArray {
				values := make([]*Expr, Size)

				for i := range values {
					values[i] = p.newZeroValue(Type, name)
				}

				expr = p.makeArray(name, Size, Type, values...)
			} else {
				p.Symbols[name.Val_string] = &Symbol{
					Name: name.Val_string,
					Type: Type,
					Kind: SYMBOL_VAR,
				}

				expr = p.newExpr(
					KIND_VARINIT,
					Type,
					name,
					p.newZeroValue(Type, name),
				)
				expr.ValueString = name.Val_string
			}

			result = append(result, expr)
		}

		p.expectKind(lexer.TOKEN_ENDLINE)

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
			)
			os.Exit(1)
		}

		if p.lexer.PeekToken().Kind == lexer.TOKEN_LSPAREN {
			values = append(values, p.parseSetArrayValues(names[0], 0, true,commons.TYPE_UNDEFINED))
		} else {
			p.newContext(p.CurrentContext, &Context{
				Kind:    CONTEXT_VARSET,
				Type:    commons.TYPE_UNDEFINED,
				VarName: names[i].Val_string,
			})

			value := p.parseExpression()

			value.checkType(
				commons.TYPE_I32, commons.TYPE_I64, // Full Number
				commons.TYPE_F64, commons.TYPE_F32, // Decimal
				commons.TYPE_I16, commons.TYPE_I8, // Smaller Signed
				commons.TYPE_U16, commons.TYPE_U8, // Smaller Unsigned
				commons.TYPE_STRING, // String
			)

			p.endContext()

			values = append(values, value)
		}

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
	}

	p.expectKind(lexer.TOKEN_ENDLINE)

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
		)
		os.Exit(1)
	}

	result := make([]*Expr, 0, len(names))

	for i, name := range names {
		value := values[i]

		if value.Type != commons.TYPE_ARRAY {
			p.Symbols[name.Val_string] = &Symbol{
				Name: name.Val_string,
				Type: value.Type,
				Kind: SYMBOL_VAR,
			}

			value = p.newExpr(
				KIND_VARINIT,
				value.Type,
				value,
				value,
			)
			value.ValueString = name.Val_string
		}

		result = append(result, value)
	}

	return result
}

// #region const-parsing
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

	case KIND_CAST:
		value := p.evalConst(expr.Children[0])

		switch expr.Type {
		case commons.TYPE_I32:
			switch value.Type {
			case commons.TYPE_I64:
				return p.newConstInt(int32(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstInt(int32(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstInt(int32(value.ValueDouble), expr)
			}

		case commons.TYPE_I64:
			switch value.Type {
			case commons.TYPE_I32:
				return p.newConstLong(int64(value.ValueInt), expr)
			case commons.TYPE_F32:
				return p.newConstLong(int64(value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstLong(int64(value.ValueDouble), expr)
			}

		case commons.TYPE_F32:
			switch value.Type {
			case commons.TYPE_I32:
				return p.newConstFloat(float32(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstFloat(float32(value.ValueLong), expr)
			case commons.TYPE_F64:
				return p.newConstFloat(float32(value.ValueDouble), expr)
			}

		case commons.TYPE_F64:
			switch value.Type {
			case commons.TYPE_I32:
				return p.newConstDouble(float64(value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstDouble(float64(value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstDouble(float64(value.ValueFloat), expr)
			}

		case commons.TYPE_STRING:
			switch value.Type {
			case commons.TYPE_I32:
				return p.newConstString(fmt.Sprintf("%d", value.ValueInt), expr)
			case commons.TYPE_I64:
				return p.newConstString(fmt.Sprintf("%d", value.ValueLong), expr)
			case commons.TYPE_F32:
				return p.newConstString(fmt.Sprintf("%f", value.ValueFloat), expr)
			case commons.TYPE_F64:
				return p.newConstString(fmt.Sprintf("%f", value.ValueDouble), expr)
			}

		default:
			commons.CrashOut(fmt.Sprintf(
				"compile-time cast not supported from type %q to type %q",
				value.Type.String(),
				expr.Type.String(),
			), p.lexer.File_path, expr.Line, expr.Column)
			os.Exit(1)
		}
	}

	commons.CrashOut(fmt.Sprintf(
		"cannot evaluate compile-time expression %q",
		expr.Kind.String(),
	), p.lexer.File_path, expr.Line, expr.Column)
	os.Exit(1)

	panic("UNREACHABLE")
}

func (p *Parser) parseConstSet(name lexer.Token) *Expr {
	p.expectKind(lexer.TOKEN_DOUBLECOLON)

	if _, exists := p.Symbols[name.Val_string]; exists {
		commons.CrashOut(
			fmt.Sprintf(
				"constant symbol %q already exists",
				name.Val_string,
			),
			p.lexer.File_path,
			name.Line,
			name.Column,
		)
		os.Exit(1)
	}

	p.newContext(p.CurrentContext, &Context{
		Kind:    CONTEXT_CONSTSET,
		Type:    commons.TYPE_UNDEFINED,
		VarName: name.Val_string,
	})

	value := p.parseExpression()

	if !value.IsConstant() {
		commons.CrashOut(
			"constant expression is not compile-time evaluable",
			p.lexer.File_path,
			value.Line,
			value.Column,
		)
		os.Exit(1)
	}

	value = p.evalConst(value)

	p.endContext()

	value.checkType(
		commons.TYPE_I32,
		commons.TYPE_I64,
		commons.TYPE_F32,
		commons.TYPE_F64,
		commons.TYPE_I16,
		commons.TYPE_I8,
		commons.TYPE_U16,
		commons.TYPE_U8,
		commons.TYPE_STRING,
	)

	p.Symbols[name.Val_string] = &Symbol{
		Name:  name.Val_string,
		Type:  value.Type,
		Kind:  SYMBOL_CONST,
		Value: value,
	}

	expr := p.newExpr(
		KIND_CONSTSET,
		value.Type,
		value,
		value,
	)

	expr.ValueString = name.Val_string

	return expr
}

//#endregion

func (p *Parser) parseCast() *Expr {
	Type := p.expectKind(lexer.TOKEN_TYPE)
	p.expectKind(lexer.TOKEN_LRPAREN)

	var ctxType commons.ExprType
	if Type.Val_type != commons.TYPE_STRING {
		ctxType = Type.Val_type
	} else {
		ctxType = commons.TYPE_UNDEFINED
	}

	p.newContext(p.CurrentContext, &Context{
		Kind: CONTEXT_CAST,
		Type: ctxType,
	})

	Value := p.parseExpression()

	p.endContext()

	p.expectKind(lexer.TOKEN_RRPAREN)

	return p.newExpr(
		KIND_CAST,
		Type.Val_type,
		Value,
		Value,
	)
}

func (p *Parser) parseStatement() []*Expr {
	switch p.lexer.PeekToken().Kind {

	case lexer.TOKEN_RETURN:
		p.HasReturn = true
		return []*Expr{p.parseReturn()}

	case lexer.TOKEN_DUMP:
		return []*Expr{p.parseDump()}

	case lexer.TOKEN_SET:
		return p.parseSet()

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
			if !p.HasFreeArena {
				root.Children = append(
					root.Children,
					p.freeArena(),
				)
			}

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
