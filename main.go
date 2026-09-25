package main

import (
	"bytes"
	"fmt"
	"lin/codegen"
	"lin/commons"
	"lin/lexer"
	"lin/parser"
	"os"
	"os/exec"
	fp "path/filepath"
	"runtime"
	"strings"

	flag "github.com/spf13/pflag"
)

func dumpIR(code string) {
	fmt.Println(code)
	os.Exit(0)
}

type BackendTarget int

const (
	TARGET_LINUX BackendTarget = iota
	TARGET_WINDOWS
)

func parseTarget(target string) BackendTarget {
	switch strings.ToLower(target) {
	case "windows", "w":
		return TARGET_WINDOWS
	case "linux", "l":
		return TARGET_LINUX
	default:
		fmt.Printf("target %s is not yet supported\n", target)
	}
	panic(1)
}

func main() {
	fmt.Println("Lin Copyright (C) 2026 Felice Di Lonardo\nThis program comes with ABSOLUTELY NO WARRANTY\nThis is free software, and you are welcome to redistribute it under certain conditions; Check 'LICENSE' for details.")

	flag.CommandLine.SetInterspersed(true)
	output := flag.StringP("output", "o", "", "output executable path")
	run := flag.BoolP("run", "r", false, "run after compilation")
	preserve_tmp := flag.BoolP("preserve", "p", false, "preserve the tmp folder")
	dump_backend := flag.BoolP("dump", "d", false, "dump the beckend code (qbe)")

	default_target := "l"
	if runtime.GOOS == "windows" {
		default_target = "w"
	}
	backend_target := flag.StringP("target", "t", default_target, "choose the backend target [linux(l)|windows(w)] - Default: linux(l)")

	flag.Parse()

	target := parseTarget(*backend_target)

	filepath := ""
	if !strings.Contains(os.Args[0], "debug") {
		if flag.NArg() != 1 {
			fmt.Printf("Usage:\n")
			fmt.Printf("  %s [options] file.lin\n\n", os.Args[0])
			flag.PrintDefaults()
			os.Exit(1)
		}
		filepath = flag.Arg(0)
	} else {
		filepath = "tests/main.lin"
	}

	if *output == "" {
		*output = commons.FileNameNoExt(filepath)
	}

	b, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	l := lexer.New(filepath, b)

	//for {
	//	kd := l.NextToken().Kind
	//	fmt.Println(kd.String())
	//	if kd == lexer.TOKEN_EOF{
	//		break
	//		os.Exit(1)
	//	}
	//}

	p := parser.New(l)

	root := p.Parse()

	generator := codegen.New()
	code := generator.Generate(root)

	if *dump_backend {
		dumpIR(code)
	}

	var cmd_qbe *exec.Cmd
	switch target {
	case TARGET_LINUX:
		cmd_qbe = exec.Command("qbe", "-t", "amd64_sysv")
	case TARGET_WINDOWS:
		cmd_qbe = exec.Command("qbe", "-t", "amd64_win")
	default:
		panic("UNREACHABLE")
	}

	cmd_qbe.Stdin = bytes.NewBufferString(code)
	cmd_qbe.Stderr = os.Stderr

	out, err := cmd_qbe.Output()
	if err != nil {
		panic(err)
	}

	tmp_dir, err := os.MkdirTemp("", "lin-*")
	if err != nil {
		panic(err)
	}

	s_file, err := os.CreateTemp(tmp_dir, "*.s")
	if err != nil {
		panic(err)
	}

	s_file.Write(out)
	s_file.Close()

	cmd_gcc := exec.Command("cc", s_file.Name(), "runtime/runtime.c", "-o", *output, "-lm")

	cmd_gcc.Stderr = os.Stderr

	if err := cmd_gcc.Run(); err != nil {
		panic(err)
	}

	fmt.Printf("%sOK:%s Compiled Successfully\n", commons.Green, commons.Reset)

	if !*preserve_tmp {
		os.Remove(tmp_dir)
	} else {
		fmt.Println("TMP Build Dir:", tmp_dir)
	}

	if *run {
		abs, err := fp.Abs(*output)
		if err != nil {
			panic(err)
		}
		cmd := exec.Command(abs)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Println("Error running: ", err)
		}
	}
}
