package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// The status bar takes up one row at the bottom of the screen.
const statusBarHeight = 1

// editor holds everything the screen needs to draw itself.
type editor struct {
	lines            []string // the file's contents, one string per line
	firstVisibleLine int      // index of the line shown at the top of the screen
	terminalHeight   int      // total rows in the terminal, updated on resize
	filename         string   // shown in the status bar
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
	lastLineOfFile := len(e.lines) - 1                            // -1 because indexes start at 0
	return min(bottomOfScreen, lastLineOfFile)
}

// maxFirstVisibleLine is the furthest we can scroll down while
// keeping the screen full. It is 0 when the whole file fits.
func (e editor) maxFirstVisibleLine() int {
	return max(0, len(e.lines)-e.textAreaHeight())
}

func (e editor) Init() tea.Cmd {
	return nil
}

func (e editor) View() tea.View {
	var screen strings.Builder
	lastLine := e.lastVisibleLine()

	for lineIndex := e.firstVisibleLine; lineIndex <= lastLine; lineIndex++ {
		lineNumber := lineIndex + 1 // people count lines from 1
		fmt.Fprintf(&screen, "%4d  %s\n", lineNumber, e.lines[lineIndex])
	}

	textArea := lipgloss.NewStyle().Height(e.textAreaHeight())
	return tea.NewView(textArea.Render(screen.String()))
}

func (e editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		e.terminalHeight = msg.Height

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

// readLines loads a file and returns its contents as a slice of lines.
func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close() // runs when readLines returns, so the file is closed right after reading

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("usage: kiln <filename>")
	}
	filename := os.Args[1]

	lines, err := readLines(filename)
	if err != nil {
		log.Fatalln(err)
	}

	app := tea.NewProgram(editor{lines: lines, filename: filename})
	if _, err := app.Run(); err != nil {
		log.Fatalln(err)
	}
}
