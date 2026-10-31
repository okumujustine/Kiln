package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
)

const (
	statusBarLine = 1
)

// Editor model
type editor struct {
	lines    []string // the files contents
	top      int      // first visible line
	height   int      // terminal rows set on resize
	filename string
}

func (e editor) lastVisibleLine() int {
	actualHeight := e.height - statusBarLine
	windowEnd := (e.top + actualHeight) - 1 // subtract 1 as the line counting start at 0
	fileEnd := len(e.lines) - 1

	return min(windowEnd, fileEnd)
}

func (m editor) Init() tea.Cmd {
	return nil
}

// view

func (e editor) View() tea.View {
	return tea.NewView(fmt.Sprintf("The current count is: %v", e.lines))
}

// update
func (e editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "+":
			e.top++
		case "-":
			e.top--
		case "q", "ctrl+c":
			return e, tea.Quit
		}
	}
	return e, nil
}

func main() {

	if len(os.Args) <= 0 {
		log.Fatal("No filename provided")
		os.Exit(1)
	}
	filename := os.Args[1]

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error reading file %v: ", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println("New Line ::: ")
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error while scanning: %v", err)
		os.Exit(1)
	}

	app := tea.NewProgram(editor{top: 0})

	if _, err := app.Run(); err != nil {
		fmt.Printf("Getting started: %v", err)
	}

}
