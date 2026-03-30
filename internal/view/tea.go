package view

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strconv"
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

// ---------------- Bubble Tea "shopping style center"----------------
func NewStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

func ForegroundColorStyle(color string) lipgloss.Style {
	return NewStyle().Foreground(lipgloss.Color(color))
}

var (
	//style = lipgloss.NewStyle()

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

	textView = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	helpView = NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Render

	loaderSpinnerStyle   = ForegroundColorStyle("69")
	batchSpinnerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	currentFilenameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))

	doneStyle = lipgloss.NewStyle().Margin(1, 2)
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	errorMark = lipgloss.NewStyle().Foreground(lipgloss.Color("160")).SetString("✗")
)

// ---------------- Review Transformations Model ----------------
type ReviewTransformationsModel struct {
	viewport  viewport.Model
	Quit      bool
	Apply     bool
	Skip      bool
	Overwrite bool
}

func NewViewPortModel(content string) (*ReviewTransformationsModel, error) {

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

	return &ReviewTransformationsModel{
		viewport: vp,
	}, nil
}

func (m ReviewTransformationsModel) Init() tea.Cmd {
	return nil
}

func (m ReviewTransformationsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m ReviewTransformationsModel) View() tea.View {
	content := m.viewport.View() + helpView("\n  ↑/↓: Navigate • q/x: Quit • esc/n: Skip • enter/a: Apply • o/w: Overwrite • s: srt\n")
	v := tea.NewView(content)
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// ------------------------------------------------------------

// ---------------- Loader Model ----------------

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
	return tea.NewView(fmt.Sprintf("\n %s%s%s\n\n%s", m.spinner.View(), " ", textView(m.Message), helpView("q: exit\n")))
}

func NewLoaderModel() LoaderModel {
	m := LoaderModel{}
	m.spinner = spinner.New()
	m.spinner.Spinner = spinners[rand.Intn(len(spinners))] //nolint:gosec
	m.spinner.Style = loaderSpinnerStyle

	return m
}

// ------------------------------------------------------------

// ---------------- Batch Model ----------------
type BatchModel struct {
	files    []string
	count    int
	index    int
	width    int
	height   int
	spinner  spinner.Model
	progress progress.Model
	done     bool
}

func NewBatchModel(files []string) BatchModel {
	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	s := spinner.New()
	s.Style = batchSpinnerStyle
	s.Spinner = spinners[rand.Intn(len(spinners))] //nolint:gosec
	return BatchModel{
		files:    files,
		spinner:  s,
		progress: p,
	}
}

func (m BatchModel) Init() tea.Cmd {
	//return tea.Batch(extractSubtitles(m.files[m.index]), m.spinner.Tick)
	return tea.Sequence(m.spinner.Tick, extractSubtitles(m.files[m.index]))
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
	case ExtractedSubtitlesMsg:
		result := msg.String()
		//file := filepath.Base(m.files[m.index])
		tryCount := 0
		if countStr, _, found := strings.Cut(result, " "); found {
			tryCount, _ = strconv.Atoi(countStr)
			if tryCount > 0 {
				m.count += tryCount
			}
		}

		maxWidth := max(0, m.width-lipgloss.Width(fmt.Sprintf("%d %s  ", tryCount, checkMark)))
		if len(result) > maxWidth {
			result = result[:maxWidth]
		}
		finalMark := checkMark
		if tryCount <= 0 {
			finalMark = errorMark
			tryCount = 0

		}
		if m.index >= len(m.files)-1 {
			// Last file processed
			m.done = true
			return m, tea.Sequence(
				tea.Printf("%s %s", finalMark, result), // print the last success message
				tea.Quit,                               // exit the program
			)
		}

		// Update progress bar
		m.index++
		progressCmd := m.progress.SetPercent(float64(m.index) / float64(len(m.files)))

		return m, tea.Batch(
			progressCmd,
			tea.Printf("%s %s", finalMark, result), // print success message above our program
			extractSubtitles(m.files[m.index]),     // download the next package
		)
	case ErrorExtractingSubtitlesMsg:
		return m, tea.Quit
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

func pluralize(count int, singular string, plural string) string {
	// 0 is singular
	if math.Abs(float64(count)) > 1 {
		return plural
	}
	return singular
}

func (m BatchModel) View() tea.View {
	n := len(m.files)
	if m.done {
		return tea.NewView(doneStyle.Render(
			fmt.Sprintf("Done! %d %s extracted from %d %s.\n",
				m.count, pluralize(m.count, "subtitle", "subtitles"),
				n, pluralize(n, "file", "files"))))
	}

	w := lipgloss.Width(fmt.Sprintf("%d", n))

	//fileCount := fmt.Sprintf(" %*d/%*d", w, m.index, w, n)

	viewSpin := fmt.Sprintf("%s ", m.spinner.View())

	//prog := m.progress.View()

	viewProgressCountfile := fmt.Sprintf("%s %*d/%*d", m.progress.View(), w, m.index, w, n)

	tmpSpinProgCountfileWidth := fmt.Sprintf("%s%s", viewSpin, viewProgressCountfile)

	cellsCountWidth := max(0, m.width-lipgloss.Width(tmpSpinProgCountfileWidth))
	// shortName := strings.TrimSuffix(filepath.Base(m.files[m.index]), filepath.Ext(m.files[m.index]))
	//fileName := currentFilenameStyle.Render(shortName)

	viewInfo := lipgloss.NewStyle().MaxWidth(cellsCountWidth).Render(fmt.Sprintf("Extracting from %s", currentFilenameStyle.Render(filepath.Base(m.files[m.index]))))

	tmpContentForWidthRemaining := fmt.Sprintf("%s%s", tmpSpinProgCountfileWidth, viewInfo)

	cellsCountWidth = max(0, m.width-lipgloss.Width(tmpContentForWidthRemaining))

	return tea.NewView(fmt.Sprintf("%s%s%s%s", viewSpin, viewInfo, strings.Repeat(" ", cellsCountWidth), viewProgressCountfile))
}

type ExtractedSubtitlesMsg string

func (m ExtractedSubtitlesMsg) String() string {
	return string(m)
}

type ErrorExtractingSubtitlesMsg string

func (m ErrorExtractingSubtitlesMsg) String() string {
	return string(m)
}

func extractSubtitles(file string) tea.Cmd {
	d := time.Millisecond * time.Duration(200) //nolint:gosec
	return tea.Tick(d, func(t time.Time) tea.Msg {
		count, _ := mkv.BatchExtractSubtitles(file)
		// if err != nil {
		// 	time.Sleep(5 * time.Second)
		// 	//fmt.Println("Error extracting subtitles from MKV file", err)
		// 	return ErrorExtractingSubtitlesMsg(err.Error())
		// }
		return ExtractedSubtitlesMsg(fmt.Sprintf("%d %s", count, filepath.Base(file)))
	})
}
