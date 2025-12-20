package cmd

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoppxi/wigo/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Apps struct {
		Terminal      string `yaml:"terminal"`
		Editor        string `yaml:"editor"`
		SystemMonitor string `yaml:"system_monitor"`
		FileManager   string `yaml:"file_manager"`
		SystemInfo    string `yaml:"system_info"`
		Screenshot    string `yaml:"screenshot_tool"`
	} `yaml:"apps"`
	General struct {
		DisplayName string `yaml:"display_name"`
		Icon        string `yaml:"top_left_icon"`
		ProfilePic  string `yaml:"profile_pic"`
	} `yaml:"general"`
	WallpaperPath string `yaml:"wallpapers_path"`
}

func getWigoDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "eww")
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Full initialization (Config + Assets)",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		wigoDir := getWigoDir()

		if _, err := os.Stat(wigoDir); !os.IsNotExist(err) {
			fmt.Printf("Warning: Wigo config already exists at %s\n", wigoDir)
			if !confirm(reader, "Continuing will overwrite your current configs Proceed?") {
				return
			}
		}

		os.MkdirAll(wigoDir, 0755)
		generateWigoYaml(reader, wigoDir)
		fmt.Println("\nExtracting assets and styles...")
		extractEmbed(wigoDir)

		fmt.Println("\nFull setup complete!")
	},
}

var generateConfigCmd = &cobra.Command{
	Use:   "generate-config",
	Short: "Only generate/update the wigo.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		wigoDir := getWigoDir()

		yamlPath := filepath.Join(wigoDir, "wigo.yaml")
		if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
			if !confirm(reader, "wigo.yaml already exists. Overwrite with new settings?") {
				return
			}
		}

		os.MkdirAll(wigoDir, 0755)
		generateWigoYaml(reader, wigoDir)
		fmt.Println("Config file updated.")
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update assets and styles (preserves wigo.yaml)",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		wigoDir := getWigoDir()

		fmt.Println("This will overwrite your src, assets, and scss files with the latest versions.")
		if !confirm(reader, "Are you sure you want to update assets?") {
			return
		}

		err := extractEmbed(wigoDir)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("Assets and styles updated successfully.")
	},
}

func generateWigoYaml(reader *bufio.Reader, targetDir string) {
	conf := Config{}
	conf.Apps.Terminal = prompt(reader, "Terminal Emulator", "alacritty")
	conf.Apps.Editor = prompt(reader, "Text Editor", "code")
	conf.Apps.SystemMonitor = prompt(reader, "System Monitor", "alacritty --hold -e btop")
	conf.Apps.FileManager = prompt(reader, "File Manager", "nautilus")
	conf.Apps.SystemInfo = prompt(reader, "System Info Tool", "alacritty --hold -e neofetch")
	conf.Apps.Screenshot = prompt(reader, "Screenshot Tool", "flameshot launcher")

	conf.General.DisplayName = prompt(reader, "Your Display Name", "User")
	conf.General.Icon = prompt(reader, "Top Left Icon", "distributor-logo-nixos")
	conf.General.ProfilePic = prompt(reader, "Profile Picture Path", "")

	conf.WallpaperPath = prompt(reader, "Wallpapers Directory", filepath.Join(os.Getenv("HOME"), "Pictures/Wallpapers"))

	d, _ := yaml.Marshal(&conf)
	os.WriteFile(filepath.Join(targetDir, "wigo.yaml"), d, 0644)
}

func prompt(r *bufio.Reader, label, defaultValue string) string {
	fmt.Printf("%s [%s]: ", label, defaultValue)
	input, _ := r.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func confirm(r *bufio.Reader, message string) bool {
	fmt.Printf("%s (y/N): ", message)
	input, _ := r.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

func extractEmbed(targetDir string) error {
	embeds := config.ConfigFS()
	return fs.WalkDir(embeds, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}

		targetPath := filepath.Join(targetDir, path)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		content, err := embeds.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, content, 0644)
	})
}
