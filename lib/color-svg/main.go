/*
 A simple Go program that changes the stroke color of all SVG files in the given folder to a specified new color.
 Usage: go run main.go <folder-path> <new-color>
 Intended for use to change SVG stroke colors in bulk for icons and vectors.
*/

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: color-svg <folder-path> <new-color>")
		return
	}

	folder := os.Args[1]
	newColor := os.Args[2]

	entries, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		return
	}

	// Regex to find stroke="someColor"
	re := regexp.MustCompile(`stroke="[^"]*"`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".svg" {
			continue
		}

		fullPath := filepath.Join(folder, entry.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", fullPath, err)
			continue
		}

		// Replace all stroke="..." with stroke="<newColor>"
		newContent := re.ReplaceAllStringFunc(string(content), func(s string) string {
			return `stroke="` + newColor + `"`
		})

		err = os.WriteFile(fullPath, []byte(newContent), fs.FileMode(0644))
		if err != nil {
			fmt.Printf("Failed to write file %s: %v\n", fullPath, err)
			continue
		}

		fmt.Printf("Updated %s\n", fullPath)
	}
}
