package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	statsURL = "http://srv.msk01.gigacorp.local/_stats"


	pollInterval = 100 * time.Millisecond
	httpTimeout  = 2 * time.Second

	maxConsecutiveErrors = 3
)

func main() {
	client := &http.Client{Timeout: httpTimeout}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	consecutiveErrors := 0

	for {
		if pollOnce(client) {
			consecutiveErrors = 0
		} else {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				fmt.Println("Unable to fetch server statistic.")
			}
		}

		<-ticker.C
	}
}

func pollOnce(client *http.Client) bool {
	resp, err := client.Get(statsURL)
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
	line = strings.TrimSpace(line)

	parts := strings.Split(line, ",")
	if len(parts) != 7 {
		return false
	}


	load, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
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


	if load > 30 {

		fmt.Printf("Load Average is too high: %s\n", strings.TrimSpace(parts[0]))
	}


	memPct := (memUsed * 100) / memTotal
	if memPct > 80 {
		fmt.Printf("Memory usage too high: %d%%\n", memPct)
	}


	diskPct := (diskUsed * 100) / diskTotal
	if diskPct > 90 {
		var freeBytes uint64
		if diskUsed >= diskTotal {
			freeBytes = 0
		} else {
			freeBytes = diskTotal - diskUsed
		}
		freeMB := freeBytes / (1024 * 1024)
		fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
	}


	netPct := (netUsed * 100) / netCap
	if netPct > 90 {
		var freeBps uint64
		if netUsed >= netCap {
			freeBps = 0
		} else {
			freeBps = netCap - netUsed
		}
		freeMbitLike := freeBps / 1_000_000
		fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", freeMbitLike)
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
