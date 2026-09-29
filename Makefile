ifeq ($(OS),Windows_NT)
    EXT := .exe
else
    EXT :=
endif

lin: main.go parser/parser.go lexer/lexer.go commons/commons.go codegen/codegen.go runtime/runtime.c
	go build -ldflags "-s -w" -o lin$(EXT)
