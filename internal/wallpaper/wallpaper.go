package wallpaper

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	WALLPAPERS_PATH_TMP = "/tmp/niv_wallpapers_path"
	WALLPAPER_TMP       = "/tmp/niv_wallpaper"
	WALLPAPERS_PATH     = "WALLPAPERS_PATH"
	WALLPAPER           = "WALLPAPER"
	compactArraySize    = 5
)

func getEwwCurrentValue() (string, error) {
	cmd := exec.Command("eww", "get", WALLPAPERS_PATH)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("Warning: 'eww get %s' failed (Exit Code %d). Assuming empty wallpaper array.", WALLPAPERS_PATH, exitErr.ExitCode())
			return "[]", nil
		}
		return "", fmt.Errorf("failed to run 'eww get %s': %w", WALLPAPERS_PATH, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func updateEww(paths []string) error {
	compactJSON, err := json.Marshal(paths)
	if err != nil {
		return fmt.Errorf("failed to marshal paths to JSON: %w", err)
	}

	jsonString := string(compactJSON)

	cmd := exec.Command("eww", "update", fmt.Sprintf("%s=%s", WALLPAPERS_PATH, jsonString))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'eww update %s': %w", WALLPAPERS_PATH, err)
	}
	log.Printf("Successfully updated eww variable %s with %d paths.", WALLPAPERS_PATH, len(paths))
	return nil
}

func readFullPaths() ([]string, error) {
	data, err := os.ReadFile(WALLPAPERS_PATH_TMP)
	if err != nil {
		return nil, fmt.Errorf("failed to read full path list from %s: %w. Run ProcessAndWriteWallpapers first", WALLPAPERS_PATH_TMP, err)
	}
	var fullPaths []string
	if err := json.Unmarshal(data, &fullPaths); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON data from %s: %w", WALLPAPERS_PATH_TMP, err)
	}
	return fullPaths, nil
}

func findCurrentPathsIndex(fullPaths []string, currentJSON string) (int, error) {
	var currentPaths []string
	if err := json.Unmarshal([]byte(currentJSON), &currentPaths); err != nil {
		return -1, fmt.Errorf("failed to unmarshal current eww value: %w", err)
	}

	if len(currentPaths) == 0 {
		return 0, fmt.Errorf("current eww wallpaper path array is empty, starting from index 0")
	}

	for i, path := range fullPaths {
		if path == currentPaths[0] {
			return i, nil
		}
	}

	return 0, fmt.Errorf("current first wallpaper path '%s' not found in full list, starting from index 0", currentPaths[0])
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
}

func ProcessAndWriteWallpapers(configPath string) error {
	log.Printf("Starting ProcessAndWriteWallpapers for config: %s", configPath)

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("fatal error reading config file: %w", err)
	}

	imageDir := v.GetString("wallpapers_path")
	if imageDir == "" {
		return fmt.Errorf("no 'wallpapers_path' (directory) found in config or it is empty")
	}

	var fullWallpaperPaths []string

	err := filepath.WalkDir(imageDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {

			return err
		}

		if !d.IsDir() && isImageFile(path) {
			fullWallpaperPaths = append(fullWallpaperPaths, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error traversing directory %s: %w", imageDir, err)
	}

	if len(fullWallpaperPaths) == 0 {
		return fmt.Errorf("no image files found in directory: %s", imageDir)
	}

	sort.Strings(fullWallpaperPaths)

	fullJSON, err := json.Marshal(fullWallpaperPaths)
	if err != nil {
		return fmt.Errorf("failed to marshal full paths to JSON: %w", err)
	}
	if err := os.WriteFile(WALLPAPERS_PATH_TMP, fullJSON, 0644); err != nil {
		return fmt.Errorf("failed to write to %s: %w", WALLPAPERS_PATH_TMP, err)
	}
	log.Printf("Successfully wrote full sorted array (%d paths) to %s", len(fullWallpaperPaths), WALLPAPERS_PATH_TMP)

	compactPaths := fullWallpaperPaths
	if len(compactPaths) > compactArraySize {
		compactPaths = compactPaths[:compactArraySize]
	}

	return updateEww(compactPaths)
}

func NextWallpapers() error {
	log.Println("Starting NextWallpapers")
	fullPaths, err := readFullPaths()
	if err != nil {
		return err
	}

	currentJSON, _ := getEwwCurrentValue()

	currentStartIndex, err := findCurrentPathsIndex(fullPaths, currentJSON)
	if err != nil {
		log.Printf("Warning: %v", err)
	}

	nextStartIndex := currentStartIndex + compactArraySize
	if nextStartIndex >= len(fullPaths) {
		nextStartIndex = 0
	}

	nextEndIndex := nextStartIndex + compactArraySize
	var nextPaths []string

	if nextEndIndex > len(fullPaths) {
		remaining := nextEndIndex - len(fullPaths)
		nextPaths = append(fullPaths[nextStartIndex:], fullPaths[:remaining]...)
	} else {
		nextPaths = fullPaths[nextStartIndex:nextEndIndex]
	}

	return updateEww(nextPaths)
}

func PreviousWallpapers() error {
	log.Println("Starting PreviousWallpapers")
	fullPaths, err := readFullPaths()
	if err != nil {
		return err
	}

	currentJSON, _ := getEwwCurrentValue()

	currentStartIndex, err := findCurrentPathsIndex(fullPaths, currentJSON)
	if err != nil {
		log.Printf("Warning: %v", err)
	}

	previousStartIndex := currentStartIndex - compactArraySize

	if previousStartIndex < 0 {
		start := len(fullPaths) - compactArraySize
		if start < 0 {
			start = 0
		}
		previousStartIndex = start
	}

	previousEndIndex := previousStartIndex + compactArraySize
	if previousEndIndex > len(fullPaths) {
		previousEndIndex = len(fullPaths)
	}

	previousPaths := fullPaths[previousStartIndex:previousEndIndex]

	if len(previousPaths) < compactArraySize && len(fullPaths) >= compactArraySize {
		previousPaths = fullPaths[len(fullPaths)-compactArraySize:]
	}

	return updateEww(previousPaths)
}

func SetWallpaperStartup(configPath string) {
	log.Printf("Starting SetWallpaperStartup for config: %s", configPath)

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return
	}

	wallpaperConfig := viper.GetString("wallpaper")

	if wallpaperConfig != "" {
		// FIXED: correct syntax
		if err := exec.Command("eww", "update", fmt.Sprintf("WALLPAPER=%s", wallpaperConfig)).Run(); err != nil {
			log.Printf("Failed to set wallpaper from config: %v", err)
		}
		return
	}

	wallpaperPathFile := WALLPAPER_TMP
	data, err := os.ReadFile(wallpaperPathFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading wallpaper file: %v", err)
		}
		return
	}

	wp := strings.TrimSpace(string(data))

	if wp != "" {
		if err := exec.Command("eww", "update", fmt.Sprintf("WALLPAPER=%s", wp)).Run(); err != nil {
			log.Printf("Failed to apply fallback wallpaper: %v", err)
		}
	}
}

func SetWallpaper(wallpaperPath string) error {
	log.Printf("Attempting to set wallpaper to: %s", wallpaperPath)

	if _, err := os.Stat(wallpaperPath); os.IsNotExist(err) {
		return fmt.Errorf("validation failed: wallpaper file does not exist at path: %s", wallpaperPath)
	} else if err != nil {
		return fmt.Errorf("validation failed: could not check file status for %s: %w", wallpaperPath, err)
	}

	ewwCommand := fmt.Sprintf("%s=%s", WALLPAPER, wallpaperPath)

	cmd := exec.Command("eww", "update", ewwCommand)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'eww update %s': %w", ewwCommand, err)
	}
	log.Printf("Successfully updated Eww variable %s.", WALLPAPER)

	data := []byte(wallpaperPath)
	if err := os.WriteFile(WALLPAPER_TMP, data, 0644); err != nil {
		return fmt.Errorf("failed to write current wallpaper path to %s: %w", WALLPAPER_TMP, err)
	}
	log.Printf("Successfully wrote current wallpaper path to %s.", WALLPAPER_TMP)

	return nil
}

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func SetRandomWallpaper(configPath string) error {
	log.Println("Starting SetRandomWallpaper")
	var wallpaperPaths []string
	var err error

	// 1. Attempt to read the full sorted list from the temporary file (Cache)
	wallpaperPaths, err = readFullPaths()

	if err != nil || len(wallpaperPaths) == 0 {
		log.Printf("Warning: Failed to read from cache %s, falling back to config file scan. Error: %v", WALLPAPERS_PATH_TMP, err)

		// 2. Fallback: Read config and scan the directory
		v := viper.New()
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")

		if readErr := v.ReadInConfig(); readErr != nil {
			return fmt.Errorf("failed to read config file for fallback scan: %w", readErr)
		}

		imageDir := v.GetString("wallpapers_path")
		if imageDir == "" {
			return fmt.Errorf("fallback failed: 'wallpapers_path' (directory) not found in config")
		}

		// Recursively find image files (similar logic to ProcessAndWriteWallpapers)
		err = filepath.WalkDir(imageDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && isImageFile(path) {
				wallpaperPaths = append(wallpaperPaths, path)
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("fallback failed: error traversing directory %s: %w", imageDir, err)
		}
	}

	if len(wallpaperPaths) == 0 {
		return fmt.Errorf("failed to find any wallpaper images after checking cache and config folder")
	}

	// 3. Pick a random path
	randomIndex := rand.Intn(len(wallpaperPaths))
	randomPath := wallpaperPaths[randomIndex]

	log.Printf("Selected random wallpaper path (Index %d of %d): %s", randomIndex, len(wallpaperPaths), randomPath)

	// 4. Run SetWallpaper with the random path
	return SetWallpaper(randomPath)
}
