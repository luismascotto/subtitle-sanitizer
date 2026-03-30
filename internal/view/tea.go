package view

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/luismascotto/subtitle-sanitizer/internal/mkv"
)

// --- Bubble Tea TUI ---

type UIModel struct {
	viewport  viewport.Model
	Quit      bool
	Apply     bool
	Skip      bool
	Overwrite bool
}

func NewViewPortModel(content string) (*UIModel, error) {

	const width = 100

	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(32))
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
	return nil
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {

		case "q", "x", "ctrl+c":
			m.Quit = true
			return m, tea.Quit
		case "n", "esc":
			m.Skip = true
			return m, tea.Quit
		case "a", "enter":
			m.Apply = true
			return m, tea.Quit
		case "o", "w":
			m.Apply = true
			m.Overwrite = true
			return m, tea.Quit
		case "s":
			m.Overwrite = true
			return m, tea.Quit
		default:
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m UIModel) View() tea.View {
	content := m.viewport.View() + helpView("\n  ↑/↓: Navigate • q/x: Quit • esc/n: Skip • enter/a: Apply • o/w: Overwrite • s: srt\n")
	v := tea.NewView(content)
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

var (
	spinners = []spinner.Spinner{
		spinner.Line,
		spinner.Dot,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Globe,
		spinner.Moon,
		spinner.Monkey,
	}

	textStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	spinnerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	helpView             = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Render
	currentFilenameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle            = lipgloss.NewStyle().Margin(1, 2)
	checkMark            = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

type LoaderModel struct {
	spinner spinner.Model
	index   int
	Message string
}

type LoaderMsg struct {
	Message string
	Quit    bool
}

func (m LoaderMsg) String() string {
	return m.Message
}

func (m LoaderModel) Init() tea.Cmd {
	return func() tea.Msg {
		return m.spinner.Tick()
	}
}
func (m LoaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		default:
			return m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case LoaderMsg:
		m.Message = msg.Message
		if msg.Quit {
			return m, tea.Quit
		}
		return m, nil
	default:
		return m, nil
	}
}
func (m LoaderModel) View() tea.View {
	var s string
	s += fmt.Sprintf("\n %s%s%s\n\n", m.spinner.View(), " ", textStyle(m.Message))
	s += helpView("q: exit\n")
	return tea.NewView(s)
}

func NewLoaderModel() *LoaderModel {
	m := &LoaderModel{}
	m.index = 0
	m.spinner = spinner.New(
		spinner.WithStyle(spinnerStyle),
		spinner.WithSpinner(spinners[m.index]),
	)
	return m
}
func (m *LoaderModel) NextSpinner() {
	m.index++
	if m.index >= len(spinners) {
		m.index = 0
	}
	m.spinner.Spinner = spinners[m.index]
}
func (m *LoaderModel) PreviousSpinner() {
	m.index--
	if m.index < 0 {
		m.index = len(spinners) - 1
	}
	m.spinner.Spinner = spinners[m.index]
}

type BatchModel struct {
	files    []string
	index    int
	width    int
	height   int
	spinner  spinner.Model
	progress progress.Model
	done     bool
}

func NewBatchModel(files []string) BatchModel {
	p := progress.New(
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	s := spinner.New(spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))))
	return BatchModel{
		files:    files,
		spinner:  s,
		progress: p,
	}
}

func (m BatchModel) Init() tea.Cmd {
	return tea.Batch(
		extractSubtitles(m.files[m.index]),
		func() tea.Msg {
			return m.spinner.Tick()
		},
	)
}

func (m BatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	case extractedSubtitlesMsg:
		file := m.files[m.index]
		if m.index >= len(m.files)-1 {
			// Everything's been installed. We're done!
			m.done = true
			return m, tea.Sequence(
				tea.Printf("%s %s", checkMark, file), // print the last success message
				tea.Quit,                             // exit the program
			)
		}

		// Update progress bar
		m.index++
		progressCmd := m.progress.SetPercent(float64(m.index) / float64(len(m.files)))

		return m, tea.Batch(
			progressCmd,
			tea.Printf("%s %s", checkMark, file), // print success message above our program
			extractSubtitles(m.files[m.index]),   // download the next package
		)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m BatchModel) View() tea.View {
	n := len(m.files)
	w := lipgloss.Width(fmt.Sprintf("%d", n))

	if m.done {
		return tea.NewView(doneStyle.Render(fmt.Sprintf("Done! Extracted subtitles from %d files.\n", n)))
	}

	fileCount := fmt.Sprintf(" %*d/%*d", w, m.index, w, n)

	spin := m.spinner.View() + " "
	prog := m.progress.View()
	cellsAvail := max(0, m.width-lipgloss.Width(spin+prog+fileCount))

	fileName := currentFilenameStyle.Render(m.files[m.index])
	info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render("Extracting subtitles from " + fileName)

	cellsRemaining := max(0, m.width-lipgloss.Width(spin+info+prog+fileCount))
	gap := strings.Repeat(" ", cellsRemaining)

	return tea.NewView(spin + info + gap + prog + fileCount)
}

type extractedSubtitlesMsg string

func extractSubtitles(file string) tea.Cmd {
	_, err := mkv.ExtractAll(file, 0)
	if err != nil {
		return tea.Printf("Error extracting subtitles from %s: %s", file, err)
	}
	d := time.Millisecond * time.Duration(100) //nolint:gosec
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return extractedSubtitlesMsg(file)
	})
}
