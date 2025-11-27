package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourname/subtitle-sanitizer/internal/model"
	"github.com/yourname/subtitle-sanitizer/internal/rules"
	"github.com/yourname/subtitle-sanitizer/internal/subtitle"
	"github.com/yourname/subtitle-sanitizer/internal/transform"
)

func main() {
	var inputPath string
	var encoding string
	var verbose bool
	var ignoreErrors bool

	flag.StringVar(&inputPath, "input", "", "Path to input subtitle file (.srt or .ass)")
	flag.StringVar(&encoding, "encoding", "utf-8", "Text encoding (currently uses utf-8)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose logging")
	flag.BoolVar(&ignoreErrors, "ignoreErrors", false, "Best-effort continue on minor errors")
	flag.Parse()

	if inputPath == "" {
		exitWithErr(errors.New("missing -input path"))
	}

	if verbose {
		fmt.Println("Input:", inputPath)
	}

	if err := validateInputPath(inputPath); err != nil {
		exitWithErr(err)
	}

	file, err := os.Open(inputPath)
	if err != nil {
		exitWithErr(fmt.Errorf("open file: %w", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		exitWithErr(fmt.Errorf("read file: %w", err))
	}

	ext := strings.ToLower(filepath.Ext(inputPath))
	var doc *model.Document
	switch ext {
	case ".srt":
		d, perr := subtitle.ParseSRT(data, ignoreErrors)
		if perr != nil {
			exitWithErr(perr)
		}
		doc = d
	case ".ass":
		d, perr := subtitle.ParseASS(data)
		if perr != nil {
			exitWithErr(perr)
		}
		doc = d
	default:
		exitWithErr(fmt.Errorf("unsupported extension: %s", ext))
	}

	conf := rules.LoadDefaultOrEmpty()
	// Current default built-in: remove uppercase words (2+ chars)
	conf.RemoveUppercaseColonWords = true

	result := transform.ApplyAll(*doc, conf)

	outPath := deriveOutputPath(inputPath)
	if verbose {
		fmt.Println("Output:", outPath)
	}

	outData := subtitle.FormatSRT(result) // Always save as .srt
	if err := os.WriteFile(outPath, outData, 0644); err != nil {
		exitWithErr(fmt.Errorf("write output: %w", err))
	}

	if verbose {
		fmt.Println("Done")
	}
}

func validateInputPath(p string) error {
	stat, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}
	if stat.IsDir() {
		return errors.New("input is a directory; expected a file")
	}
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".srt", ".ass":
		return nil
	default:
		return fmt.Errorf("unsupported extension: %s (only .srt, .ass)", ext)
	}
}

func deriveOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"-his.srt")
}

func exitWithErr(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
