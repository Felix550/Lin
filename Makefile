lin: main.go parser/parser.go lexer/lexer.go commons/commons.go codegen/codegen.go runtime/runtime.o
	go build -ldflags "-s -w" -o lin

run: main.go parser/parser.go lexer/lexer.go commons/commons.go codegen/codegen.go runtime/runtime.o
	go run . main.lin && ./main

runtime/runtime.o: runtime/runtime.c
	cc -c -o runtime/runtime.o runtime/runtime.c