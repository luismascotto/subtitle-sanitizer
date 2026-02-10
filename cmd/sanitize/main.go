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

	"github.com/alexflint/go-arg"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

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
	// Current default built-in: remove uppercase words (2+ chars)
	conf.RemoveUppercaseColonWords = true
	conf.RemoveBetweenDelimiters = []rules.Delimiter{
		{Left: "(", Right: ")"},
		{Left: "[", Right: "]"},
		{Left: "{", Right: "}"},
		//{Left: "?", Right: "?"},
		//{Left: "<", Right: ">"},
		{Left: "¶", Right: "¶"},
		{Left: "♪", Right: "♪"},
		{Left: "♫", Right: "♫"},
		{Left: "♬", Right: "♬"},
		{Left: "♭", Right: "♭"},
		{Left: "*", Right: "*"},
	}
	conf.RemoveLineIfContains = " music *"

	json, err := json.Marshal(conf)
	//json, err := colorjson.Marshal(conf)
	if err != nil {
		exitWithErr(fmt.Errorf("marshal rules: %w", err))
	}

	for _, inputPath := range args.Input {

		// inputPath := args.Input[0]
		if inputPath == "" {
			exitWithErr(errors.New("missing -input path"))
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

		vpModel, err := newModel(sbContent.String())
		if err != nil {
			exitWithErr(fmt.Errorf("new model: %w", err))
		}

		retModel, err := tea.NewProgram(vpModel, tea.WithMouseAllMotion()).Run()
		if err != nil {
			exitWithErr(fmt.Errorf("run tea program: %w", err))
		}
		retModelCheck, ok := retModel.(UIModel)
		if !ok {
			exitWithErr(errors.New("retModel is not of type UIModel"))
		}
		if retModelCheck.skip {
			continue
		}
		outPath := deriveOutputPath(inputPath, retModelCheck.overwrite)

		outData := subtitle.FormatSRT(result) // Always save as .srt
		if err := os.WriteFile(outPath, outData, 0644); err != nil {
			exitWithErr(fmt.Errorf("write output: %w", err))
		}

		if fromASS && retModelCheck.overwrite {
			_ = os.Remove(inputPath)
		}

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

// --- Bubble Tea TUI ---
//type tickMsg struct{}

type UIModel struct {
	viewport  viewport.Model
	quit      bool
	apply     bool
	skip      bool
	overwrite bool
}

func newModel(content string) (*UIModel, error) {

	const width = 100

	vp := viewport.New(width, 32)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	// We need to adjust the width of the glamour render from our main width
	// to account for a few things:
	//
	//  * The viewport border width
	//  * The viewport padding
	//  * The viewport margins
	//  * The gutter glamour applies to the left side of the content
	//
	const glamourGutter = 2
	glamourRenderWidth := width - vp.Style.GetHorizontalFrameSize() - glamourGutter

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(glamourRenderWidth),
	)
	if err != nil {
		return nil, err
	}

	str, err := renderer.Render(content)
	if err != nil {
		return nil, err
	}

	vp.SetContent(str)

	return &UIModel{
		viewport: vp,
	}, nil
}

func (m UIModel) Init() tea.Cmd {
	//return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })

	return nil
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Pass mouse events to the viewport component
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {

		case "q", "x", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "n", "esc":
			m.skip = true
			return m, tea.Quit
		case "s", "a", "enter":
			m.apply = true
			return m, tea.Quit
		case "o", "w":
			m.apply = true
			m.overwrite = true
			return m, tea.Quit
		default:
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m UIModel) View() string {
	return m.viewport.View() + helpView.Render("\n  ↑/↓: Navigate • q/x: Quit • esc/n: Skip • s/a: Apply • o/w: Overwrite\n")
}

// type colorTheme struct {
// 	fg string
// 	bg string
// }

//	var colorThemes = []colorTheme{
//		{fg: "#FFD166", bg: "#073B4C"}, // golden on deep teal
//		{fg: "#06D6A0", bg: "#1B1F3B"}, // mint on midnight
//		{fg: "#EF476F", bg: "#2F2E41"}, // pink on ink
//		{fg: "#A78BFA", bg: "#111827"}, // violet on near-black
//		{fg: "#F59E0B", bg: "#0F172A"}, // amber on slate
//	}
var (
	// headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#93C5FD")).Background(lipgloss.Color("#1F2937")).Bold(true).Padding(0, 1)
	// dividerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
	// timeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	// messageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	// secondaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	helpView = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true)
)
