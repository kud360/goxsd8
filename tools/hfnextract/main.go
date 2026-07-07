package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/kud360/goxsd8/tools/hfnextract/builtins"
	"github.com/kud360/goxsd8/tools/hfnextract/internal"
)

const (
	structuresPath = "docs/specs/md/xmlschema11-2.md"
	precisionPath  = "docs/specs/md/xsd-precisionDecimal.md"
)

func main() {
	filePath := flag.String("file", "", "Path to the markdown file")
	headerText := flag.String("header", "", "Search for table by header text (partial match)")
	tableIndex := flag.Int("index", -1, "Extract n-th table (0-indexed)")
	builtinsMode := flag.Bool("builtins", false, "Emit the parsed builtin datatype table (from the two spec files) as JSON")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if *builtinsMode {
		if err := emitBuiltins(logger); err != nil {
			logger.Error("failed to parse builtins", "error", err)
			os.Exit(1)
		}
		return
	}

	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -file <path> [-header <text>] [-index <n>] | %s -builtins\n", os.Args[0], os.Args[0])
		os.Exit(1)
	}

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

func emitBuiltins(logger *slog.Logger) error {
	logger.Info("parsing builtins", "structures", structuresPath, "precision", precisionPath)
	types, err := builtins.Parse(structuresPath, precisionPath)
	if err != nil {
		return err
	}
	logger.Info("parsed builtins", "count", len(types))
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(types)
}
