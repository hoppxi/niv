package search

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func handleScriptExtension(ext ExtConfig, query string) []Result {
	path := expandHome(ext.Path)

	// Auto-fix permissions if script isn't executable
	info, err := os.Stat(path)
	if err == nil && info.Mode()&0111 == 0 {
		os.Chmod(path, 0755)
	}

	cmd := exec.Command(path, query)
	out, err := cmd.Output()
	if err != nil {
		return []Result{{Name: "Execution Error", Source: ext.Name, Command: "echo " + err.Error()}}
	}

	var results []Result
	if err := json.Unmarshal(out, &results); err != nil {
		return []Result{{Name: "Invalid JSON from Extension", Source: ext.Name}}
	}
	return results
}

func handleFolderExtension(ext ExtConfig, query string) []Result {
	dirPath := expandHome(ext.FromFolder)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return []Result{{Name: "Folder Error", Source: ext.Name, Command: "echo folder not found"}}
	}

	var results []Result
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Skip excluded file extensions
		if ext.ExcludeType != "" && strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext.ExcludeType)) {
			continue
		}

		// Basic filter for the file list
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			continue
		}

		fullPath := filepath.Join(dirPath, name)
		results = append(results, Result{
			Name:    name,
			GUI:     true,
			Type:    "file",
			Source:  ext.Name,
			Command: strings.ReplaceAll(ext.OnSelect, "{}", fullPath),
			Comment: "extension: " + ext.Name,
			Icon:    fullPath,
		})
	}
	return results
}
