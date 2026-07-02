package main

import (
	"fmt"
	"log"
	"os"
	"st2c/src/lexer"
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

	l := lexer.New(string(data))
	for {
		tok := l.NextToken()
		fmt.Printf("line %2d  %-12s %q\n", tok.Line, tok.Type, tok.Literal)
		if tok.Type == lexer.EOF {
			break
		}
	}
}
