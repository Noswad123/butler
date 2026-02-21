package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styling
var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Padding(0, 1)
	docStyle    = lipgloss.NewStyle().Margin(1, 2)
	listStyle   = lipgloss.NewStyle().Width(40).Border(lipgloss.NormalBorder(), false, true, false, false)
	prevStyle   = lipgloss.NewStyle().Padding(0, 2)
)

type item struct {
	name, description, path, preview string
	line                             int
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.name + " " + i.description }

type model struct {
	list     list.Model
	viewport viewport.Model
	choice   string
	quitting bool
	ready    bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = fmt.Sprintf("%s:%d", i.path, i.line)
			}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		listWidth := 40
		m.list.SetSize(listWidth, msg.Height-v)
		
		if !m.ready {
			m.viewport = viewport.New(msg.Width-listWidth-h-4, msg.Height-v)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - listWidth - h - 4
			m.viewport.Height = msg.Height - v
		}
	}

	// Update List
	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	cmds = append(cmds, listCmd)

	// Update Preview Content based on selection
	if i, ok := m.list.SelectedItem().(item); ok {
		m.viewport.SetContent(i.preview)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.quitting { return "" }
	if !m.ready { return "\n  Initializing Butler..." }

	return docStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			listStyle.Render(m.list.View()),
			prevStyle.Render(m.viewport.View()),
		),
	)
}

func main() {
	home, _ := os.UserHomeDir()
	// Explicitly check the path. If your dotfiles are in a different spot, 
	// use an env var or a hardcoded path.
	dotfiles := filepath.Join(home, ".dotfiles")
	
	var items []list.Item

	// Walk through the directories you listed earlier
	err := filepath.Walk(dotfiles, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() { return nil }
		
		// Only scan .zsh, .sh files
		ext := filepath.Ext(path)
		if ext == ".zsh" || ext == ".sh" {
			file, _ := os.Open(path)
			defer file.Close()

			var (
				scanner = bufio.NewScanner(file)
				lineNum = 0
				currentName string
				lines []string
			)

			// Read whole file for preview logic
			var fileContent []string
			for scanner.Scan() {
				fileContent = append(fileContent, scanner.Text())
			}

			for i, line := range fileContent {
				lineNum = i + 1
				if strings.Contains(line, "@name:") {
					currentName = strings.TrimSpace(strings.Split(line, "@name:")[1])
				} else if strings.Contains(line, "@description:") && currentName != "" {
					desc := strings.TrimSpace(strings.Split(line, "@description:")[1])
					
					// Build a 20-line preview window starting from the @name tag
					end := i + 20
					if end > len(fileContent) { end = len(fileContent) }
					preview := strings.Join(fileContent[i-1:end], "\n")

					items = append(items, item{
						name:        currentName,
						description: desc,
						path:        path,
						line:        i, // The line number for nvim
						preview:     preview,
					})
					currentName = ""
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking path: %v\n", err)
		os.Exit(1)
	}

	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "BUTLER"
	m.list.SetShowStatusBar(false)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	choice := finalModel.(model).choice
	if choice != "" {
		fmt.Print(choice) // Output to stdout for the shell wrapper
	}
}
