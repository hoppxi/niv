package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill EWW widgets and stop watchers via IPC.",
	Run: func(cmd *cobra.Command, args []string) {
		socketPath := filepath.Join(os.TempDir(), "niv.sock")

		// 1. Connect to the IPC server
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			fmt.Println("Error: Niv start daemon is not running or socket not found.")
			fmt.Printf("Details: %v\n", err)
			return
		}
		defer conn.Close()

		// 2. Send the command
		fmt.Println("Connected. Sending STOP command...")
		conn.Write([]byte("STOP"))

		// 3. Read the server's response
		response, _ := io.ReadAll(conn)
		responseStr := strings.TrimSpace(string(response))
		
		fmt.Printf("Server response: %s\n", responseStr)
        
		if strings.Contains(responseStr, "Shutting down") {
					fmt.Println("Niv start process successfully instructed to shut down.")
		}
	},
}

