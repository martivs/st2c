package main

import (
	"fmt"
	"log"
	"os"

	"st2c/src/lexer"
	"st2c/src/parser"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: st2c <file.st>")
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ST source loaded: %d bytes\n\n", len(data))

	// Этап 2: лексер → парсер → печать AST.
	// Лексер одноразовый (pull-модель), поэтому вывод потока токенов и
	// разбор в дерево — взаимоисключающие. Печать токенов см. в git-истории
	// Этапа 1.
	p := parser.New(lexer.New(string(data)))
	sf, err := p.ParseSourceFile()
	if err != nil {
		log.Fatal(err)
	}

	for _, pou := range sf.POUs {
		fmt.Print(pou.String())
	}
}
