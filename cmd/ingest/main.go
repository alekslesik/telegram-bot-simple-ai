package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	ingestservice "github.com/alekslesik/telegram-bot-simple/internal/service/ingest"
)

func main() {
	root := flag.String("root", "", "root directory with readable PDF files")
	flag.Parse()

	if strings.TrimSpace(*root) == "" {
		log.Fatal("flag -root is required")
	}

	svc := ingestservice.New()
	files, err := svc.PrepareInputs(*root)
	if err != nil {
		log.Fatalf("prepare inputs: %v", err)
	}

	fmt.Printf("would ingest %d file(s) from %s\n", len(files), *root)
	for _, file := range files {
		fmt.Printf("- %s\n", file.Path)
	}
}
