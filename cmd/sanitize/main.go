package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	var inputPath string
	var encoding string
	// var verbose bool
	var ignoreErrors bool

	flag.StringVar(&inputPath, "input", "", "Path to input subtitle file (.srt or .ass)")
	flag.StringVar(&encoding, "encoding", "utf-8", "Text encoding (currently uses utf-8)")
	// flag.BoolVar(&verbose, "verbose", false, "Verbose logging")
	flag.BoolVar(&ignoreErrors, "ignoreErrors", false, "Best-effort continue on minor errors")
	flag.Parse()

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
		d, perr := subtitle.ParseSRT(data, ignoreErrors)
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

	json, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		exitWithErr(fmt.Errorf("marshal rules: %w", err))
	}

	sbContent := strings.Builder{}
	sbContent.WriteString("# Subtitle Sanitizer\n\n## Rules\n```json\n")
	sbContent.WriteString(string(json))
	sbContent.WriteString("\n```\n\n")

	sbTransformations := strings.Builder{}
	result := transform.ApplyAll(*doc, conf, fromASS, &sbTransformations)

	sbContent.WriteString("## Transformations\n")
	if sbTransformations.Len() > 0 {
		sbContent.WriteString("| Line# | Original | Transformed | Rules Applied |\n")
		sbContent.WriteString("| --- | --- | --- | --- |\n")
		sbContent.WriteString(sbTransformations.String())
	} else {
		sbContent.WriteString("Nothing to remove...\n")
	}

	vpModel, err := newModel(sbContent.String())
	if err != nil {
		exitWithErr(fmt.Errorf("new model: %w", err))
	}

	if _, err := tea.NewProgram(vpModel).Run(); err != nil {
		exitWithErr(fmt.Errorf("run tea program: %w", err))
	}

	outPath := deriveOutputPath(inputPath)
	// if verbose {
	// 	fmt.Println("Output:", outPath)
	// }

	outData := subtitle.FormatSRT(result) // Always save as .srt
	if err := os.WriteFile(outPath, outData, 0644); err != nil {
		exitWithErr(fmt.Errorf("write output: %w", err))
	}

	// if verbose {
	// 	fmt.Println("Done")
	// }
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
	//name := strings.TrimSuffix(base, filepath.Ext(base))
	newName := filepath.Join(dir, base+"-his.srt")
	if _, err := os.Stat(newName); err != nil && os.IsNotExist(err) {
		// Happy path
		return newName
	}
	return filepath.Join(dir, base+"-his_"+strconv.FormatInt(int64(rand.Intn(1000)), 16)+".srt")
}

func exitWithErr(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

// --- Bubble Tea TUI ---
type tickMsg struct{}

type UIModel struct {
	viewport viewport.Model
}

func newModel(content string) (*UIModel, error) {

	const width = 120

	vp := viewport.New(width, 40)
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "q", "esc", "ctrl+c":
			//m.shouldQuit = true
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m UIModel) View() string {
	return m.viewport.View() + helpView.Render("\n  ↑/↓: Navigate • q: Quit\n")
}

type colorTheme struct {
	fg string
	bg string
}

var colorThemes = []colorTheme{
	{fg: "#FFD166", bg: "#073B4C"}, // golden on deep teal
	{fg: "#06D6A0", bg: "#1B1F3B"}, // mint on midnight
	{fg: "#EF476F", bg: "#2F2E41"}, // pink on ink
	{fg: "#A78BFA", bg: "#111827"}, // violet on near-black
	{fg: "#F59E0B", bg: "#0F172A"}, // amber on slate
}
var (
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#93C5FD")).Background(lipgloss.Color("#1F2937")).Bold(true).Padding(0, 1)
	dividerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
	timeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	messageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	secondaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	helpView       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true)
)
