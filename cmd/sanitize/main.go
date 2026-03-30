package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/alexflint/go-arg"

	"github.com/luismascotto/subtitle-sanitizer/internal/mkv"
	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
	"github.com/luismascotto/subtitle-sanitizer/internal/sanitize"
	"github.com/luismascotto/subtitle-sanitizer/internal/subtitle"
	"github.com/luismascotto/subtitle-sanitizer/internal/transform"
	"github.com/luismascotto/subtitle-sanitizer/internal/view"
)

func main() {
	var args struct {
		Input        []string `arg:"positional"`
		IgnoreErrors bool     `arg:"-i,--ignore-errors" help:"ignore minor errors" default:"true"`
		MkvExtract   bool     `arg:"-m,--mkv-extract" help:"extract subtitles from mkv files" default:"false"`
	}
	arg.MustParse(&args)

	normalizePwdPath()

	mkvDependenciesError := mkv.VerifyDependencies()

	if len(args.Input) == 0 {
		exitWithErr(errors.New("no input files provided"))
	}
	for _, inputPath := range args.Input {
		ext := strings.ToLower(filepath.Ext(inputPath))
		if ext == ".mkv" && mkvDependenciesError != nil {
			exitWithErr(mkvDependenciesError)
		}
	}

	conf := rules.LoadDefaultOrEmpty()

	rulesDisplay := conf.DescribeEffective()
	backupJSON, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		exitWithErr(fmt.Errorf("marshal config backup: %w", err))
	}
	if !conf.LoadedFromFile {
		_ = conf.SaveToBackupFile(backupJSON)
	}
	if args.MkvExtract {
		batchModel := view.NewBatchModel(args.Input)
		_, err := tea.NewProgram(batchModel).Run()
		if err != nil {
			exitWithErr(fmt.Errorf("run batch program: %w", err))
		}
		return
	}

	for _, inputPath := range args.Input {
		var data []byte
		var err error

		ext := strings.ToLower(filepath.Ext(inputPath))
		if ext == "" {
			exitWithErr(fmt.Errorf("extension is empty"))
		}

		if ext == ".mkv" {
			loader := tea.NewProgram(view.NewLoaderModel())

			go func() {
				loader.Send(view.LoaderMsg{Message: "Extracting subtitles from MKV file...", Quit: false})
				inputPath, data, err = mkv.Extract(inputPath)
				if err != nil {
					loader.Send(view.LoaderMsg{Message: "Error extracting subtitles from MKV file", Quit: false})
					time.Sleep(1 * time.Second)
					loader.Send(view.LoaderMsg{Message: "Error extracting subtitles from MKV file", Quit: true})
				} else {
					loader.Send(view.LoaderMsg{Message: "Subtitles extracted successfully", Quit: true})
				}
			}()

			if _, err := loader.Run(); err != nil {
				exitWithErr(fmt.Errorf("run loader program: %w", err))
			}

			if err != nil {
				exitWithErr(fmt.Errorf("extract mkv subtitles: %w", err))
			}

			ext = strings.ToLower(filepath.Ext(inputPath))
			if ext == "" {
				exitWithErr(fmt.Errorf("extension is empty"))
			}
		} else {
			data = ReadFileContent(inputPath)
		}

		if len(data) == 0 {
			exitWithErr(fmt.Errorf("data is empty"))
		}

		format := model.SubtitleFormatUnknown
		switch ext {
		case ".srt":
			format = model.SubtitleFormatSRT
		case ".ass":
			format = model.SubtitleFormatASS
		default:
			exitWithErr(fmt.Errorf("unsupported extension: %s", ext))
		}
		doc, err := subtitle.Parse(data, format)
		if err != nil {
			exitWithErr(err)
		}

		result, retModel := RenderTransformations(rulesDisplay, inputPath, doc, conf)
		retModelCheck, ok := retModel.(view.UIModel)
		if !ok {
			exitWithErr(errors.New("retModel is not of type UIModel"))
		}
		if retModelCheck.Quit {
			break
		}
		if retModelCheck.Skip {
			continue
		}
		var final *model.Document
		if retModelCheck.Apply {
			final = &result
		} else {
			final = doc
		}
		ApplyTransformations(inputPath, retModelCheck, final)
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

func ApplyTransformations(inputPath string, retModelCheck view.UIModel, result *model.Document) {
	if result.Format == model.SubtitleFormatSRT && retModelCheck.Overwrite && !retModelCheck.Apply {
		return
	}
	outPath := deriveOutputPath(inputPath, retModelCheck.Overwrite)

	outData := subtitle.FormatSRT(*result) // Always save as .srt
	if err := os.WriteFile(outPath, outData, 0644); err != nil {
		exitWithErr(fmt.Errorf("write output: %w", err))
	}

	if result.Format == model.SubtitleFormatASS && retModelCheck.Overwrite {
		_ = os.Remove(inputPath)
	}
}

func RenderTransformations(rulesDisplay string, inputPath string, doc *model.Document, conf rules.Config) (model.Document, tea.Model) {
	sbContent := strings.Builder{}
	sbContent.WriteString("\n\n# Subtitle Sanitizer\n\n## Active rules\n\n```\n")
	sbContent.WriteString(rulesDisplay)
	sbContent.WriteString("\n```\n\n")

	sbContent.WriteString("## " + filepath.Base(inputPath) + "\n")

	res := sanitize.Apply(*doc, conf)

	sbContent.WriteString("## Transformations\n")
	if len(res.Changes) > 0 {
		sbContent.WriteString("| Pos# | Original | Transformed/removed/empty | Rules |\n")
		sbContent.WriteString("| --- | --- | --- | --- |\n")
		sbContent.WriteString(transform.MarkdownRows(res.Changes))
	} else {
		sbContent.WriteString("Nothing to remove...\n")
	}

	vpModel, err := view.NewViewPortModel(sbContent.String())
	if err != nil {
		exitWithErr(fmt.Errorf("new model: %w", err))
	}

	retModel, err := tea.NewProgram(vpModel).Run()
	if err != nil {
		exitWithErr(fmt.Errorf("run tea program: %w", err))
	}
	return res.Document, retModel
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

func normalizePwdPath() {
	wd, err := os.Getwd()
	if err != nil {
		exitWithErr(fmt.Errorf("get working directory: %w", err))
	}
	//fmt.Printf("Current working directory: %s\n", wd)

	// 1. Get the absolute path of the running executable.
	executablePath, err := os.Executable()
	if err != nil {
		exitWithErr(fmt.Errorf("get executable path: %w", err))
	}

	// 2. Extract the directory part from the full path.
	// filepath.Dir is used for cross-platform compatibility.
	executableDir := filepath.Dir(executablePath)
	//fmt.Printf("Executable directory: %s\n", executableDir)

	if wd == executableDir {
		return
	}

	// 3. Change the current working directory to the executable's directory.
	if err := os.Chdir(executableDir); err != nil {
		exitWithErr(fmt.Errorf("change working directory: %w", err))
	}

	// newWd, err := os.Getwd()
	// if err != nil {
	// 	exitWithErr(fmt.Errorf("get working directory: %w", err))
	// }
	// fmt.Printf("New current working directory: %s\n", newWd)
}
