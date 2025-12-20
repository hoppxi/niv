package wallpaper

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ncruces/zenity"
	"github.com/spf13/viper"
)

const (
	WALLPAPERS_PATH_TMP = "/tmp/wigo/wallpapers_path"
	WALLPAPER_TMP       = "/tmp/wigo/wallpaper"
	WALLPAPER           = "WALLPAPER"
)

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
}

func SetWallpaperStartup() {
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
		if err := exec.Command("eww", "update", fmt.Sprintf("%s=%s", WALLPAPER, wp)).Run(); err != nil {
			log.Printf("Failed to apply wallpaper: %v", err)
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

	if err := os.MkdirAll(filepath.Dir(WALLPAPER_TMP), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", WALLPAPER_TMP, err)
	}

	if err := os.WriteFile(WALLPAPER_TMP, data, 0644); err != nil {
		return fmt.Errorf("failed to write current wallpaper path to %s: %w", WALLPAPER_TMP, err)
	}
	log.Printf("Successfully wrote current wallpaper path to %s.", WALLPAPER_TMP)

	return nil
}

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func SetRandomWallpaper(v *viper.Viper) error {
	log.Println("Starting SetRandomWallpaper")
	var wallpaperPaths []string
	var err error

	if readErr := v.ReadInConfig(); readErr != nil {
		return fmt.Errorf("failed to read config file for fallback scan: %w", readErr)
	}

	imageDir := v.GetString("wallpapers_path")
	if imageDir == "" {
		return fmt.Errorf("fallback failed: 'wallpapers_path' (directory) not found in config")
	}

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

	if len(wallpaperPaths) == 0 {
		return fmt.Errorf("failed to find any wallpaper images after checking cache and config folder")
	}

	randomIndex := rand.Intn(len(wallpaperPaths))
	randomPath := wallpaperPaths[randomIndex]

	log.Printf("Selected random wallpaper path (Index %d of %d): %s", randomIndex, len(wallpaperPaths), randomPath)

	return SetWallpaper(randomPath)
}

func SelectAndSetWallpaper() error {
	file, err := zenity.SelectFile(
		zenity.Title("Select Wallpaper"),
		zenity.FileFilters{
			{Name: "Images", Patterns: []string{"png", "jpg", "jpeg", "webp"}},
		},
	)

	log.Print(file)
	if err != nil {
		return err
	}

	SetWallpaper(file)

	return nil
}
