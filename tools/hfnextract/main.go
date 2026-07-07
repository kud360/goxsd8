package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/kud360/goxsd8/tools/hfnextract/internal"
)

func main() {
	filePath := flag.String("file", "", "Path to the markdown file")
	headerText := flag.String("header", "", "Search for table by header text (partial match)")
	tableIndex := flag.Int("index", -1, "Extract n-th table (0-indexed)")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -file <path> [-header <text>] [-index <n>]\n", os.Args[0])
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info("extracting tables", "file", *filePath)

	tables, err := internal.ExtractTables(*filePath, *headerText, *tableIndex)
	if err != nil {
		logger.Error("failed to extract tables", "error", err)
		os.Exit(1)
	}

	if len(tables) == 0 {
		logger.Warn("no matching tables found")
		os.Exit(0)
	}

	if err := internal.EncodeTables(os.Stdout, tables); err != nil {
		logger.Error("failed to encode json", "error", err)
		os.Exit(1)
	}
}
