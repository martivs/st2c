package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: st2c <file.st>")
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ST source loaded: %d bytes\n", len(data))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Print(string(data))
}
