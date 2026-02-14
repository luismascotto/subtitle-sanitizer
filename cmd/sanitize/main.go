package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
	"github.com/luismascotto/subtitle-sanitizer/internal/subtitle"
	"github.com/luismascotto/subtitle-sanitizer/internal/transform"
)

func main() {
	var args struct {
		Input []string `arg:"positional"`
		// SrtBackup    bool     `arg:"-b,--srt-backup" help:"backup original srt files"`
		// SrtOverwrite bool     `arg:"-o,--srt-overwrite" help:"overwrite original srt files"`
		IgnoreErrors bool `arg:"-i,--ignore-errors" help:"ignore minor errors" default:"true"`
	}
	arg.MustParse(&args)

	conf := rules.LoadDefaultOrEmpty()
	if !conf.LoadedFromFile {
		conf.RemoveUppercaseColonWords = true
		conf.RemoveSingleLineColon = true
		conf.RemoveBetweenDelimiters = []rules.Delimiter{
			{Left: "(", Right: ")"},
			{Left: "[", Right: "]"},
			{Left: "{", Right: "}"},
			{Left: "*", Right: "*"},
		}
		conf.RemoveLineIfContains = " music *"
	}

	json, err := rules.MarshalIndentCompact(conf, "", "  ", 50)
	if err != nil {

		exitWithErr(fmt.Errorf("marshal rules: %w", err))
	}
	if !conf.LoadedFromFile {
		_ = conf.SaveToBackupFile(json)
	}

	for _, inputPath := range args.Input {

		// inputPath := args.Input[0]
		data := ReadFileContent(inputPath)

		ext := strings.ToLower(filepath.Ext(inputPath))
		var doc *model.Document
		var fromASS bool
		switch ext {
		case ".srt":
			d, perr := subtitle.ParseSRT(data, args.IgnoreErrors)
			if perr != nil {
				exitWithErr(perr)
			}
			doc = d
		case ".ass":
			fromASS = true
			d, perr := subtitle.ParseASS(data)
			if perr != nil {
				exitWithErr(perr)
			}
			doc = d
		default:
			exitWithErr(fmt.Errorf("unsupported extension: %s", ext))
		}

		result, retModel := RenderTransformations(json, inputPath, doc, conf, fromASS)
		retModelCheck, ok := retModel.(model.UIModel)
		if !ok {
			exitWithErr(errors.New("retModel is not of type UIModel"))
		}
		if retModelCheck.Quit {
			break
		}
		if retModelCheck.Skip {
			continue
		}
		ApplyTransformations(inputPath, retModelCheck, result, fromASS)

	}
}

func ReadFileContent(inputPath string) []byte {
	if inputPath == "" {
		exitWithErr(errors.New("missing input file(s)"))
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
	return data
}

func ApplyTransformations(inputPath string, retModelCheck model.UIModel, result model.Document, fromASS bool) {
	outPath := deriveOutputPath(inputPath, retModelCheck.Overwrite)

	outData := subtitle.FormatSRT(result) // Always save as .srt
	if err := os.WriteFile(outPath, outData, 0644); err != nil {
		exitWithErr(fmt.Errorf("write output: %w", err))
	}

	if fromASS && retModelCheck.Overwrite {
		_ = os.Remove(inputPath)
	}
}

func RenderTransformations(json []byte, inputPath string, doc *model.Document, conf rules.Config, fromASS bool) (model.Document, tea.Model) {
	sbContent := strings.Builder{}
	sbContent.WriteString("\n\n# Subtitle Sanitizer\n\n## Rules\n```json\n")
	sbContent.WriteString(string(json))
	sbContent.WriteString("\n```\n\n")

	sbContent.WriteString("## " + filepath.Base(inputPath) + "\n")

	sbTransformations := strings.Builder{}
	result := transform.ApplyAll(*doc, conf, fromASS, &sbTransformations)

	sbContent.WriteString("## Transformations\n")
	if sbTransformations.Len() > 0 {
		sbContent.WriteString("| Pos# | Original | Transformed/removed/empty | Rules |\n")
		sbContent.WriteString("| --- | --- | --- | --- |\n")
		sbContent.WriteString(sbTransformations.String())
	} else {
		sbContent.WriteString("Nothing to remove...\n")
	}

	vpModel, err := model.NewModel(sbContent.String())
	if err != nil {
		exitWithErr(fmt.Errorf("new model: %w", err))
	}

	retModel, err := tea.NewProgram(vpModel, tea.WithMouseAllMotion()).Run()
	if err != nil {
		exitWithErr(fmt.Errorf("run tea program: %w", err))
	}
	return result, retModel
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

func deriveOutputPath(inputPath string, overwrite bool) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	newName := filepath.Join(dir, name+".srt")
	if !FileExists(newName) || overwrite {
		// Happy path .ass to .srt
		return newName
	}

	newName = filepath.Join(dir, name+"-his.srt")
	if !FileExists(newName) {
		// Happy path .srt to -his.srt
		return newName
	}

	for range 5 {
		newName = filepath.Join(dir, name+"-his_"+strconv.FormatInt(int64(rand.Intn(1000)), 16)+".srt")
		if !FileExists(newName) {
			return newName
		}
	}
	exitWithErr(errors.New("failed to derive output path"))
	return ""
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func exitWithErr(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
