package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	statsURL  = "http://srv.msk01.gigacorp.local/_stats"
	interval  = 1 * time.Second
	timeout   = 3 * time.Second
	maxErrors = 3

	loadThreshold = 30.0
	memThreshold  = 80.0
	diskThreshold = 90.0
	netThreshold  = 90.0
)

func main() {
	client := &http.Client{Timeout: timeout}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	consecutiveErrors := 0

	for {
		ok := pollOnce(client)
		if ok {
			consecutiveErrors = 0
		} else {
			consecutiveErrors++
			if consecutiveErrors >= maxErrors {
				fmt.Println("Unable to fetch server statistic.")
			}
		}

		<-ticker.C
	}
}

func pollOnce(client *http.Client) bool {
	req, err := http.NewRequest(http.MethodGet, statsURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return false
	}

	line, err := readSingleLine(resp.Body)
	if err != nil {
		return false
	}

	parts := strings.Split(strings.TrimSpace(line), ",")
	if len(parts) != 7 {
		return false
	}

	load, err := parseFloat(parts[0])
	if err != nil {
		return false
	}

	memTotal, err := parseUint(parts[1])
	if err != nil || memTotal == 0 {
		return false
	}
	memUsed, err := parseUint(parts[2])
	if err != nil {
		return false
	}

	diskTotal, err := parseUint(parts[3])
	if err != nil || diskTotal == 0 {
		return false
	}
	diskUsed, err := parseUint(parts[4])
	if err != nil {
		return false
	}

	netCap, err := parseUint(parts[5])
	if err != nil || netCap == 0 {
		return false
	}
	netUsed, err := parseUint(parts[6])
	if err != nil {
		return false
	}

	if load > loadThreshold {

		fmt.Printf("Load Average is too high: %s\n", strings.TrimSpace(parts[0]))
	}

	memPct := (float64(memUsed) / float64(memTotal)) * 100.0
	if memPct > memThreshold {
		fmt.Printf("Memory usage too high: %d%%\n", roundToInt(memPct))
	}

	diskPct := (float64(diskUsed) / float64(diskTotal)) * 100.0
	if diskPct > diskThreshold {
		freeBytes := int64(diskTotal) - int64(diskUsed)
		if freeBytes < 0 {
			freeBytes = 0
		}
		freeMB := freeBytes / (1024 * 1024)
		fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
	}

	netPct := (float64(netUsed) / float64(netCap)) * 100.0
	if netPct > netThreshold {
		freeBytesPerSec := int64(netCap) - int64(netUsed)
		if freeBytesPerSec < 0 {
			freeBytesPerSec = 0
		}
		// Mbit/s: bytes/s * 8 / 1024 / 1024
		freeMbit := (float64(freeBytesPerSec) * 8.0) / (1024.0 * 1024.0)
		fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", roundToInt(freeMbit))
	}

	return true
}

func readSingleLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	s, err := br.ReadString('\n')
	if err == io.EOF {
		return s, nil
	}
	return s, err
}

func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(s), 10, 64)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func roundToInt(x float64) int64 {
	if x < 0 {
		return int64(x - 0.5)
	}
	return int64(x + 0.5)
}


var _ = os.Stdout
