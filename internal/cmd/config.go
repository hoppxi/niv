package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flags variables
var (
	flagGet      string
	flagSet      string
	flagToggle   string
	flagValidate bool
	flagList     bool
	flagFile     bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration via flags",
	Long:  `Manage configuration. Use flags to interact with the settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. --file: Show config file path
		if flagFile {
			fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
			return
		}

		// 2. --list: Dump all settings
		if flagList {
			fmt.Println("Current Settings:")
			for key, value := range viper.AllSettings() {
				fmt.Printf("  %s: %v\n", key, value)
			}
			return
		}

		// 3. --validate: Run deep validation
		if flagValidate {
			performValidation()
			return
		}

		// 4. --get: Retrieve a value
		if flagGet != "" {
			val := viper.Get(flagGet)
			if val == nil {
				fmt.Printf("Key '%s' not found.\n", flagGet)
			} else {
				fmt.Printf("%v\n", val)
			}
			return
		}

		// 5. --set: Update a value (Format: "key=value")
		if flagSet != "" {
			parts := strings.SplitN(flagSet, "=", 2)
			if len(parts) != 2 {
				log.Fatal("Invalid format for --set. Use: --set 'key=value'")
			}
			key, val := parts[0], parts[1]
			
			viper.Set(key, val)
			if err := viper.WriteConfig(); err != nil {
				log.Fatalf("Error writing config: %v", err)
			}
			fmt.Printf("Updated '%s' to '%s'\n", key, val)
			return
		}

		// 6. --toggle: Switch a boolean
		if flagToggle != "" {
			val := viper.Get(flagToggle)
			b, ok := val.(bool)
			if !ok {
				log.Fatalf("Key '%s' is not a boolean (current value: %v)", flagToggle, val)
			}
			
			viper.Set(flagToggle, !b)
			if err := viper.WriteConfig(); err != nil {
				log.Fatalf("Error writing config: %v", err)
			}
			fmt.Printf("Toggled '%s' to %v\n", flagToggle, !b)
			return
		}

		// If no flags provided
		cmd.Help()
	},
}

// performValidation checks if apps exist in PATH and if paths are valid
func performValidation() {
	fmt.Println("🔍 Validating configuration...")
	hasError := false

	// Helper to check executable
	checkBin := func(name, bin string) {
		if bin == "" {
			return 
		}
		// Handle command strings like "alacritty -e htop" -> just check "alacritty"
		cmdParts := strings.Fields(bin)
		if len(cmdParts) > 0 {
			if _, err := exec.LookPath(cmdParts[0]); err != nil {
				fmt.Printf("❌ [App] %s: Command '%s' not found in PATH.\n", name, cmdParts[0])
				hasError = true
			} else {
				fmt.Printf("✅ [App] %s: OK\n", name)
			}
		}
	}

	// 1. Validate Apps (Terminal, Browser, etc.)
	checkBin("Terminal", viper.GetString("apps.terminal"))
	checkBin("Browser", viper.GetString("apps.browser"))
	checkBin("Editor", viper.GetString("apps.editor"))
	checkBin("File Manager", viper.GetString("apps.file_manager"))

	// 2. Validate General Paths
	profilePic := viper.GetString("general.profile_pic")
	if profilePic != "" {
		if _, err := os.Stat(profilePic); os.IsNotExist(err) {
			fmt.Printf("❌ [General] Profile Pic: File not found at '%s'\n", profilePic)
			hasError = true
		} else {
			fmt.Printf("✅ [General] Profile Pic: OK\n")
		}
	}

	// 3. Validate Pinned Apps structure
	// We need to unmarshal to check the list structure easily
	type PinnedApp struct {
		Name    string
		Command string
	}
	var pinned []PinnedApp
	if err := viper.UnmarshalKey("pinned_apps", &pinned); err == nil {
		for i, app := range pinned {
			if app.Name == "" {
				fmt.Printf("⚠️ [Pinned App #%d] Missing 'name'\n", i+1)
			}
			if app.Command == "" {
				fmt.Printf("❌ [Pinned App #%d] Missing 'command'\n", i+1)
				hasError = true
			} else {
				checkBin(fmt.Sprintf("Pinned:%s", app.Name), app.Command)
			}
		}
	}

	// 4. Validate Notification History Path
	notifHistory := viper.GetString("notifications.history")
	if notifHistory != "" {
		// Expand ~ if necessary (simple check)
		if strings.HasPrefix(notifHistory, "~/") {
			home, _ := os.UserHomeDir()
			notifHistory = strings.Replace(notifHistory, "~", home, 1)
		}
		dir := notifHistory[:strings.LastIndex(notifHistory, "/")]
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("❌ [Notifications] History directory does not exist: %s\n", dir)
			hasError = true
		}
	}

	if !hasError {
		fmt.Println("\n🎉 Configuration is valid and healthy!")
	} else {
		fmt.Println("\n⚠️  Configuration has errors.")
		os.Exit(1)
	}
}

func init() {
	// Register flags for the command
	configCmd.Flags().StringVar(&flagGet, "get", "", "Get specific key value")
	configCmd.Flags().StringVar(&flagSet, "set", "", "Set key=value")
	configCmd.Flags().StringVar(&flagToggle, "toggle", "", "Toggle boolean key")
	configCmd.Flags().BoolVar(&flagValidate, "validate", false, "Validate config integrity")
	configCmd.Flags().BoolVar(&flagList, "list", false, "List all settings")
	configCmd.Flags().BoolVar(&flagFile, "file", false, "Show config file path")

	// Assuming rootCmd exists
	// rootCmd.AddCommand(configCmd)
}