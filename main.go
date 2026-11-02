package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/okumujustine/kiln/internal/buffer"
)

// The status bar takes up one row at the bottom of the screen.
const statusBarHeight = 1

// editor holds everything the screen needs to draw itself.
type editor struct {
	doc              *buffer.PieceTable // the file's contents
	firstVisibleLine int                // index of the line shown at the top of the screen
	terminalHeight   int                // total rows in the terminal, updated on resize
	terminalWidth    int
	filename         string // shown in the status bar
}

// textAreaHeight is how many rows are left for file text
// after reserving space for the status bar.
func (e editor) textAreaHeight() int {
	return e.terminalHeight - statusBarHeight
}

// lastVisibleLine is the index of the last line on screen.
// It stops at whichever comes first: the bottom of the screen
// or the end of the file.
func (e editor) lastVisibleLine() int {
	bottomOfScreen := e.firstVisibleLine + e.textAreaHeight() - 1 // -1 because indexes start at 0
	lastLineOfFile := e.doc.LineCount() - 1                       // -1 because indexes start at 0
	return min(bottomOfScreen, lastLineOfFile)
}

// maxFirstVisibleLine is the furthest we can scroll down while
// keeping the screen full. It is 0 when the whole file fits.
func (e editor) maxFirstVisibleLine() int {
	return max(0, e.doc.LineCount()-e.textAreaHeight())
}

func (e editor) Init() tea.Cmd {
	return nil
}

func (e editor) View() tea.View {
	var screen strings.Builder
	lastLine := e.lastVisibleLine()

	for lineIndex := e.firstVisibleLine; lineIndex <= lastLine; lineIndex++ {
		lineNumber := lineIndex + 1 // people count lines from 1
		fmt.Fprintf(&screen, "%4d  %s\n", lineNumber, e.doc.Line(lineIndex))
	}

	textArea := lipgloss.NewStyle().Height(e.textAreaHeight())
	text := textArea.Render(strings.TrimSuffix(screen.String(), "\n")) // trim last line which could add more spacing

	statusBarArea := lipgloss.NewStyle().Height(statusBarHeight).Width(e.terminalWidth).Background(lipgloss.Color("#1b1f29")).Foreground(lipgloss.Color("#e5a54b"))

	visibleLineNumber := e.firstVisibleLine + 1
	lineLength := e.doc.LineCount()
	if lineLength <= 0 {
		visibleLineNumber = 0
	}
	status := statusBarArea.Render(fmt.Sprintf("%s          line %d of %d", e.filename, visibleLineNumber, lineLength))

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, text, status))
}

func (e editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		e.terminalHeight = msg.Height
		e.terminalWidth = msg.Width
		e.firstVisibleLine = min(e.maxFirstVisibleLine(), e.firstVisibleLine)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if e.firstVisibleLine < e.maxFirstVisibleLine() {
				e.firstVisibleLine++
			}
		case "k", "up":
			if e.firstVisibleLine > 0 {
				e.firstVisibleLine--
			}
		case "q", "ctrl+c":
			return e, tea.Quit
		}
	}
	return e, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("usage: kiln <filename>")
	}
	filename := os.Args[1]

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	app := tea.NewProgram(editor{doc: buffer.New(data), filename: filename})
	if _, err := app.Run(); err != nil {
		log.Fatalln(err)
	}
}
