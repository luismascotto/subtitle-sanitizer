package model

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// --- Bubble Tea TUI ---
//type tickMsg struct{}

type UIModel struct {
	viewport  viewport.Model
	Quit      bool
	Apply     bool
	Skip      bool
	Overwrite bool
}

func NewModel(content string) (*UIModel, error) {

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
			m.Quit = true
			return m, tea.Quit
		case "n", "esc":
			m.Skip = true
			return m, tea.Quit
		case "s", "a", "enter":
			m.Apply = true
			return m, tea.Quit
		case "o", "w":
			m.Apply = true
			m.Overwrite = true
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
