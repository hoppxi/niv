package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	const bars = 60

	config := fmt.Sprintf(`[general]
bars = %d
[output]
method = raw
data_format = ascii
ascii_max_range = 100
`, bars)

	cmd := exec.Command("cava", "-p", "/dev/stdin")
	cmd.Stdin = bytes.NewBufferString(config)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(stdout)

	// batch updates here, updating eww once per interval
	const updateInterval = 10 * time.Millisecond
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	var lastOutput string
	updateNeeded := false

	go func() {
		for range ticker.C {
			if updateNeeded {
				// Run eww update only if data changed
				ewwCmd := exec.Command("eww", "update", fmt.Sprintf("CAVA_BARS=%s", lastOutput))
				if err := ewwCmd.Run(); err != nil {
					fmt.Printf("failed to run eww update: %v\n", err)
				}
				updateNeeded = false
			}
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()
		rawBars := strings.Split(line, ";")

		for i, v := range rawBars {
			trimmed := strings.TrimSpace(v)
			val, err := strconv.Atoi(trimmed)
			if err != nil {
				val = 0
			}
			if val < 0 {
				val = 0
			}
			if val > 100 {
				val = 100
			}
			rawBars[i] = strconv.Itoa(val)
		}

		cavaOutput := "[" + strings.Join(rawBars, ",") + "]"

		// Only mark updateNeeded if output changed
		if cavaOutput != lastOutput {
			lastOutput = cavaOutput
			updateNeeded = true
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if err := cmd.Wait(); err != nil {
		panic(err)
	}
}
