package lexer

import (
	"fmt"
	"lin/commons"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type TokenKind int

const (
	TOKEN_EOF TokenKind = iota

	//Lit
	TOKEN_ID
	TOKEN_I64
	TOKEN_I32
	TOKEN_F64
	TOKEN_F32
	TOKEN_STRING

	//Keywords
	TOKEN_RETURN
	TOKEN_DUMP
	TOKEN_SET
	TOKEN_CAST
	TOKEN_IF
	TOKEN_ELSE
	TOKEN_ELIF
	TOKEN_THEN
	TOKEN_END
	TOKEN_WHILE
	TOKEN_FOR
	TOKEN_DO
	TOKEN_QBE
	TOKEN_RAW_QBE

	// Del
	TOKEN_ENDLINE
	TOKEN_LRPAREN
	TOKEN_RRPAREN
	TOKEN_LSPAREN
	TOKEN_RSPAREN
	TOKEN_COMMA
	TOKEN_EQUAL
	TOKEN_LSHIFT

	//BINOP
	TOKEN_PLUS
	TOKEN_MUL
	TOKEN_MINUS
	TOKEN_DIV
	TOKEN_POW
	TOKEN_MOD
	TOKEN_INC
	TOKEN_DEC

	//Assignment BINOP
	TOKEN_PLUS_EQUAL
	TOKEN_MUL_EQUAL
	TOKEN_MINUS_EQUAL
	TOKEN_DIV_EQUAL

	//Types
	TOKEN_TYPE
	TOKEN_TRUE
	TOKEN_FALSE

	//Grammar
	TOKEN_COLON
	TOKEN_DOUBLECOLON
	TOKEN_SEMICOLON

	//Boolean Algebra
	TOKEN_GT
	TOKEN_GE
	TOKEN_LT
	TOKEN_LE
	TOKEN_EQ
	TOKEN_NEQ
	TOKEN_NOT
	TOKEN_OR
	TOKEN_AND
)

func (t TokenKind) String() string {
	switch t {
	case TOKEN_EOF:
		return "EOF"
	case TOKEN_ID:
		return "identifier"
	case TOKEN_I64:
		return "long"
	case TOKEN_F64:
		return "double"
	case TOKEN_I32:
		return "int"
	case TOKEN_F32:
		return "float"
	case TOKEN_STRING:
		return "string"
	case TOKEN_RETURN:
		return "return"
	case TOKEN_DUMP:
		return "dump"
	case TOKEN_ENDLINE:
		return "endline"
	case TOKEN_LRPAREN:
		return "("
	case TOKEN_RRPAREN:
		return ")"
	case TOKEN_LSPAREN:
		return "["
	case TOKEN_RSPAREN:
		return "]"
	case TOKEN_COMMA:
		return ","
	case TOKEN_PLUS:
		return "+"
	case TOKEN_MUL:
		return "*"
	case TOKEN_MINUS:
		return "-"
	case TOKEN_DIV:
		return "/"
	case TOKEN_POW:
		return "^"
	case TOKEN_MOD:
		return "%"
	case TOKEN_EQUAL:
		return "="
	case TOKEN_SET:
		return "set"
	case TOKEN_CAST:
		return "cast"
	case TOKEN_TYPE:
		return "type"
	case TOKEN_PLUS_EQUAL:
		return "+="
	case TOKEN_MUL_EQUAL:
		return "*="
	case TOKEN_MINUS_EQUAL:
		return "-="
	case TOKEN_DIV_EQUAL:
		return "/="
	case TOKEN_INC:
		return "inc (++)"
	case TOKEN_DEC:
		return "dec (--)"
	case TOKEN_DOUBLECOLON:
		return "::"
	case TOKEN_COLON:
		return ":"
	case TOKEN_TRUE:
		return "true"
	case TOKEN_FALSE:
		return "false"
	case TOKEN_IF:
		return "if"
	case TOKEN_THEN:
		return "then"
	case TOKEN_ELSE:
		return "else"
	case TOKEN_ELIF:
		return "elif"
	case TOKEN_END:
		return "end"
	case TOKEN_GT:
		return ">"
	case TOKEN_GE:
		return ">="
	case TOKEN_LT:
		return "<"
	case TOKEN_LE:
		return "<="
	case TOKEN_EQ:
		return "=="
	case TOKEN_NEQ:
		return "!="
	case TOKEN_NOT:
		return "!"
	case TOKEN_OR:
		return "||"
	case TOKEN_AND:
		return "&&"
	case TOKEN_DO:
		return "do"
	case TOKEN_QBE:
		return "qbe"
	case TOKEN_WHILE:
		return "while"
	case TOKEN_FOR:
		return "for"
	case TOKEN_SEMICOLON:
		return ";"
	case TOKEN_LSHIFT:
		return "<<"
	default:
		return "unknown"
	}
}

type Token struct {
	Kind       TokenKind
	Lexeme     string
	Val_string string
	Val_long   int64
	Val_double float64
	Val_int    int32
	Val_float  float32
	Val_type   commons.ExprType
	Line       int
	Column     int
}

type Lexer struct {
	File_path string
	src       []byte
	pos       int
	line      int
	column    int

	peeked *Token

	lastToken     Token
	has_lastToken bool

	parenDepth int
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}

func isValidIDChar(ch byte) bool {
	return ch == '_'
}

func isWhitespace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\r':
		return true
	default:
		return false
	}
}

func isBINOP(ch byte) bool {
	switch ch {
	case '+', '-', '/', '*':
		return true
	default:
		return false
	}
}

func (l *Lexer) continuesLine() bool {
	switch l.peek() {
	case '(':
		return true

	case ';':
		return true

	case '&':
		return l.peekNext(1) == '&'

	case '|':
		return l.peekNext(1) == '|'
	}

	return isBINOP(l.peek())
}

func (l *Lexer) identifier() Token {
	start_col := l.column
	start := l.pos

	for isLetter(l.peek()) || isDigit(l.peek()) || isValidIDChar(l.peek()) {
		l.next()
	}

	lexeme := string(l.src[start:l.pos])

	switch lexeme {
	case "return":
		return l.emit(Token{Kind: TOKEN_RETURN, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "dump":
		return l.emit(Token{Kind: TOKEN_DUMP, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "set":
		return l.emit(Token{Kind: TOKEN_SET, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "cast":
		return l.emit(Token{Kind: TOKEN_CAST, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "i64":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_I64, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "f64":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_F64, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "i32":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_I32, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "f32":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_F32, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "i16":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_I16, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "u16":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_U16, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "i8":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_I8, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "u8":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_U8, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "string":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_STRING, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "bool":
		return l.emit(Token{Kind: TOKEN_TYPE, Val_type: commons.TYPE_BOOL, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "true":
		return l.emit(Token{Kind: TOKEN_TRUE, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "false":
		return l.emit(Token{Kind: TOKEN_FALSE, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "if":
		return l.emit(Token{Kind: TOKEN_IF, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "else":
		return l.emit(Token{Kind: TOKEN_ELSE, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "elif":
		return l.emit(Token{Kind: TOKEN_ELIF, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "then":
		return l.emit(Token{Kind: TOKEN_THEN, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "end":
		return l.emit(Token{Kind: TOKEN_END, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "while":
		return l.emit(Token{Kind: TOKEN_WHILE, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "do":
		return l.emit(Token{Kind: TOKEN_DO, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "for":
		return l.emit(Token{Kind: TOKEN_FOR, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	case "qbe":
		return l.emit(Token{Kind: TOKEN_QBE, Lexeme: lexeme, Column: start_col,
			Line: l.line})
	}

	return l.emit(Token{
		Kind:       TOKEN_ID,
		Lexeme:     lexeme,
		Val_string: lexeme,
		Column:     start_col,
		Line:       l.line,
	})
}

func (l *Lexer) number() Token {
	start_col := l.column
	start := l.pos

	has_decimal := false

	for {
		ch := l.peek()

		if isDigit(ch) {
			l.next()
			continue
		}

		if ch == '.' {
			if has_decimal {
				commons.CrashOut(
					"LEXER:numbers can only have ONE decimal point",
					l.File_path,
					l.line,
					start_col,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}

			has_decimal = true
			l.next()
			continue
		}

		break
	}

	suffix := byte(0)

	switch l.peek() {
	case 'f', 'F', 'l', 'L':
		suffix = l.peek()
		l.next()
	}

	lexeme := string(l.src[start:l.pos])

	number := lexeme
	if suffix != 0 {
		number = lexeme[:len(lexeme)-1]
	}

	switch {
	case (suffix == 'f' || suffix == 'F'):
		value, _ := strconv.ParseFloat(number, 32)
		return l.emit(Token{
			Kind:      TOKEN_F32,
			Lexeme:    lexeme,
			Val_float: float32(value),
			Column:    start_col,
			Line:      l.line,
		})

	case !has_decimal && (suffix == 'l' || suffix == 'L'):
		value, _ := strconv.ParseInt(number, 10, 64)
		return l.emit(Token{
			Kind:     TOKEN_I64,
			Lexeme:   lexeme,
			Val_long: value,
			Column:   start_col,
			Line:     l.line,
		})

	case has_decimal && suffix == 0:
		value, _ := strconv.ParseFloat(number, 64)
		return l.emit(Token{
			Kind:       TOKEN_F64,
			Lexeme:     lexeme,
			Val_double: value,
			Column:     start_col,
			Line:       l.line,
		})

	case !has_decimal && suffix == 0:
		value, _ := strconv.ParseInt(number, 10, 32)
		return l.emit(Token{
			Kind:    TOKEN_I32,
			Lexeme:  lexeme,
			Val_int: int32(value),
			Column:  start_col,
			Line:    l.line,
		})
	default:
		panic("YEEEE")
	}
}

func (l *Lexer) string() Token {
	start_col := l.column
	start := l.pos

	l.next() // consume opening "

	for {
		ch := l.peek()

		if ch == 0 || ch == '\n' || ch == '\r' {
			commons.CrashOut(
				"LEXER:Unmatched quotes on String literal",
				l.File_path,
				l.line,
				start_col,
				commons.CRASH_ERROR,
			)
			os.Exit(1)
		}

		if ch == '"' {
			break
		}

		// escape sequence
		if ch == '\\' {
			l.next() // consume '\'

			if l.peek() == 0 {
				commons.CrashOut(
					"LEXER:Invalid escape sequence",
					l.File_path,
					l.line,
					l.column,
					commons.CRASH_ERROR,
				)
				os.Exit(1)
			}

			l.next() // consume escaped character
			continue
		}

		l.next()
	}

	l.next() // consume closing "

	return l.emit(Token{
		Kind:       TOKEN_STRING,
		Lexeme:     string(l.src[start:l.pos]),
		Val_string: string(l.src[start+1 : l.pos-1]),
		Column:     start_col,
		Line:       l.line,
	})
}

// func (t Token) checkKind(kind TokenKind, l Lexer) {
// 	if t.Kind != kind {
// 		commons.CrashOut(fmt.Sprintf(
// 			"expected %v got %v",
// 			kind.String(),
// 			t.Kind.String(),
// 		), l.File_path, t.Line, t.Column)
// 		os.Exit(1)
// 	}
// }

func New(file_path string, src []byte) *Lexer {
	return &Lexer{
		File_path: file_path,
		src:       src,
		line:      1,
		column:    1,
		pos:       0,
	}
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) peekNext(much int) byte {
	if l.pos+much >= len(l.src) {
		return 0
	}

	return l.src[l.pos+much]
}

func (l *Lexer) next() byte {
	ch := l.peek()
	if ch == 0 {
		return 0
	}

	l.pos++

	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	return ch
}

func (l *Lexer) emit(tok Token) Token {
	l.lastToken = tok
	l.has_lastToken = true
	return tok
}

func (l *Lexer) ReadRawQBE() Token {
	startLine, startColumn := l.line, l.column
	l.skipWhitespace()

	start := l.pos
	for isLetter(l.peek()) || isDigit(l.peek()) || isValidIDChar(l.peek()) {
		l.next()
	}
	delimiter := string(l.src[start:l.pos])
	if delimiter == "" {
		commons.CrashOut("LEXER: expected a QBE heredoc delimiter", l.File_path, l.line, l.column, commons.CRASH_ERROR)
		os.Exit(1)
	}

	for l.peek() != 0 && l.peek() != '\n' {
		l.next()
	}
	if l.peek() == '\n' {
		l.next()
	}

	bodyStart := l.pos
	for {
		lineStart := l.pos
		for l.peek() != 0 && l.peek() != '\n' {
			l.next()
		}
		line := string(l.src[lineStart:l.pos])
		if strings.TrimSpace(line) == delimiter {
			body := string(l.src[bodyStart:lineStart])
			return l.emit(Token{Kind: TOKEN_RAW_QBE, Val_string: body, Line: startLine, Column: startColumn})
		}
		if l.peek() == 0 {
			commons.CrashOut(fmt.Sprintf("LEXER: unterminated QBE block, expected %q", delimiter), l.File_path, startLine, startColumn, commons.CRASH_ERROR)
			os.Exit(1)
		}
		l.next()
	}
}

func (l *Lexer) skipWhitespace() {
	for {
		switch l.peek() {
		case ' ', '\t', '\r':
			l.next()
		default:
			return
		}
	}
}

func (l *Lexer) skipComment() {
	if l.peek() == '#' {
		l.next()
		for {
			switch l.peek() {
			case '\n', 0:
				return
			default:
				l.next()
			}
		}
	}
}

func (l *Lexer) skipBlankLines() (bool, int, int) {
	found := false
	line := 0
	column := 0

	for {
		l.skipWhitespace()

		if l.peek() == '#' {
			l.skipComment()
			continue
		}

		if l.peek() == '\n' {
			if !found {
				line = l.line
				column = l.column
			}

			found = true
			l.next()
			continue
		}

		break
	}

	return found, line, column
}

func (l *Lexer) PeekToken() Token {
	if l.peeked == nil {
		tok := l.NextToken()
		l.peeked = &tok
	}

	return *l.peeked
}

func (l *Lexer) NextToken() Token {
	if l.peeked != nil {
		tok := *l.peeked
		l.peeked = nil
		return tok
	}

	if found, line, column := l.skipBlankLines(); found {
		if l.has_lastToken &&
			l.lastToken.Kind != TOKEN_ENDLINE &&
			l.lastToken.Kind != TOKEN_EOF &&
			l.peek() != 0 &&
			!l.continuesLine() &&
			l.parenDepth == 0 {
			return l.emit(Token{
				Kind:   TOKEN_ENDLINE,
				Line:   line,
				Column: column,
			})
		}
	}

	switch ch := l.peek(); {
	case ch == 0:
		if l.lastToken.Kind != TOKEN_ENDLINE &&
			l.lastToken.Kind != TOKEN_EOF && l.parenDepth == 0 {
			return l.emit(Token{
				Kind:   TOKEN_ENDLINE,
				Lexeme: "\n",
				Line:   l.line,
				Column: l.column,
			})
		}

		return l.emit(Token{
			Kind:   TOKEN_EOF,
			Line:   l.line,
			Column: l.column,
		})
	case ch == '"':
		return l.string()
	case isDigit(ch):
		return l.number()
	case isLetter(ch):
		return l.identifier()
	}

	start_col := l.column
	switch l.next() {
	case ';':
		return l.emit(Token{Kind: TOKEN_SEMICOLON, Lexeme: ";", Line: l.line, Column: start_col})
	case ',':
		return l.emit(Token{Kind: TOKEN_COMMA, Lexeme: ",", Line: l.line, Column: start_col})
	case '=':
		switch l.peek() {
		case '=':
			l.next()
			return l.emit(Token{Kind: TOKEN_EQ, Lexeme: "==", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_EQUAL, Lexeme: "=", Line: l.line, Column: start_col})
	case '(':
		l.parenDepth++
		return l.emit(Token{Kind: TOKEN_LRPAREN, Lexeme: "(", Line: l.line, Column: start_col})
	case ')':
		l.parenDepth--
		return l.emit(Token{Kind: TOKEN_RRPAREN, Lexeme: ")", Line: l.line, Column: start_col})
	case '[':
		return l.emit(Token{Kind: TOKEN_LSPAREN, Lexeme: "[", Line: l.line, Column: start_col})
	case ']':
		return l.emit(Token{Kind: TOKEN_RSPAREN, Lexeme: "]", Line: l.line, Column: start_col})
	case '+':
		switch l.peek() {
		case '=':
			l.next()
			return l.emit(Token{Kind: TOKEN_PLUS_EQUAL, Lexeme: "+=", Line: l.line, Column: start_col})
		case '+':
			l.next()
			return l.emit(Token{Kind: TOKEN_INC, Lexeme: "++", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_PLUS, Lexeme: "+", Line: l.line, Column: start_col})
	case '-':
		switch l.peek() {
		case '=':
			l.next()
			return l.emit(Token{Kind: TOKEN_MINUS_EQUAL, Lexeme: "-=", Line: l.line, Column: start_col})
		case '-':
			l.next()
			return l.emit(Token{Kind: TOKEN_DEC, Lexeme: "--", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_MINUS, Lexeme: "-", Line: l.line, Column: start_col})
	case '*':
		if l.peek() == '=' {
			l.next()
			return l.emit(Token{Kind: TOKEN_MUL_EQUAL, Lexeme: "*=", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_MUL, Lexeme: "*", Line: l.line, Column: start_col})
	case '/':
		if l.peek() == '=' {
			l.next()
			return l.emit(Token{Kind: TOKEN_DIV_EQUAL, Lexeme: "/=", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_DIV, Lexeme: "/", Line: l.line, Column: start_col})
	case '^':
		return l.emit(Token{Kind: TOKEN_POW, Lexeme: "^", Line: l.line, Column: start_col})
	case '%':
		return l.emit(Token{Kind: TOKEN_MOD, Lexeme: "%", Line: l.line, Column: start_col})
	case ':':
		if l.peek() == ':' {
			l.next()
			return l.emit(Token{Kind: TOKEN_DOUBLECOLON, Lexeme: "::", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_COLON, Lexeme: ":", Line: l.line, Column: start_col})
	case '>':
		if l.peek() == '=' {
			l.next()
			return l.emit(Token{Kind: TOKEN_GE, Lexeme: ">=", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_GT, Lexeme: ">", Line: l.line, Column: start_col})
	case '<':
		if l.peek() == '<' {
			l.next()
			return l.emit(Token{Kind: TOKEN_LSHIFT, Lexeme: "<<", Line: l.line, Column: start_col})
		}
		if l.peek() == '=' {
			l.next()
			return l.emit(Token{Kind: TOKEN_LE, Lexeme: "<=", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_LT, Lexeme: ">", Line: l.line, Column: start_col})
	case '!':
		if l.peek() == '=' {
			l.next()
			return l.emit(Token{Kind: TOKEN_NEQ, Lexeme: "!=", Line: l.line, Column: start_col})
		}
		return l.emit(Token{Kind: TOKEN_NOT, Lexeme: "!", Line: l.line, Column: start_col})
	case '|':
		if l.peek() == '|' {
			l.next()
			return l.emit(Token{Kind: TOKEN_OR, Lexeme: "||", Line: l.line, Column: start_col})
		}
	case '&':
		if l.peek() == '&' {
			l.next()
			return l.emit(Token{Kind: TOKEN_AND, Lexeme: "&&", Line: l.line, Column: start_col})
		}
	}

	ch := l.peek()
	commons.CrashOut(
		fmt.Sprintf("LEXER:unexpected character %q", ch),
		l.File_path,
		l.line,
		l.column,
		commons.CRASH_ERROR,
	)
	os.Exit(1)

	return l.emit(Token{})
}
