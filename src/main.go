package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"st2c/src/codegen"
	"st2c/src/lexer"
	"st2c/src/parser"
	"st2c/src/sema"
)

func main() {
	log.SetFlags(0)
	out := flag.String("o", "", "файл C-вывода (по умолчанию stdout)")
	withMain := flag.Bool("main", false, "дописать драйвер main() — файл самодостаточен для gcc")
	scans := flag.Int("scans", 1, "сколько раз драйвер вызывает _step (только с -main)")
	dumpAST := flag.Bool("dump-ast", false, "печатать AST вместо генерации C")
	flag.Parse()
	if flag.NArg() > 1 {
		log.Fatal("usage: st2c [-o out.c] [-main] [-scans N] [-dump-ast] [file.st]")
	}

	// Основной способ получить исходник — stdin: так его вызывает
	// tools/tester (пишет ST в stdin процесса и закрывает канал, ждёт C на
	// stdout). Позиционный аргумент file.st — вспомогательный, для ручного
	// запуска и отладки (go run ./src examples/example.st и т.п.).
	var data []byte
	var err error
	if flag.NArg() == 1 {
		data, err = os.ReadFile(flag.Arg(0))
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Конвейер: лексер → парсер → sema → codegen. Лексер одноразовый
	// (pull-модель); печать потока токенов Этапа 1 — в git-истории.
	p := parser.New(lexer.New(string(data)))
	sf, err := p.ParseSourceFile()
	if err != nil {
		log.Fatal(err)
	}

	// Семантический анализ: в отличие от fail-fast парсера печатаются все
	// найденные ошибки за один запуск. Side-table типов (sema.Info) уходит
	// в codegen: по ней эмитятся адаптивные литералы и хелперы конверсий.
	info, errs := sema.Check(sf)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		os.Exit(1)
	}

	if *dumpAST {
		fmt.Printf("ST source loaded: %d bytes\n\n", len(data))
		for _, pou := range sf.POUs {
			fmt.Print(pou.String())
		}
		return
	}

	code, err := codegen.Generate(sf, info, codegen.Options{Main: *withMain, Scans: *scans})
	if err != nil {
		log.Fatal(err)
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(code), 0o644); err != nil {
			log.Fatal(err)
		}
		return
	}
	fmt.Print(code)
}
