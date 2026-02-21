package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

type Item struct {
	Name, Summary, Path, Preview string
	Line                             int
}

func (i Item) Title() string       { return i.Name }
func (i Item) Description() string { return i.Summary }
func (i Item) FilterValue() string { return i.Name + " " + i.Summary }

func ScanDotfiles(root string) ([]list.Item, error) {
	var items []list.Item

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() { return nil }
		
		ext := filepath.Ext(path)
		if ext != ".zsh" && ext != ".sh" && ext != ".lua" { return nil }

		file, _ := os.Open(path)
		defer file.Close()

		var fileContent []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fileContent = append(fileContent, scanner.Text())
		}

		var currentName string
		var nameLine int

		for i, line := range fileContent {
			cleanLine := strings.TrimSpace(line)
			
			if strings.Contains(cleanLine, "@name:") {
				currentName = strings.TrimSpace(strings.Split(cleanLine, "@name:")[1])
				nameLine = i
				continue
			}

			if strings.Contains(cleanLine, "@description:") && currentName != "" {
				desc := strings.TrimSpace(strings.Split(cleanLine, "@description:")[1])
				
				// Smart Preview Logic
				endIdx := i + 4
				for j := i + 1; j < len(fileContent); j++ {
					if strings.Contains(fileContent[j], "@end") {
						endIdx = j - 1
						break
					}
					if strings.Contains(fileContent[j], "@name:") { break }
				}

				if endIdx >= len(fileContent) { endIdx = len(fileContent) - 1 }
				
				items = append(items, Item{
					Name:        currentName,
					Summary: desc,
					Path:        path,
					Line:        nameLine + 1,
					Preview:     strings.Join(fileContent[nameLine:endIdx+1], "\n"),
				})
				currentName = "" 
			}
		}
		return nil
	})

	return items, err
}
