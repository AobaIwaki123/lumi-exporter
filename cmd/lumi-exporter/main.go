package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/AobaIwaki123/lumi-exporter/internal/matching"
)

func main() {
	inputPath := flag.String("input", "testdata/evaluation.json", "評価fixtureのJSONパス")
	pretty := flag.Bool("pretty", true, "JSONをインデントして出力する")
	flag.Parse()

	if err := run(*inputPath, *pretty); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath string, pretty bool) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open fixture: %w", err)
	}
	defer func() { _ = file.Close() }()

	fixture, err := matching.LoadFixture(file)
	if err != nil {
		return err
	}
	report := matching.Evaluate(fixture, matching.DefaultConfig())

	encoder := json.NewEncoder(os.Stdout)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(report)
}
