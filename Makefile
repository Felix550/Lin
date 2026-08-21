lin: main.go parser/parser.go lexer/lexer.go commons/commons.go codegen/codegen.go runtime/runtime.c
	go build -ldflags "-s -w" -o lin

run: main.go parser/parser.go lexer/lexer.go commons/commons.go codegen/codegen.go runtime/runtime.c
	go run . main.lin && ./main