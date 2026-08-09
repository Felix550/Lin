package parser

import (
	"fmt"
	"lin/commons"
	"lin/lexer"
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
	KIND_VARINIT
	KIND_VARGET
	KIND_VARASSIGN
	KIND_STRCAT
	KIND_NEG
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
	case KIND_VARINIT:
		return "init var"
	case KIND_VARGET:
		return "get var"
	case KIND_VARASSIGN:
		return "assign var"
	case KIND_STRCAT:
		return "string concatenation"
	case KIND_FREEARENA:
		return "free arena"
	default:
		return "unknown"
	}
}

type SymbolKind int

const (
	SYMBOL_VAR SymbolKind = iota
)

type ContextKind int

const (
	CONTEXT_VARSET ContextKind = iota
	CONTEXT_CAST
	CONTEXT_ROOT
	CONTEXT_DUMP
	CONTEXT_RETURN
)

type Expr struct {
	Kind ExprKind
	Type commons.ExprType

	ValueLong   int64
	ValueDouble float64
	ValueInt    int32
	ValueFloat  float32
	ValueString string

	Children []*Expr

	Line   int
	Column int

	Parser *Parser
}

type Symbol struct {
	Name string
	Type commons.ExprType
	Kind SymbolKind
}

type Context struct {
	Parent   *Context
	Children []*Context
	Type     commons.ExprType
	Kind     ContextKind
	// KIND == CONTEXT_VARSET
	VarName string
}

type Parser struct {
	lexer          *lexer.Lexer
	Symbols        map[string]*Symbol
	Strings        map[int]string
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
	}

	return false
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

func (p *Parser) parseString() *Expr {
	tok := p.expectKind(lexer.TOKEN_STRING)

	id := len(p.Strings)

	p.Strings[id] = tok.Val_string

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

	p.newContext(p.CurrentContext, &Context{
		Kind:    CONTEXT_VARSET,
		Type:    symbol.Type,
		VarName: symbol.Name,
	})

	value := p.parseExpression()

	value.checkType(symbol.Type)

	p.expectKind(lexer.TOKEN_ENDLINE)

	p.endContext()
	expr := p.newExpr(KIND_VARASSIGN, symbol.Type, id, value)
	expr.ValueString = symbol.Name
	return expr
}

func (p *Parser) parseConstSet(id lexer.Token) *Expr {
	return &Expr{} // TODO: Implement consts
}

func (p *Parser) parseId() *Expr {
	id := p.expectKind(lexer.TOKEN_ID)

	switch p.lexer.PeekToken().Kind {
	case lexer.TOKEN_EQUAL:
		return p.parseAssign(id)
	case lexer.TOKEN_COLON:
		return p.parseConstSet(id)
	default:
		symbol, exists := p.Symbols[id.Val_string]

		if !exists {
			commons.CrashOut(fmt.Sprintf("no symbol exists with name %q", id.Val_string), p.lexer.File_path, id.Line, id.Column)
			os.Exit(1)
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

	case lexer.TOKEN_LPAREN:
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

func (p *Parser) parsePow() *Expr {
	left := p.parsePrimary()

	for {
		if p.lexer.PeekToken().Kind != lexer.TOKEN_POW {
			return left
		}

		p.expectKind(lexer.TOKEN_POW)

		right := p.parseUnary()

		finalType := commons.TYPE_F64

		left = p.implicitCast(left, finalType)
		right = p.implicitCast(right, finalType)

		left = p.newExpr(
			KIND_POW,
			finalType,
			left,
			left,
			right,
		)
	}
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
	p.expectKind(lexer.TOKEN_LPAREN)

	expr := p.parseExpression()

	p.expectKind(lexer.TOKEN_RPAREN)

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

		names = append(names, name)

		if p.lexer.PeekToken().Kind != lexer.TOKEN_COMMA {
			break
		}

		p.expectKind(lexer.TOKEN_COMMA)
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

		p.Symbols[name.Val_string] = &Symbol{
			Name: name.Val_string,
			Type: value.Type,
			Kind: SYMBOL_VAR,
		}

		expr := p.newExpr(
			KIND_VARINIT,
			value.Type,
			value,
			value,
		)
		expr.ValueString = name.Val_string

		result = append(result, expr)
	}

	return result
}

func (p *Parser) parseCast() *Expr {
	Type := p.expectKind(lexer.TOKEN_TYPE)
	p.expectKind(lexer.TOKEN_LPAREN)

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

	p.expectKind(lexer.TOKEN_RPAREN)

	return p.newExpr(
		KIND_CAST,
		Type.Val_type,
		Value,
		Value,
	)
}

func (p *Parser) Parse() *Expr {
	root := &Expr{
		Kind:   KIND_PROGRAM,
		Parser: p,
	}
	p.Root = root

	for {
		tok := p.lexer.PeekToken()

		switch tok.Kind {

		case lexer.TOKEN_RETURN:
			p.HasReturn = true
			root.Children = append(
				root.Children,
				p.parseReturn(),
			)

		case lexer.TOKEN_DUMP:
			root.Children = append(
				root.Children,
				p.parseDump(),
			)

		case lexer.TOKEN_SET:
			root.Children = append(
				root.Children,
				p.parseSet()...,
			)

		case lexer.TOKEN_ID:
			root.Children = append(
				root.Children,
				p.parseId(),
			)

		case lexer.TOKEN_EOF:
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

		default:
			commons.CrashOut(fmt.Sprintf("top: unexpected %q", tok.Kind.String()), p.lexer.File_path, tok.Line, tok.Column)
			os.Exit(1)
		}
	}
}
