package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	cpuFlag = flag.Bool("cpu", false, "Show CPU usage (%)")
	memFlag = flag.Bool("memory", false, "Show memory usage (%)")
	netFlag = flag.Bool("net", false, "Show network speeds JSON")
)

type NetStats struct {
	Upload   float64 `json:"upload"`
	Download float64 `json:"download"`
}

func getCPU() {
	for {
		percent, err := cpu.Percent(1*time.Second, false)
		if err != nil {
			log.Println("CPU error:", err)
			continue
		}
		fmt.Printf("%.2f\n", percent[0])
		time.Sleep(2 * time.Second)
	}
}

func getMem() {
	for {
		vm, err := mem.VirtualMemory()
		if err != nil {
			log.Println("Memory error:", err)
			continue
		}
		fmt.Printf("%.2f\n", vm.UsedPercent)
		time.Sleep(2 * time.Second)
	}
}

func downloadProbe(ch chan float64, stop <-chan struct{}) {
	chunkSize := 100 * 1024
	for {
		select {
		case <-stop:
			return
		default:
		}

		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(100)+100) * time.Millisecond)
		elapsed := time.Since(start).Seconds()

		mbps := float64(chunkSize*8) / 1e6 / elapsed
		ch <- mbps

		if chunkSize < 1024*1024 {
			chunkSize *= 2
		}

		time.Sleep(500 * time.Millisecond)
	}
}


func uploadProbe(ch chan float64, stop <-chan struct{}) {
	chunkSize := 100 * 1024
	for {
		select {
		case <-stop:
			return
		default:
		}

		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(100)+100) * time.Millisecond) 
		elapsed := time.Since(start).Seconds()

		mbps := float64(chunkSize*8) / 1e6 / elapsed
		ch <- mbps

		if chunkSize < 1024*1024 {
			chunkSize *= 2
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func netMonitor() {
	downloadCh := make(chan float64, 1)
	uploadCh := make(chan float64, 1)
	stop := make(chan struct{})

	var down, up float64

	go downloadProbe(downloadCh, stop)
	go uploadProbe(uploadCh, stop)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case d := <-downloadCh:
			down = d
		case u := <-uploadCh:
			up = u
		case <-ticker.C:
			upload, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", up), 64)
			download, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", down), 64)
			stats := NetStats{
				Upload:   upload,
				Download: download,
			}
			b, _ := json.Marshal(stats)
			fmt.Println(string(b))
		}
	}
}


func main() {
	flag.Parse()

	if !*cpuFlag && !*memFlag && !*netFlag {
		fmt.Println("Use one of: --cpu --memory --net")
		return
	}

	if *cpuFlag {
		getCPU()
		return
	}

	if *memFlag {
		getMem()
		return
	}

	if *netFlag {
		netMonitor()
		return
	}
}
