package codegen

import (
	"fmt"
	"lin/commons"
	"lin/parser"
	"strings"
)

type Generator struct {
	NumTemp      int
	Code         string
	Terminated   bool
	Preallocated map[string]bool
	Layouts      map[string]struct {
		Size    int
		Offsets []int
	}
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
	return &Generator{Preallocated: make(map[string]bool)}
}

func (g *Generator) newTemp(prefix string) string {
	name := fmt.Sprintf("%%%s_t%s", prefix, commons.NewHashName(prefix, g.NumTemp))
	g.NumTemp++
	return name
}

func (g *Generator) newLabel(name string, id int) string {
	label := fmt.Sprintf("@__lin_%s_%d", name, id)
	return label
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
	case commons.TYPE_I32, commons.TYPE_I16, commons.TYPE_U16, commons.TYPE_I8, commons.TYPE_U8, commons.TYPE_BOOL:
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
	case commons.TYPE_I32, commons.TYPE_BOOL:
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

func (g *Generator) getComparisonPrefixExprType(Type commons.ExprType) string {
	switch Type {
	case commons.TYPE_F64, commons.TYPE_F32:
		return "c"
	case commons.TYPE_I64, commons.TYPE_I32, commons.TYPE_I16, commons.TYPE_I8:
		return "cs"
	case commons.TYPE_U16, commons.TYPE_U8:
		return "cu"
	default:
		panic(fmt.Sprintf(
			"unsupported comparison type %q",
			Type.String(),
		))
	}
}

func (g *Generator) getComparisonExprKind(Kind parser.ExprKind) string {
	switch Kind {
	case parser.KIND_GT:
		return "gt"
	case parser.KIND_GE:
		return "ge"
	case parser.KIND_LT:
		return "lt"
	case parser.KIND_LE:
		return "le"
	default:
		panic(fmt.Sprintf(
			"unsupported comparison type %q",
			Kind.String(),
		))
	}
}

func (g *Generator) getNumberExprStoreType(Type commons.ExprType) string {
	switch Type {
	case commons.TYPE_F64:
		return "d"
	case commons.TYPE_I64, commons.TYPE_STRING:
		return "l"
	case commons.TYPE_I32, commons.TYPE_BOOL:
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
	case commons.TYPE_I32, commons.TYPE_BOOL:
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

func (g *Generator) rawCaptureValue(sym *parser.Symbol) Value {
	if sym.Kind == parser.SYMBOL_CONST {
		return g.GenerateExpr(sym.Value)
	}

	typ := g.getNumberExprType(sym.Type)
	loadType := g.getNumberExprLoadType(sym.Type)
	tmp := g.newTemp(typ)
	g.Code += fmt.Sprintf("\t%s =%s load%s %%%s\n", tmp, typ, loadType, sym.Internal)
	return NewValue(tmp, sym.Type)
}

func (g *Generator) expandRawQBE(source string, captures map[string]*parser.Symbol) string {
	var out strings.Builder
	for pos := 0; pos < len(source); {
		start := strings.Index(source[pos:], "${")
		if start < 0 {
			out.WriteString(source[pos:])
			break
		}
		start += pos
		out.WriteString(source[pos:start])
		end := strings.IndexByte(source[start+2:], '}')
		if end < 0 {
			panic("CODEGEN: unterminated raw QBE capture")
		}
		end += start + 2
		name := source[start+2 : end]
		sym, ok := captures[name]
		if !ok {
			panic(fmt.Sprintf("CODEGEN: unknown raw QBE capture %q", name))
		}
		out.WriteString(g.rawCaptureValue(sym).String)
		pos = end + 1
	}
	return out.String()
}

func (g *Generator) getFieldSize(field *parser.Field) int {
	switch field.Type {
	case commons.TYPE_STRUCT:
		return g.getStructSize(field.TypeSymbol)
	case commons.TYPE_ARRAY:
		return field.Size * g.getTypeSize(field.Element, field.ElementSym)
	default:
		return g.getNumberExprSize(field.Type)
	}
}

func (g *Generator) getStructSize(sym *parser.Symbol) int {
	if g.Layouts == nil {
		g.Layouts = make(map[string]struct {
			Size    int
			Offsets []int
		})
	}
	if l, ok := g.Layouts[sym.Internal]; ok {
		return l.Size
	}

	size := 0
	offsets := make([]int, 0, len(sym.Fields))
	for _, f := range sym.Fields {
		offsets = append(offsets, size)
		size += g.getFieldSize(f)
	}
	g.Layouts[sym.Internal] = struct {
		Size    int
		Offsets []int
	}{Size: size, Offsets: offsets}
	return size
}

func symElemInfo(sym *parser.Symbol) *parser.Symbol {
	switch sym.Type {
	case commons.TYPE_STRUCT:
		if sym.TypeSymbol != nil {
			return sym.TypeSymbol
		}
		return sym
	case commons.TYPE_ARRAY:
		return sym
	default:
		return nil
	}
}

func (g *Generator) getTypeSize(t commons.ExprType, sym *parser.Symbol) int {
	switch t {
	case commons.TYPE_STRUCT:
		return g.getStructSize(sym)
	case commons.TYPE_ARRAY:
		return sym.Size * g.getTypeSize(sym.Element, sym.ElementSym)
	default:
		return g.getNumberExprSize(t)
	}
}

func (g *Generator) getTypeAlign(t commons.ExprType, sym *parser.Symbol) int {
	// If symbol metadata is missing, fallback to a sensible default alignment.
	// It's quite an hack but who cares
	if sym == nil {
		return 8
	}

	switch t {
	case commons.TYPE_STRUCT:
		max := 1
		for _, f := range sym.Fields {
			var a int
			if f.Type == commons.TYPE_STRUCT {
				a = g.getTypeAlign(commons.TYPE_STRUCT, f.TypeSymbol)
			} else if f.Type == commons.TYPE_ARRAY {
				a = g.getTypeAlign(commons.TYPE_ARRAY, f.ElementSym)
			} else {
				a = g.getNumberExprSize(f.Type)
			}
			if a > max {
				max = a
			}
		}
		if max > 16 {
			max = 16
		}
		return max
	case commons.TYPE_ARRAY:
		return g.getTypeAlign(sym.Element, sym.ElementSym)
	default:
		return g.getNumberExprSize(t)
	}
}

func allocOp(align int) string {
	switch {
	case align <= 4:
		return "alloc4"
	case align <= 8:
		return "alloc8"
	default:
		return "alloc16"
	}
}

// resolveAddr computes the address and element type/symbol for a lvalue expr
func (g *Generator) resolveAddr(expr *parser.Expr) (ptr string, elemType commons.ExprType, elemSym *parser.Symbol) {
	switch expr.Kind {
	case parser.KIND_VARGET:
		sym := expr.ValueSymbol
		return "%" + sym.Internal, sym.Type, symElemInfo(sym)

	case parser.KIND_VARSLICEGET:
		basePtr, baseType, baseSym := g.resolveAddr(expr.Children[0])
		if baseType != commons.TYPE_ARRAY && baseType != commons.TYPE_STRING {
			panic("CODEGEN: cannot index a non-array value")
		}

		index := g.GenerateExpr(expr.Children[1])
		indexLong := g.newTemp("l")
		g.Code += fmt.Sprintf("\t%s =l extuw %s\n", indexLong, index.String)

		// Prefer per-element symbol information when available and the index
		// is a constant into an array initializer. This allows arrays of
		// literals to contain elements with different nested shapes while
		// still enabling compile-time indexing on constant indices.
		// For arrays, prefer element symbol; for strings, element is u8
		if baseType == commons.TYPE_STRING {
			elemType = commons.TYPE_U8
			elemSym = nil
		} else {
			elemType = baseSym.Element
			elemSym = baseSym.ElementSym
		}
		if expr.Children[1].IsConstant() && baseSym != nil && baseSym.Value != nil {
			idx := int(expr.Children[1].ValueInt)
			if idx >= 0 && idx < len(baseSym.Value.Children) {
				child := baseSym.Value.Children[idx]
				if child != nil && child.ValueSymbol != nil {
					// if the specific stored element has its own array symbol, use it
					// to compute sizes for further indexing
					elemType = child.ValueSymbol.Type
					elemSym = child.ValueSymbol
				}
			}
		}
		// For array storage, elements that are arrays are stored as pointers
		// (8 bytes) in the outer array. Use the storage slot size when
		// computing byte offsets (not the total inline size of nested
		// arrays. Strings are byte arrays, so slot size is 1.)
		var slotSize int
		if baseType == commons.TYPE_STRING {
			slotSize = 1
		} else if elemType == commons.TYPE_ARRAY {
			slotSize = 8
		} else {
			slotSize = g.getTypeSize(elemType, elemSym)
		}

		offset := g.newTemp("l")
		g.Code += fmt.Sprintf("\t%s =l mul %s, %d\n", offset, indexLong, slotSize)

		addr := g.newTemp("l")

		if baseType == commons.TYPE_STRING {
			// load stored pointer to string data then add offset
			innerPtr := g.newTemp("l")
			g.Code += fmt.Sprintf("\t%s =l loadl %s\n", innerPtr, basePtr)
			g.Code += fmt.Sprintf("\t%s =l add %s, %s\n", addr, innerPtr, offset)
			return addr, elemType, elemSym
		}

		// For arrays, addr points to the slot; if element is array we need
		// to load the stored inner pointer from that slot.
		g.Code += fmt.Sprintf("\t%s =l add %s, %s\n", addr, basePtr, offset)
		if elemType == commons.TYPE_ARRAY {
			innerPtr := g.newTemp("l")
			g.Code += fmt.Sprintf("\t%s =l loadl %s\n", innerPtr, addr)
			return innerPtr, elemType, elemSym
		}

		return addr, elemType, elemSym

	case parser.KIND_VARFIELDGET:
		basePtr, baseType, baseSym := g.resolveAddr(expr.Children[0])
		if baseType != commons.TYPE_STRUCT {
			panic("CODEGEN: cannot access a field of a non-struct value")
		}

		idx := int(expr.ValueInt)
		field := baseSym.Fields[idx]

		offset := 0
		for j := 0; j < idx; j++ {
			offset += g.getFieldSize(baseSym.Fields[j])
		}

		addr := basePtr
		if offset != 0 {
			addr = g.newTemp("l")
			g.Code += fmt.Sprintf("\t%s =l add %s, %d\n", addr, basePtr, offset)
		}

		var fieldSym *parser.Symbol
		switch field.Type {
		case commons.TYPE_STRUCT:
			fieldSym = field.TypeSymbol
		case commons.TYPE_ARRAY:
			fieldSym = &parser.Symbol{Type: commons.TYPE_ARRAY, Element: field.Element, ElementSym: field.ElementSym, Size: field.Size}
		}
		return addr, field.Type, fieldSym
	default:
		panic(fmt.Sprintf("CODEGEN: resolveAddr: unexpected node %q", expr.Kind.String()))
	}
}

func (g *Generator) generateGet(expr *parser.Expr) Value {
	addr, resultType, _ := g.resolveAddr(expr)

	if resultType == commons.TYPE_STRUCT || resultType == commons.TYPE_ARRAY {
		return NewValue(addr, resultType)
	}

	prefix := g.getNumberExprType(resultType)
	loadType := g.getNumberExprLoadType(resultType)
	tmp := g.newTemp(prefix)
	g.Code += fmt.Sprintf("\t%s =%s load%s %s\n", tmp, prefix, loadType, addr)
	return NewValue(tmp, resultType)
}

func (g *Generator) generateAssign(target, valueExpr *parser.Expr) Value {
	addr, resultType, elemSym := g.resolveAddr(target)
	value := g.GenerateExpr(valueExpr)

	if resultType == commons.TYPE_STRUCT || resultType == commons.TYPE_ARRAY {
		size := g.getTypeSize(resultType, elemSym)
		src := value.String
		dst := addr
		g.Code += fmt.Sprintf("\tcall $memcpy(l %s, l %s, l %d)\n", dst, src, size)
		return value
	}

	storeType := g.getNumberExprStoreType(resultType)
	g.Code += fmt.Sprintf("\tstore%s %s, %s\n", storeType, value.String, addr)
	return value
}

// Emits store recurevely starting basePtr+baseOffset
func (g *Generator) generateStructFields(structExpr *parser.Expr, basePtr string, baseOffset int) {
	typeSym := structExpr.ValueSymbol
	offset := baseOffset

	for i, field := range typeSym.Fields {
		childExpr := structExpr.Children[i]

		if field.Type == commons.TYPE_STRUCT {
			g.generateStructFields(childExpr, basePtr, offset)
		} else if field.Type == commons.TYPE_ARRAY {
			// store array elements into struct memory at basePtr+offset
			elemType := field.Element
			elemSym := field.ElementSym
			elemSize := g.getTypeSize(elemType, elemSym)

			// If the initializer is a literal with element children, expand elementwise.
			// If the initializer is a direct variable reference (or a single-child
			// wrapper that references a variable), perform a single memcpy from
			// the source array address into the struct field. This avoids
			// attempting to store an address into each primitive element.
			if childExpr != nil {
				// Direct variable reference -> memcpy
				if childExpr.Kind == parser.KIND_VARGET || childExpr.Kind == parser.KIND_VARFIELDGET || childExpr.Kind == parser.KIND_VARSLICEGET {
					// memcpy path handled below by falling through to the else-case
					// so normalize to a "no-children" situation by setting
					// a flag. We'll reuse the existing memcpy code.
				}
			}

			// Direct variable reference as initializer -> memcpy
			if childExpr != nil && (childExpr.Kind == parser.KIND_VARGET || childExpr.Kind == parser.KIND_VARFIELDGET || childExpr.Kind == parser.KIND_VARSLICEGET) {
				src := g.GenerateExpr(childExpr)
				dst := basePtr
				if offset != 0 {
					tmp := g.newTemp("l")
					g.Code += fmt.Sprintf("\t%s =l add %s, %d\n", tmp, basePtr, offset)
					dst = tmp
				}
				total := elemSize * field.Size
				g.Code += fmt.Sprintf("\tcall $memcpy(l %s, l %s, l %d)\n", dst, src.String, total)
				offset += g.getFieldSize(field)
				continue
			}

			if childExpr != nil && len(childExpr.Children) > 0 {
				// If any child is itself a variable reference, prefer a single
				// memcpy from the source array to avoid storing addresses
				// into primitive elements.
				anyVarRef := false
				for _, ch := range childExpr.Children {
					if ch.Kind == parser.KIND_VARGET || ch.Kind == parser.KIND_VARFIELDGET || ch.Kind == parser.KIND_VARSLICEGET {
						anyVarRef = true
						break
					}
				}
				if anyVarRef {
					// Copy whole array from source address
					src := g.GenerateExpr(childExpr)
					dst := basePtr
					if offset != 0 {
						tmp := g.newTemp("l")
						g.Code += fmt.Sprintf("\t%s =l add %s, %d\n", tmp, basePtr, offset)
						dst = tmp
					}
					total := elemSize * field.Size
					g.Code += fmt.Sprintf("\tcall $memcpy(l %s, l %s, l %d)\n", dst, src.String, total)
					// advance offset and continue outer loop
					offset += g.getFieldSize(field)
					continue
				}

				for idx, child := range childExpr.Children {
					if elemType == commons.TYPE_STRUCT {
						g.generateStructFields(child, basePtr, offset+idx*elemSize)
						continue
					}
					val := g.GenerateExpr(child)
					storeType := g.getNumberExprStoreType(elemType)

					ptr := basePtr
					if offset+idx*elemSize != 0 {
						tmp := g.newTemp("l")
						g.Code += fmt.Sprintf("\t%s =l add %s, %d\n", tmp, basePtr, offset+idx*elemSize)
						ptr = tmp
					}

					g.Code += fmt.Sprintf("\tstore%s %s, %s\n", storeType, val.String, ptr)
				}
			} else {
				// Copy whole array from source address
				src := g.GenerateExpr(childExpr)
				dst := basePtr
				if offset != 0 {
					tmp := g.newTemp("l")
					g.Code += fmt.Sprintf("\t%s =l add %s, %d\n", tmp, basePtr, offset)
					dst = tmp
				}
				total := elemSize * field.Size
				g.Code += fmt.Sprintf("\tcall $memcpy(l %s, l %s, l %d)\n", dst, src.String, total)
			}
		} else {
			val := g.GenerateExpr(childExpr)
			storeType := g.getNumberExprStoreType(field.Type)

			ptr := basePtr
			if offset != 0 {
				tmp := g.newTemp("l")
				g.Code += fmt.Sprintf("\t%s =l add %s, %d\n", tmp, basePtr, offset)
				ptr = tmp
			}

			g.Code += fmt.Sprintf("\tstore%s %s, %s\n", storeType, val.String, ptr)
		}

		offset += g.getFieldSize(field)
	}
}

func (g *Generator) allocateStruct(expr *parser.Expr, basePtr string) Value {
	typeSym := expr.ValueSymbol

	size := g.getStructSize(typeSym)

	align := g.getTypeAlign(commons.TYPE_STRUCT, typeSym)
	op := allocOp(align)
	g.Code += fmt.Sprintf("\t%s = l %s %d\n", basePtr, op, size)
	g.generateStructFields(expr, basePtr, 0)

	return NewValue(basePtr, expr.Type)
}

func (g *Generator) GenerateExpr(expr *parser.Expr) Value {
	switch expr.Kind {
	case parser.KIND_VARGET, parser.KIND_VARFIELDGET, parser.KIND_VARSLICEGET:
		return g.generateGet(expr)

	case parser.KIND_VARFIELDASSIGN, parser.KIND_VARSLICEASSIGN, parser.KIND_VARASSIGN:
		return g.generateAssign(expr.Children[0], expr.Children[1])
	case parser.KIND_STRUCT:
		tmp := g.newTemp("l")
		return g.allocateStruct(expr, tmp)

	case parser.KIND_RAW_QBE:
		g.Code += g.expandRawQBE(expr.ValueString, expr.RawCaptures)
		if !strings.HasSuffix(g.Code, "\n") {
			g.Code += "\n"
		}
		return NewValue(g.expandRawQBE(expr.RawResult, expr.RawCaptures), expr.Type)

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
			g.Code += fmt.Sprintf("\t%s =l call $lin_%stoa(%s %s)\n", tmp, Type, Type, right.String)
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

	case parser.KIND_VARINIT:
		// If this variable was preallocated in the function prologue, skip
		// emitting the allocation here; only perform initialization stores.
		internal := expr.ValueSymbol.Internal
		if g.Preallocated[internal] {
			switch expr.Type {
			case commons.TYPE_ARRAY:
				if expr.ValueType == commons.TYPE_STRUCT {
					elementSym := expr.ValueSymbol.ElementSym
					if elementSym == nil {
						panic("CODEGEN: missing type symbol for array of structs")
					}
					elementSize := g.getStructSize(elementSym)
					for idx, childExpr := range expr.Children {
						g.generateStructFields(childExpr, "%"+internal, idx*elementSize)
					}
					return NewValue("%"+internal, expr.Type)
				}
				if expr.ValueType == commons.TYPE_ARRAY {
					elementSize := 8
					for idx, childExpr := range expr.Children {
						child := g.GenerateExpr(childExpr)
						tmp := g.newTemp("l")
						g.Code += fmt.Sprintf(
							"\t%s = l add %%%s, %d\n\tstorel %s, %s\n",
							tmp,
							internal,
							idx*elementSize,
							child.String,
							tmp,
						)
					}
					return NewValue("%"+internal, expr.Type)
				}
				StoreType := g.getNumberExprStoreType(expr.ValueType)
				ElementSize := g.getNumberExprSize(expr.ValueType)
				for idx, childExpr := range expr.Children {
					child := g.GenerateExpr(childExpr)
					tmp := g.newTemp("l")
					g.Code += fmt.Sprintf(
						"\t%s = l add %%%s, %d\n\tstore%s %s, %s\n",
						tmp,
						internal,
						idx*ElementSize,
						StoreType,
						child.String,
						tmp,
					)
				}
				return NewValue("%"+internal, expr.Type)
			case commons.TYPE_STRUCT:
				structExpr := expr.Children[0]
				g.generateStructFields(structExpr, "%"+internal, 0)
				return NewValue("%"+internal, expr.Type)
			default:
				value := g.GenerateExpr(expr.Children[0])
				StoreType := g.getNumberExprStoreType(expr.Type)
				g.Code += fmt.Sprintf("\tstore%s %s, %%%s\n", StoreType, value.String, internal)
				return NewValue("%"+internal, expr.Type)
			}
		}
		switch expr.Type {
		case commons.TYPE_ARRAY:
			internal := expr.ValueSymbol.Internal
			if expr.ValueType == commons.TYPE_STRUCT {
				elementSym := expr.ValueSymbol.ElementSym
				if elementSym == nil {
					panic("CODEGEN: missing type symbol for array of structs")
				}
				elementSize := g.getStructSize(elementSym)
				totalSize := int(expr.ValueInt) * elementSize

				align := g.getTypeAlign(commons.TYPE_STRUCT, elementSym)
				op := allocOp(align)
				g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, totalSize)
				for idx, childExpr := range expr.Children {
					g.generateStructFields(childExpr, "%"+internal, idx*elementSize)
				}
				return NewValue("%"+internal, expr.Type)
			}
			if expr.ValueType == commons.TYPE_ARRAY {
				elementSize := 8
				totalSize := int(expr.ValueInt) * elementSize

				align := 8
				op := allocOp(align)
				g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, totalSize)
				for idx, childExpr := range expr.Children {
					child := g.GenerateExpr(childExpr)
					tmp := g.newTemp("l")
					g.Code += fmt.Sprintf(
						"\t%s = l add %%%s, %d\n\tstorel %s, %s\n",
						tmp,
						internal,
						idx*elementSize,
						child.String,
						tmp,
					)
				}
				return NewValue("%"+internal, expr.Type)
			}

			StoreType := g.getNumberExprStoreType(expr.ValueType)
			ElementSize := g.getNumberExprSize(expr.ValueType)
			ArraySize := int(expr.ValueInt)

			op := allocOp(ElementSize)
			g.Code += fmt.Sprintf(
				"\t%%%s = l %s %d\n",
				internal,
				op,
				ArraySize*ElementSize,
			)

			for idx, childExpr := range expr.Children {
				child := g.GenerateExpr(childExpr)
				tmp := g.newTemp("l")
				g.Code += fmt.Sprintf(
					"\t%s = l add %%%s, %d\n\tstore%s %s, %s\n",
					tmp,
					internal,
					idx*ElementSize,
					StoreType,
					child.String,
					tmp,
				)
			}

			return NewValue("%"+internal, expr.Type)
		case commons.TYPE_STRUCT:
			structExpr := expr.Children[0]
			internal := expr.ValueSymbol.Internal

			return g.allocateStruct(structExpr, "%"+internal)

		default:
			value := g.GenerateExpr(expr.Children[0])
			StoreType := g.getNumberExprStoreType(expr.Type)
			Size := g.getNumberExprSize(expr.Type)
			internal := expr.ValueSymbol.Internal

			op := allocOp(Size)
			g.Code += fmt.Sprintf(
				"\t%%%s = l %s %d\n\tstore%s %s, %%%s\n",
				internal, op, Size, StoreType, value.String, internal,
			)
			return NewValue("%"+internal, expr.Type)
		}

	//case parser.KIND_VARASSIGN:
	//	value := g.GenerateExpr(expr.Children[0])
	//
	//	if expr.Type == commons.TYPE_STRUCT {
	//		// write struct fields into the variable's storage
	//		structExpr := expr.Children[0]
	//		g.generateStructFields(structExpr, "%"+expr.ValueSymbol.Internal, 0)
	//		return value
	//	}
	//
	//	StoreType := g.getNumberExprStoreType(expr.Type)
	//
	//	g.Code += fmt.Sprintf(
	//		"\tstore%s %s, %%%s\n",
	//		StoreType,
	//		value.String,
	//		expr.ValueSymbol.Internal,
	//	)
	//
	//	return value

	case parser.KIND_STRING:
		if expr.ValueLong < 0 {
			return NewValue("0", commons.TYPE_STRING)
		}
		return NewValue(fmt.Sprintf("$s_%d", expr.ValueLong), commons.TYPE_STRING)

	case parser.KIND_BOOL:
		return NewValue(fmt.Sprintf("%d", expr.ValueInt), commons.TYPE_BOOL)

	case parser.KIND_AND:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])
		tmp := g.newTemp("w")
		g.Code += fmt.Sprintf(
			"\t%s = w and %s, %s\n",
			tmp,
			left.String,
			right.String,
		)
		return NewValue(tmp, commons.TYPE_BOOL)

	case parser.KIND_OR:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])
		tmp := g.newTemp("w")
		g.Code += fmt.Sprintf(
			"\t%s = w or %s, %s\n",
			tmp,
			left.String,
			right.String,
		)
		return NewValue(tmp, commons.TYPE_BOOL)

	case parser.KIND_EQ:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])
		tmp := g.newTemp("w")

		//Basically i check this ahead in parsing, both left and right are the same type
		if left.Type == commons.TYPE_STRING {
			g.Code += fmt.Sprintf(
				"\t%s = w call $lin_str_eq(l %s, l %s)\n",
				tmp,
				left.String,
				right.String,
			)
		} else {
			finalType := g.getNumberExprType(left.Type)
			g.Code += fmt.Sprintf(
				"\t%s = w ceq%s %s, %s\n",
				tmp,
				finalType,
				left.String,
				right.String,
			)
		}
		return NewValue(tmp, commons.TYPE_BOOL)

	case parser.KIND_NEQ:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])
		tmp := g.newTemp("w")

		//Basically i check this ahead in parsing, both left and right are the same type
		if left.Type == commons.TYPE_STRING {
			g.Code += fmt.Sprintf(
				"\t%s = w call $lin_str_ne(l %s, l %s)\n",
				tmp,
				left.String,
				right.String,
			)
		} else {
			finalType := g.getNumberExprType(left.Type)
			g.Code += fmt.Sprintf(
				"\t%s = w cne%s %s, %s\n",
				tmp,
				finalType,
				left.String,
				right.String,
			)
		}
		return NewValue(tmp, commons.TYPE_BOOL)

	case parser.KIND_GT, parser.KIND_GE, parser.KIND_LT, parser.KIND_LE:
		left := g.GenerateExpr(expr.Children[0])
		right := g.GenerateExpr(expr.Children[1])
		tmp := g.newTemp("w")

		finalType := g.getNumberExprType(left.Type)
		comparisonPrefix := g.getComparisonPrefixExprType(left.Type)
		comparisonExpr := g.getComparisonExprKind(expr.Kind)

		g.Code += fmt.Sprintf(
			"\t%s = w %s%s%s %s, %s\n",
			tmp,
			comparisonPrefix,
			comparisonExpr,
			finalType,
			left.String,
			right.String,
		)

		return NewValue(tmp, commons.TYPE_BOOL)

	case parser.KIND_NOT:
		left := g.GenerateExpr(expr.Children[0])
		tmp := g.newTemp("w")

		g.Code += fmt.Sprintf(
			"\t%s = w xor %s, 1\n",
			tmp,
			left.String,
		)

		return NewValue(tmp, commons.TYPE_BOOL)

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

	case parser.KIND_PREINC, parser.KIND_PREDEC:
		return g.GenerateExpr(expr.Children[0])

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
func (g *Generator) GenerateStatement(expr *parser.Expr) bool {
	switch expr.Kind {
	case parser.KIND_RAW_QBE:
		g.GenerateExpr(expr)
		return false

	case parser.KIND_RETURN:
		value := g.GenerateExpr(expr.Children[0])

		// free arena before returning to avoid double-free: codegen emits single call
		g.Code += "\tcall $lin_arena_free()\n"

		g.Code += fmt.Sprintf(
			"\tret %s\n",
			value.String,
		)
		g.Terminated = true
		return true

	case parser.KIND_DUMP:
		value := g.GenerateExpr(expr.Children[0])

		switch expr.Children[0].Type {
		case commons.TYPE_STRING:

			g.Code += fmt.Sprintf(
				"\tcall $printf(l %s)\n",
				value.String,
			)

			return false
		case commons.TYPE_U8, commons.TYPE_I8:
			// print as numeric value for small integers
			g.Code += fmt.Sprintf(
				"\tcall $printf(l $w_fmt, ..., w %s)\n",
				value.String,
			)

			return false
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

			return false
		}

	case parser.KIND_FREEARENA:
		g.Code += "\tcall $lin_arena_free()\n"

		return false

	case parser.KIND_CONSTSET, parser.KIND_TYPEDECL:
		return false

	case parser.KIND_BODY:
		for _, child := range expr.Children {
			if g.GenerateStatement(child) {
				return true
			}
		}

		return false

	case parser.KIND_IF:
		cond := g.GenerateExpr(expr.Children[0])
		thenBody := expr.Children[1]
		id := expr.ID

		elseLabel := g.newLabel("if_else", id)
		endLabel := g.newLabel("if_end", id)
		thenLabel := g.newLabel("if_then", id)

		if expr.HasElse {
			g.Code += fmt.Sprintf(
				"\tjnz %s, %s, %s\n",
				cond.String,
				thenLabel,
				elseLabel,
			)
		} else {
			g.Code += fmt.Sprintf(
				"\tjnz %s, %s, %s\n",
				cond.String,
				thenLabel,
				endLabel,
			)
		}

		g.Code += thenLabel + "\n"

		if !g.GenerateStatement(thenBody) {
			g.Code += "\tjmp " + endLabel + "\n"
		}

		if expr.HasElse {
			g.Code += elseLabel + "\n"

			if !g.GenerateStatement(expr.Children[2]) {
				g.Code += "\tjmp " + endLabel + "\n"
			}
		}

		g.Code += endLabel + "\n"

		return false

	case parser.KIND_WHILE:
		id := expr.ID

		condLabel := g.newLabel("while_cond", id)
		bodyLabel := g.newLabel("while_body", id)
		endLabel := g.newLabel("while_end", id)

		g.Code += condLabel + "\n"

		//condition in [0]
		cond := g.GenerateExpr(expr.Children[0])

		g.Code += fmt.Sprintf(
			"\tjnz %s, %s, %s\n",
			cond.String,
			bodyLabel,
			endLabel,
		)

		g.Code += bodyLabel + "\n"

		// Body in [1]
		if !g.GenerateStatement(expr.Children[1]) {
			g.Code += "\tjmp " + condLabel + "\n"
		}

		g.Code += endLabel + "\n"

		return false

	case parser.KIND_FOR:
		id := expr.ID

		condLabel := g.newLabel("for_cond", id)
		bodyLabel := g.newLabel("for_body", id)
		endLabel := g.newLabel("for_end", id)

		//begin in [0]
		g.GenerateStatement(expr.Children[0])

		g.Code += condLabel + "\n"

		cond := g.GenerateExpr(expr.Children[1])

		g.Code += fmt.Sprintf(
			"\tjnz %s, %s, %s\n",
			cond.String,
			bodyLabel,
			endLabel,
		)

		g.Code += bodyLabel + "\n"

		// Body in [3]
		if !g.GenerateStatement(expr.Children[3]) {
			updt := expr.Children[2]
			if updt.Kind != parser.KIND_NONE {
				g.GenerateExpr(updt)
			}
			g.Code += "\tjmp " + condLabel + "\n"
		}

		g.Code += endLabel + "\n"

		return false

	case parser.KIND_SWITCH:
		switchID := expr.ID

		startLabel := g.newLabel("switch_start", switchID)
		endLabel := g.newLabel("switch_end", switchID)

		g.Code += startLabel + "\n"

		// Default destination
		defaultLabel := endLabel

		if expr.ValueSwitch.Default >= 0 {
			defaultLabel = g.newLabel("switch_default", switchID)
		}

		// Generate labels for every case body.
		type caseInfo struct {
			expr   *parser.Expr
			body   string
			checks []string
		}

		casesInfo := make([]caseInfo, 0, len(expr.ValueSwitch.Cases))

		for _, index := range expr.ValueSwitch.Cases {
			c := expr.Children[index]
			info := caseInfo{
				expr: c,
				body: g.newLabel(
					fmt.Sprintf("case_%d_body", c.ID),
					switchID,
				),
			}

			for _, check := range c.Children[:len(c.Children)-1] {
				info.checks = append(
					info.checks,
					g.newLabel(
						fmt.Sprintf(
							"case_%d_check_%d",
							c.ID,
							check.ID,
						),
						switchID,
					),
				)
			}

			casesInfo = append(casesInfo, info)
		}

		// Generate comparison chain.
		for i, c := range casesInfo {
			for j, checkExpr := range c.expr.Children[:len(c.expr.Children)-1] {
				checkLabel := c.checks[j]

				g.Code += checkLabel + "\n"

				cmp := g.GenerateExpr(checkExpr)

				var nextLabel string

				if j+1 < len(c.checks) {
					nextLabel = c.checks[j+1]
				} else if i+1 < len(casesInfo) {
					nextLabel = casesInfo[i+1].checks[0]
				} else {
					nextLabel = defaultLabel
				}

				g.Code += fmt.Sprintf(
					"\tjnz %s, %s, %s\n",
					cmp.String,
					c.body,
					nextLabel,
				)
			}
		}

		// First entry point.
		if len(casesInfo) == 0 {
			g.Code += fmt.Sprintf(
				"\tjmp %s\n",
				defaultLabel,
			)
		}

		// Bodies.
		for _, c := range casesInfo {
			g.Code += c.body + "\n"

			body := c.expr.Children[len(c.expr.Children)-1]

			for _, child := range body.Children {
				if g.GenerateStatement(child) {
					g.Terminated = true
					break
				}
			}

			if !g.Terminated {
				g.Code += fmt.Sprintf(
					"\tjmp %s\n",
					endLabel,
				)
			}

			g.Terminated = false
		}

		// Default.
		if expr.ValueSwitch.Default >= 0 {
			g.Code += defaultLabel + "\n"

			body := expr.Children[expr.ValueSwitch.Default]

			for _, child := range body.Children {
				if g.GenerateStatement(child) {
					g.Terminated = true
					break
				}
			}

			if !g.Terminated {
				g.Code += fmt.Sprintf(
					"\tjmp %s\n",
					endLabel,
				)
			}

			g.Terminated = false
		}

		g.Code += endLabel + "\n"

		return false

	case parser.KIND_BREAK:
		ctx := expr.ValueContext
		id := ctx.ID

		var endLabel string

		switch ctx.BodyKind {
		case parser.CONTEXT_FOR:
			endLabel = g.newLabel("for_end", id)
		case parser.CONTEXT_WHILE:
			endLabel = g.newLabel("while_end", id)
		case parser.CONTEXT_CASE:
			endLabel = g.newLabel("switch_end", id)
		}

		g.Code += "\tjmp " + endLabel + "\n"

		return true

	case parser.KIND_CONTINUE:
		ctx := expr.ValueContext
		id := ctx.ID

		var condLabel string

		switch ctx.BodyKind {
		case parser.CONTEXT_FOR:
			condLabel = g.newLabel("for_cond", id)
		case parser.CONTEXT_WHILE:
			condLabel = g.newLabel("while_cond", id)
		case parser.CONTEXT_CASE:
			condLabel = g.newLabel("switch_start", id)
		}

		g.Code += "\tjmp " + condLabel + "\n"

		return true

	case parser.KIND_POSTINC, parser.KIND_POSTDEC, parser.KIND_PREINC, parser.KIND_PREDEC, parser.KIND_VARINIT, parser.KIND_VARASSIGN, parser.KIND_VARSLICEASSIGN,parser.KIND_VARFIELDASSIGN:
		g.GenerateExpr(expr)
		return false
	}

	panic(fmt.Sprintf("CODEGEN: unknown statement: %q", expr.Kind.String()))
}

func (g *Generator) collectStructsFromContext(ctx *parser.Context) []*parser.Symbol {
	var out []*parser.Symbol
	for _, sym := range ctx.Symbols {
		if sym.Kind == parser.SYMBOL_TYPE && sym.Type == commons.TYPE_STRUCT {
			out = append(out, sym)
		}
	}
	for _, child := range ctx.Children {
		out = append(out, g.collectStructsFromContext(child)...)
	}
	return out
}

func (g *Generator) collectVarInits(expr *parser.Expr, out *[]*parser.Expr) {
	if expr == nil {
		return
	}
	if expr.Kind == parser.KIND_VARINIT {
		*out = append(*out, expr)
	}
	for _, c := range expr.Children {
		g.collectVarInits(c, out)
	}
}

func (g *Generator) preAllocateVars(program *parser.Expr) {
	var varinits []*parser.Expr
	for _, child := range program.Children {
		g.collectVarInits(child, &varinits)
	}

	for _, expr := range varinits {
		internal := expr.ValueSymbol.Internal
		if g.Preallocated[internal] {
			continue
		}

		switch expr.Type {
		case commons.TYPE_ARRAY:
			if expr.ValueType == commons.TYPE_STRUCT {
				elementSym := expr.ValueSymbol.ElementSym
				if elementSym == nil {
					panic("CODEGEN: missing type symbol for array of structs")
				}
				elementSize := g.getStructSize(elementSym)
				totalSize := int(expr.ValueInt) * elementSize
				align := g.getTypeAlign(commons.TYPE_STRUCT, elementSym)
				op := allocOp(align)
				g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, totalSize)
			} else if expr.ValueType == commons.TYPE_ARRAY {
				elementSize := 8
				totalSize := int(expr.ValueInt) * elementSize
				op := allocOp(elementSize)
				g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, totalSize)
			} else {
				ElementSize := g.getNumberExprSize(expr.ValueType)
				ArraySize := int(expr.ValueInt)
				op := allocOp(ElementSize)
				g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, ArraySize*ElementSize)
			}
		case commons.TYPE_STRUCT:
			// For struct variable, compute size from type symbol if available
			var typeSym *parser.Symbol
			if expr.ValueSymbol != nil && expr.ValueSymbol.TypeSymbol != nil {
				typeSym = expr.ValueSymbol.TypeSymbol
			} else if len(expr.Children) > 0 && expr.Children[0].ValueSymbol != nil {
				typeSym = expr.Children[0].ValueSymbol
			}
			if typeSym == nil {
				panic("CODEGEN: missing type symbol for struct varinit")
			}
			size := g.getStructSize(typeSym)
			align := g.getTypeAlign(commons.TYPE_STRUCT, typeSym)
			op := allocOp(align)
			g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, size)
		default:
			Size := g.getNumberExprSize(expr.Type)
			op := allocOp(Size)
			g.Code += fmt.Sprintf("\t%%%s = l %s %d\n", internal, op, Size)
		}

		g.Preallocated[internal] = true
	}
}

func (g *Generator) qbeFieldTypeRef(field *parser.Field) string {
	var build func(t commons.ExprType, sym *parser.Symbol, size int) string
	build = func(t commons.ExprType, sym *parser.Symbol, size int) string {
		switch t {
		case commons.TYPE_STRUCT:
			return ":" + sym.Internal
		case commons.TYPE_ARRAY:
			// build inner representation
			inner := build(sym.Element, sym.ElementSym, sym.Size)
			// repeat inner `sym.Size` times as inline members
			parts := make([]string, 0, sym.Size)
			for i := 0; i < sym.Size; i++ {
				parts = append(parts, inner)
			}
			return strings.Join(parts, ", ")
		default:
			return g.getNumberExprType(t)
		}
	}

	if field.Type == commons.TYPE_STRUCT {
		return build(field.Type, field.TypeSymbol, 0)
	}

	if field.Type == commons.TYPE_ARRAY {
		return build(commons.TYPE_ARRAY, &parser.Symbol{Type: commons.TYPE_ARRAY, Size: field.Size, Element: field.Element, ElementSym: field.ElementSym}, field.Size)
	}

	return build(field.Type, field.TypeSymbol, 0)
}

func (g *Generator) emitStructTypeDecls(syms []*parser.Symbol) {
	emitted := make(map[string]bool)
	visiting := make(map[string]bool)

	var emit func(sym *parser.Symbol)
	emit = func(sym *parser.Symbol) {
		if emitted[sym.Internal] {
			return
		}
		if visiting[sym.Internal] {
			panic(fmt.Sprintf("CODEGEN: recursive struct type %q", sym.Name))
		}
		visiting[sym.Internal] = true

		for _, field := range sym.Fields {
			if field.Type == commons.TYPE_STRUCT {
				emit(field.TypeSymbol)
			}
		}

		visiting[sym.Internal] = false
		emitted[sym.Internal] = true

		g.Code += fmt.Sprintf("type :%s = { ", sym.Internal)
		for _, field := range sym.Fields {
			g.Code += fmt.Sprintf("%s, ", g.qbeFieldTypeRef(field))
		}
		g.Code += "} # " + sym.Name + "\n"
	}

	for _, sym := range syms {
		emit(sym)
	}
}

func (g *Generator) emitTypeInfoTable(syms []*parser.Symbol) {
	// Minimal type info: emit data entries with struct sizes. This is a
	// placeholder for the full TypeInfo table requested in the review.
	for _, sym := range syms {
		size := g.getStructSize(sym)
		g.Code += fmt.Sprintf("data $ti_%s = { l %d } # typeinfo %s\n", sym.Internal, size, sym.Name)
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
			g.Code += "data $c_fmt = { b \"%c\\n\", b 0 }\n"
		}

		usedStrings := make(map[int]struct{})
		g.collectUsedStrings(expr, usedStrings)

		for id, str := range expr.Parser.Strings {
			if _, ok := usedStrings[id]; ok {
				g.Code += fmt.Sprintf("data $s_%d = { b \"%s\", b 0 }\n", id, str)
			}
		}

		structs := g.collectStructsFromContext(expr.Parser.CurrentContext)
		g.emitStructTypeDecls(structs)
		g.emitTypeInfoTable(structs)

		g.Code += "export function w $main() {\n@start\n"

		// Pre-allocate space for local variables at function prologue to
		// avoid emitting invalid alloc ops in nested contexts.
		g.preAllocateVars(expr)

		for _, child := range expr.Children {
			if g.GenerateStatement(child) {
				break
			}
		}

		g.Code += "}\n"
		return g.Code

	}

	panic(fmt.Sprintf("CODEGEN: unknown Kind: %q", expr.Kind.String()))
}
