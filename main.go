package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	URL          = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval = 1 * time.Second
)

func main() {
	url := os.Getenv("STATS_URL")
	if url == "" {
		url = URL
	}
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	consecutiveFails := 0

	for range ticker.C {
		alerts, ok := fetchAndCheck(client, url)
		if ok {
			consecutiveFails = 0
			for _, line := range alerts {
				fmt.Println(line)
			}
			continue
		}
		consecutiveFails++
		if consecutiveFails >= 3 {
			fmt.Println("Unable to fetch server statistic.")
		}
	}
}

func fetchAndCheck(client *http.Client, url string) ([]string, bool) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}

	fields := strings.Split(strings.TrimSpace(string(body)), ",")
	if len(fields) != 7 {
		return nil, false
	}

	trim := func(s string) string { return strings.TrimSpace(s) }

	loadToken := trim(fields[0])
	load, err0 := strconv.ParseFloat(loadToken, 64)

	memTotal, err1 := parseU64(fields[1])
	memUsed, err2 := parseU64(fields[2])
	diskTotal, err3 := parseU64(fields[3])
	diskUsed, err4 := parseU64(fields[4])
	netCap, err5 := parseU64(fields[5])
	netUsed, err6 := parseU64(fields[6])
	if err0 != nil || err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
		return nil, false
	}

	var out []string

	if load > 30.0 {
		out = append(out, fmt.Sprintf("Load Average is too high: %s", loadToken))
	}

	if memTotal > 0 {
		pct := (memUsed * 100) / memTotal
		if pct > 80 {
			out = append(out, fmt.Sprintf("Memory usage too high: %d%%", pct))
		}
	}

	if diskTotal > 0 && diskUsed*100 > diskTotal*90 {
		freeMB := (diskTotal - diskUsed) / (1024 * 1024)
		out = append(out, fmt.Sprintf("Free disk space is too low: %d Mb left", freeMB))
	}

	if netCap > 0 && netUsed*100 > netCap*90 {
		freeMbit := ((netCap - netUsed) * 8) / (1024 * 1024)
		out = append(out, fmt.Sprintf("Network bandwidth usage high: %d Mbit/s available", freeMbit))
	}
	return out, true
}

func parseU64(s string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(s), 10, 64)
}
