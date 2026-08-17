package commons

import (
	"fmt"
	"path"
)

type ExprType int

const (
	TYPE_UNDEFINED ExprType = iota
	TYPE_U0
	TYPE_STRING
	TYPE_I64
	TYPE_F64
	TYPE_F32
	TYPE_I32
	TYPE_I8
	TYPE_U8
	TYPE_I16
	TYPE_U16
	TYPE_ARRAY
)

func (t ExprType) String() string {
	switch t {
	case TYPE_I64:
		return "i64"
	case TYPE_F64:
		return "f64"
	case TYPE_I32:
		return "i32"
	case TYPE_F32:
		return "f32"
	case TYPE_I8:
		return "i8"
	case TYPE_U8:
		return "u8"
	case TYPE_I16:
		return "i16"
	case TYPE_U16:
		return "u16"
	case TYPE_U0:
		return "u0"
	case TYPE_STRING:
		return "string"
	case TYPE_UNDEFINED:
		return "undefined"
	case TYPE_ARRAY:
		return "array"
	default:
		return "unknown"
	}
}

func CrashOut(msg string, file_path string, line int, col int) {
	fmt.Printf("%s:%d:%d: %s\n", file_path, line, col, msg)
}

func FileNameNoExt(filepath string) string {
	name := path.Base(filepath)
	ext := path.Ext(name)

	name = name[:len(name)-len(ext)]

	return name
}
