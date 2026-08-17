package codegen

import (
	"fmt"
	"lin/commons"
	"lin/parser"
)

type Generator struct {
	NumTemp int
	Code    string
}

type CastOp struct {
	Op   string
	Type commons.ExprType
}

type Value struct {
	String string
	Type   commons.ExprType
}

func NewValue(_string string, t commons.ExprType) Value {
	return Value{
		String: _string,
		Type:   t,
	}
}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) newTemp(prefix string) string {
	name := fmt.Sprintf("%%%s_t%d", prefix, g.NumTemp)
	g.NumTemp++
	return name
}

func (g *Generator) getBinopExprType(left, right Value) commons.ExprType {
	switch {
	case left.Type == commons.TYPE_F64 || right.Type == commons.TYPE_F64:
		return commons.TYPE_F64

	case left.Type == commons.TYPE_F32 || right.Type == commons.TYPE_F32:
		return commons.TYPE_F32

	case left.Type == commons.TYPE_I64 || right.Type == commons.TYPE_I64:
		return commons.TYPE_I64

	default:
		return commons.TYPE_I32
	}
}

func (g *Generator) getNumberExprType(Type commons.ExprType) string {
	switch Type {
	case commons.TYPE_F64:
		return "d"
	case commons.TYPE_I64, commons.TYPE_STRING:
		return "l"
	case commons.TYPE_I32, commons.TYPE_I16, commons.TYPE_U16, commons.TYPE_I8, commons.TYPE_U8:
		return "w"
	case commons.TYPE_F32:
		return "s"
	default:
		panic(fmt.Sprintf(
			"unsupported type %q",
			Type.String(),
		))
	}
}

func (g *Generator) getNumberExprLoadType(Type commons.ExprType) string {
	switch Type {
	case commons.TYPE_F64:
		return "d"
	case commons.TYPE_I64, commons.TYPE_STRING:
		return "l"
	case commons.TYPE_I32:
		return "w"
	case commons.TYPE_F32:
		return "s"
	case commons.TYPE_I16:
		return "sh"
	case commons.TYPE_U16:
		return "uh"
	case commons.TYPE_I8:
		return "sb"
	case commons.TYPE_U8:
		return "ub"
	default:
		panic(fmt.Sprintf(
			"unsupported load type %q",
			Type.String(),
		))
	}
}

func (g *Generator) getNumberExprStoreType(Type commons.ExprType) string {
	switch Type {
	case commons.TYPE_F64:
		return "d"
	case commons.TYPE_I64, commons.TYPE_STRING:
		return "l"
	case commons.TYPE_I32:
		return "w"
	case commons.TYPE_F32:
		return "s"
	case commons.TYPE_I16, commons.TYPE_U16:
		return "h"
	case commons.TYPE_I8, commons.TYPE_U8:
		return "b"
	default:
		panic(fmt.Sprintf(
			"unsupported store type %q",
			Type.String(),
		))
	}
}

func (g *Generator) getNumberExprSize(Type commons.ExprType) int {
	switch Type {
	case commons.TYPE_I8, commons.TYPE_U8:
		return 1
	case commons.TYPE_I16, commons.TYPE_U16:
		return 2
	case commons.TYPE_F64:
		return 8
	case commons.TYPE_I64, commons.TYPE_STRING:
		return 8
	case commons.TYPE_I32:
		return 4
	case commons.TYPE_F32:
		return 4
	default:
		panic(fmt.Sprintf(
			"unsupported type %q",
			Type.String(),
		))
	}
}

func normalizeType(t commons.ExprType) commons.ExprType {
	switch t {
	case commons.TYPE_I8, commons.TYPE_U8, commons.TYPE_I16, commons.TYPE_U16:
		return commons.TYPE_I32
	default:
		return t
	}
}

func (g *Generator) getCastExpr(from, to commons.ExprType) []CastOp {
	from = normalizeType(from)
	to = normalizeType(to)

	if from == to {
		return []CastOp{{"copy", to}}
	}

	switch from {
	case commons.TYPE_I32:
		switch to {
		case commons.TYPE_I64:
			return []CastOp{{"extsw", commons.TYPE_I64}}
		case commons.TYPE_F32:
			return []CastOp{{"swtof", commons.TYPE_F32}}
		case commons.TYPE_F64:
			return []CastOp{{"swtof", commons.TYPE_F32}, {"exts", commons.TYPE_F64}}
		}

	case commons.TYPE_I64:
		switch to {
		case commons.TYPE_I32:
			return []CastOp{{"copy", commons.TYPE_I32}}
		case commons.TYPE_F32:
			return []CastOp{{"sltof", commons.TYPE_F32}}
		case commons.TYPE_F64:
			return []CastOp{{"sltof", commons.TYPE_F32}, {"exts", commons.TYPE_F64}}
		}

	case commons.TYPE_F32:
		switch to {
		case commons.TYPE_F64:
			return []CastOp{{"exts", commons.TYPE_F64}}
		case commons.TYPE_I32:
			return []CastOp{{"stosi", commons.TYPE_I32}}
		case commons.TYPE_I64:
			return []CastOp{{"stosi", commons.TYPE_I32}, {"extsw", commons.TYPE_I64}}
		}

	case commons.TYPE_F64:
		switch to {
		case commons.TYPE_F32:
			return []CastOp{{"truncd", commons.TYPE_F32}}
		case commons.TYPE_I32:
			return []CastOp{{"dtosi", commons.TYPE_I32}}
		case commons.TYPE_I64:
			return []CastOp{{"dtosi", commons.TYPE_I32}, {"extsw", commons.TYPE_I64}}
		}
	}

	panic(fmt.Sprintf(
		"invalid cast from %q to %q",
		from.String(),
		to.String(),
	))
}

func (g *Generator) getModExpr(Type commons.ExprType) (string, bool) {
	switch Type {
	case commons.TYPE_I32, commons.TYPE_I64:
		return "rem", false
	case commons.TYPE_F32:
		return "fmodf", true
	case commons.TYPE_F64:
		return "fmod", true
	default:
		panic(fmt.Sprintf(
			"invalid mod for type %q",
			Type.String(),
		))
	}
}

func (g *Generator) emitCast(from, to commons.ExprType, value Value) Value {
	result := value.String

	for _, cast := range g.getCastExpr(from, to) {
		prefix := g.getNumberExprType(cast.Type)

		tmp := g.newTemp(prefix)

		g.Code += fmt.Sprintf(
			"\t%s =%s %s %s\n",
			tmp,
			prefix,
			cast.Op,
			result,
		)

		result = tmp
	}

	return NewValue(result, to)
}

func (g *Generator) GenerateExpr(expr *parser.Expr) Value {
	switch expr.Kind {

	case parser.KIND_LONG:
		return NewValue(fmt.Sprintf("%d", expr.ValueLong), commons.TYPE_I64)

	case parser.KIND_INT:
		return NewValue(fmt.Sprintf("%d", expr.ValueInt), commons.TYPE_I32)

	case parser.KIND_DOUBLE:
		return NewValue(fmt.Sprintf("d_%f", expr.ValueDouble), commons.TYPE_F64)

	case parser.KIND_FLOAT:
		return NewValue(fmt.Sprintf("s_%f", expr.ValueFloat), commons.TYPE_F32)

	case parser.KIND_CAST:
		child := expr.Children[0]

		if child.IsConstant() {
			switch expr.Type {
			case commons.TYPE_I32:
				switch child.Type {
				case commons.TYPE_I64:
					return NewValue(fmt.Sprintf("%d", int32(child.ValueLong)), commons.TYPE_I32)
				case commons.TYPE_F32:
					return NewValue(fmt.Sprintf("%d", int32(child.ValueFloat)), commons.TYPE_I32)
				case commons.TYPE_F64:
					return NewValue(fmt.Sprintf("%d", int32(child.ValueDouble)), commons.TYPE_I32)
				}

			case commons.TYPE_I64:
				switch child.Type {
				case commons.TYPE_I32:
					return NewValue(fmt.Sprintf("%d", int64(child.ValueInt)), commons.TYPE_I64)
				case commons.TYPE_F32:
					return NewValue(fmt.Sprintf("%d", int64(child.ValueFloat)), commons.TYPE_I64)
				case commons.TYPE_F64:
					return NewValue(fmt.Sprintf("%d", int64(child.ValueDouble)), commons.TYPE_I64)
				}

			case commons.TYPE_F32:
				switch child.Type {
				case commons.TYPE_I32:
					return NewValue(fmt.Sprintf("s_%f", float32(child.ValueInt)), commons.TYPE_F32)
				case commons.TYPE_I64:
					return NewValue(fmt.Sprintf("s_%f", float32(child.ValueLong)), commons.TYPE_F32)
				case commons.TYPE_F64:
					return NewValue(fmt.Sprintf("s_%f", float32(child.ValueDouble)), commons.TYPE_F32)
				}

			case commons.TYPE_F64:
				switch child.Type {
				case commons.TYPE_I32:
					return NewValue(fmt.Sprintf("d_%f", float64(child.ValueInt)), commons.TYPE_F64)
				case commons.TYPE_I64:
					return NewValue(fmt.Sprintf("d_%f", float64(child.ValueLong)), commons.TYPE_F64)
				case commons.TYPE_F32:
					return NewValue(fmt.Sprintf("d_%f", float64(child.ValueFloat)), commons.TYPE_F64)
				}
			}
		}

		// cast function when i get a temp variable
		right := g.GenerateExpr(child)

		switch {
		case expr.Type == commons.TYPE_STRING:
			Type := g.getNumberExprType(child.Type)
			tmp := g.newTemp(Type)
			g.Code += fmt.Sprintf("\t%s =l call $%stoa(%s %s)\n", tmp, Type, Type, right.String)
			return NewValue(tmp, commons.TYPE_STRING)

		default:
			fromType := child.Type
			toType := expr.Type
			return g.emitCast(fromType, toType, right)
		}

	case parser.KIND_ADD:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		finalType := g.getBinopExprType(left, right)
		prefix := g.getNumberExprType(finalType)
		tmp := g.newTemp(prefix)

		g.Code += fmt.Sprintf(
			"\t%s =%s add %s, %s\n",
			tmp,
			prefix,
			left.String,
			right.String,
		)

		return NewValue(tmp, finalType)

	case parser.KIND_MUL:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		finalType := g.getBinopExprType(left, right)
		prefix := g.getNumberExprType(finalType)

		tmp := g.newTemp(prefix)

		g.Code += fmt.Sprintf(
			"\t%s =%s mul %s, %s\n",
			tmp,
			prefix,
			left.String,
			right.String,
		)

		return NewValue(tmp, finalType)

	case parser.KIND_SUB:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		finalType := g.getBinopExprType(left, right)
		prefix := g.getNumberExprType(finalType)

		tmp := g.newTemp(prefix)

		g.Code += fmt.Sprintf(
			"\t%s =%s sub %s, %s\n",
			tmp,
			prefix,
			left.String,
			right.String,
		)

		return NewValue(tmp, finalType)

	case parser.KIND_DIV:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		finalType := g.getBinopExprType(left, right)
		prefix := g.getNumberExprType(finalType)

		tmp := g.newTemp(prefix)

		g.Code += fmt.Sprintf(
			"\t%s =%s div %s, %s\n",
			tmp,
			prefix,
			left.String,
			right.String,
		)

		return NewValue(tmp, finalType)

	case parser.KIND_MOD:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		finalType := g.getBinopExprType(left, right)
		prefix := g.getNumberExprType(finalType)

		tmp := g.newTemp(prefix)

		mod, call := g.getModExpr(finalType)

		if call {
			g.Code += fmt.Sprintf(
				"\t%s =%s call $%s(%s %s, %s %s)\n",
				tmp,
				prefix,
				mod,
				prefix,
				left.String,
				prefix,
				right.String,
			)
		} else {
			g.Code += fmt.Sprintf(
				"\t%s =%s %s %s, %s\n",
				tmp,
				prefix,
				mod,
				left.String,
				right.String,
			)
		}

		return NewValue(tmp, finalType)

	case parser.KIND_POW:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		tmp := g.newTemp("d")

		g.Code += fmt.Sprintf(
			"\t%s =d call $pow(d %s, d %s)\n",
			tmp,
			left.String,
			right.String,
		)

		return NewValue(tmp, commons.TYPE_F64)

	case parser.KIND_NEG:
		child := expr.Children[0]

		if child.IsConstant() {
			switch expr.Type {
			case commons.TYPE_I32, commons.TYPE_I16, commons.TYPE_I8, commons.TYPE_U16, commons.TYPE_U8:
				return NewValue(fmt.Sprintf("%d", -child.ValueInt), child.Type)
			case commons.TYPE_I64:
				return NewValue(fmt.Sprintf("%d", -child.ValueLong), child.Type)
			case commons.TYPE_F32:
				return NewValue(fmt.Sprintf("s_%f", -child.ValueFloat), child.Type)
			case commons.TYPE_F64:
				return NewValue(fmt.Sprintf("d_%f", -child.ValueDouble), child.Type)
			}
		}

		left := g.GenerateExpr(child)
		prefix := g.getNumberExprType(left.Type)

		tmp := g.newTemp(prefix)

		g.Code += fmt.Sprintf(
			"\t%s =%s neg %s\n",
			tmp,
			prefix,
			left.String,
		)

		return NewValue(tmp, left.Type)

	case parser.KIND_VARGET:
		Type := g.getNumberExprType(expr.Type)
		LoadType := g.getNumberExprLoadType(expr.Type)

		tmp := g.newTemp(Type)

		g.Code += fmt.Sprintf(
			"\t%s =%s load%s %%%s\n",
			tmp,
			Type,
			LoadType,
			expr.ValueString,
		)

		return NewValue(tmp, expr.Type)

	case parser.KIND_VARSLICEGET:
		Type := g.getNumberExprType(expr.Type)
		LoadType := g.getNumberExprLoadType(expr.Type)
		Size := g.getNumberExprSize(expr.Type)
		idx := expr.ValueInt

		tmp_ptr := g.newTemp("l")
		tmp_val := g.newTemp(Type)

		g.Code += fmt.Sprintf(
			"\t%s =l add %%%s, %d\n\t%s = %s load%s %s\n",
			tmp_ptr,
			expr.ValueString,
			Size*int(idx),
			tmp_val,
			Type,
			LoadType,
			tmp_ptr,
		)

		return NewValue(tmp_val, expr.Type)

	case parser.KIND_VARASSIGN:
		value := g.GenerateExpr(expr.Children[0])

		StoreType := g.getNumberExprStoreType(expr.Type)

		g.Code += fmt.Sprintf(
			"\tstore%s %s, %%%s\n",
			StoreType,
			value.String,
			expr.ValueString,
		)

		return value

	case parser.KIND_VARSLICEASSIGN:
		value := g.GenerateExpr(expr.Children[0])

		StoreType := g.getNumberExprStoreType(expr.Type)
		Size := g.getNumberExprSize(expr.Type)
		idx := expr.ValueInt

		tmp_ptr := g.newTemp("l")

		g.Code += fmt.Sprintf(
			"\t%s =l add %%%s, %d\n\tstore%s %s, %s\n",
			tmp_ptr,
			expr.ValueString,
			Size*int(idx),
			StoreType,
			value.String,
			tmp_ptr,
		)

		return value

	case parser.KIND_STRING:
		if expr.ValueLong < 0 {
			return NewValue("0", commons.TYPE_STRING)
		}
		return NewValue(fmt.Sprintf("$s_%d", expr.ValueLong), commons.TYPE_STRING)

	case parser.KIND_STRCAT:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])

		tmp := g.newTemp("l")

		g.Code += fmt.Sprintf(
			"\t%s = l call $lin_strcat(l %s, l %s)\n",
			tmp,
			left.String,
			right.String,
		)

		return NewValue(tmp, commons.TYPE_STRING)

	case parser.KIND_POSTINC, parser.KIND_POSTDEC:
		oldValue := g.GenerateExpr(expr.Children[0])
		g.GenerateExpr(expr.Children[1])
		return oldValue
	}

	panic(fmt.Sprintf("CODEGEN: unknown expression %q", expr.Kind.String()))
}

func (g *Generator) collectUsedStrings(expr *parser.Expr, used map[int]struct{}) {
	if expr == nil {
		return
	}

	switch expr.Kind {
	case parser.KIND_CONSTSET:
		// for constants i don't need to generate this
		return
	case parser.KIND_STRING:
		if expr.Type == commons.TYPE_STRING && expr.ValueLong >= 0 {
			used[int(expr.ValueLong)] = struct{}{}
		}
	}

	for _, child := range expr.Children {
		g.collectUsedStrings(child, used)
	}
}

func (g *Generator) Generate(expr *parser.Expr) string {
	switch expr.Kind {

	case parser.KIND_PROGRAM:
		if expr.Parser.HasDump {
			g.Code += "data $d_fmt = { b \"%f\\n\", b 0 }\n"
			g.Code += "data $l_fmt = { b \"%ld\\n\", b 0 }\n"
			g.Code += "data $w_fmt = { b \"%d\\n\", b 0 }\n"
			g.Code += "data $s_fmt = { b \"%f\\n\", b 0 }\n"
		}

		usedStrings := make(map[int]struct{})
		g.collectUsedStrings(expr, usedStrings)

		for id, str := range expr.Parser.Strings {
			if _, ok := usedStrings[id]; ok {
				g.Code += fmt.Sprintf("data $s_%d = { b \"%s\", b 0 }\n", id, str)
			}
		}
		g.Code += "export function w $main() {\n@start\n"

		for _, child := range expr.Children {
			g.Generate(child)
		}

		g.Code += "}\n"
		return g.Code

	case parser.KIND_RETURN:
		value := g.GenerateExpr(expr.Children[0])

		g.Code += fmt.Sprintf(
			"\tret %s\n",
			value.String,
		)

		return ""

	case parser.KIND_DUMP:
		value := g.GenerateExpr(expr.Children[0])

		switch expr.Children[0].Type {
		case commons.TYPE_STRING:

			g.Code += fmt.Sprintf(
				"\tcall $printf(l %s)\n",
				value.String,
			)

			return ""
		default:
			Type := g.getNumberExprType(expr.Children[0].Type)

			if expr.Children[0].Type == commons.TYPE_F32 {
				value = g.emitCast(commons.TYPE_F32, commons.TYPE_F64, value)
				Type = "d"
			}

			g.Code += fmt.Sprintf(
				"\tcall $printf(l $%s_fmt, ..., %s %s)\n",
				Type,
				Type,
				value.String,
			)

			return ""
		}

	case parser.KIND_VARINIT:
		if expr.Type == commons.TYPE_ARRAY {
			StoreType := g.getNumberExprStoreType(expr.ValueType)
			ElementSize := g.getNumberExprSize(expr.ValueType)
			ArraySize := int(expr.ValueInt)
			name := expr.ValueString

			g.Code += fmt.Sprintf(
				"\t%%%s = l alloc%d %d\n",
				name,
				ElementSize,
				ArraySize*ElementSize,
			)

			for idx, childExpr := range expr.Children {
				child := g.GenerateExpr(childExpr)
				tmp := g.newTemp("l")
				g.Code += fmt.Sprintf(
					"\t%s = l add %%%s, %d\n\tstore%s %s, %s\n",
					tmp,
					name,
					idx*ElementSize,
					StoreType,
					child.String,
					tmp,
				)
			}

			return ""
		}

		value := g.GenerateExpr(expr.Children[0])

		StoreType := g.getNumberExprStoreType(expr.Type)
		Size := g.getNumberExprSize(expr.Type)
		name := expr.ValueString

		g.Code += fmt.Sprintf(
			"\t%%%s = l alloc%d %d\n\tstore%s %s, %%%s\n",
			name,
			Size,
			Size,
			StoreType,
			value.String,
			name,
		)

		return ""

	case parser.KIND_FREEARENA:
		g.Code += "\tcall $lin_arena_free()\n"

		return ""

	case parser.KIND_CONSTSET:
		return ""

	case parser.KIND_POSTINC, parser.KIND_POSTDEC, parser.KIND_VARASSIGN, parser.KIND_VARSLICEASSIGN:
		g.GenerateExpr(expr)
		return ""
	}

	panic(fmt.Sprintf("CODEGEN: unknown Kind: %q", expr.Kind.String()))
}
