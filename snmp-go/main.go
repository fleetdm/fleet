package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type HostInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	MAC      string `json:"mac"`
}

func snmpGet(ip, community, oid string) (string, error) {
	cmd := exec.Command("snmpget", "-v2c", "-c", community, "-t", "1", "-r", "0", "-Oqv", ip, oid)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func firstValidMAC(ip, community string) string {
	for i := 1; i <= 10; i++ { // Scan first 10 interfaces
		oid := fmt.Sprintf("1.3.6.1.2.1.2.2.1.6.%d", i)
		val, _ := snmpGet(ip, community, oid)
		if strings.Contains(val, "No Such") || val == "" {
			continue
		}
		cleaned := strings.ToLower(strings.ReplaceAll(strings.Trim(val, `"`), ":", ""))
		if cleaned != "" {
			return cleaned
		}
	}
	return ""
}

func isSNMPPortOpen(ip string) (bool, error) {
	cmd := exec.Command("nmap", "-sU", "-p", "161", "--host-timeout", "2s", "--max-retries", "1", ip)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		// Non-zero exit can still include useful output
		return false, fmt.Errorf("nmap error: %v\noutput:\n%s", err, out.String())
	}

	output := out.String()
	// Look for "161/udp open"
	if strings.Contains(output, "161/udp open") {
		return true, nil
	}
	return false, nil
}

func scanHost(ip, community string, results *[]HostInfo, mu *sync.Mutex, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	defer func() { <-sem }()

	open, err := isSNMPPortOpen(ip)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error checking SNMP port for", ip, ":", err)
		return
	}
	if !open {
		fmt.Println("SNMP port not open for", ip)
		return
	}

	fmt.Println("Scanning host:", ip)
	get := func(oid string) string {
		out, _ := snmpGet(ip, community, oid)
		if strings.Contains(out, "No Such Object") || strings.Contains(out, "No Such Instance") {
			return ""
		}
		return strings.Trim(out, `"`)
	}

	hostname := get("1.3.6.1.2.1.1.5.0")
	osVal := get("1.3.6.1.2.1.1.1.0")
	mac := firstValidMAC(ip, community)

	if hostname == "" && osVal == "" && mac == "" {
		fmt.Println("No valid SNMP data found for", ip)
		return
	}

	mac = strings.ToLower(strings.ReplaceAll(mac, ":", ""))

	info := HostInfo{
		IP:       ip,
		Hostname: hostname,
		OS:       osVal,
		MAC:      mac,
	}

	mu.Lock()
	*results = append(*results, info)
	mu.Unlock()

	fmt.Println("scan completed for", ip)
}

func main() {
	// subnet := "10.0.200.0/24"
	subnet := "172.21.0.0/21"
	community := "public"
	if len(os.Args) > 1 {
		subnet = os.Args[1]
	}
	if len(os.Args) > 2 {
		community = os.Args[2]
	}

	fmt.Fprintln(os.Stderr, "Running fping to detect live hosts...")

	cmd := exec.Command("fping", "-a", "-g", subnet)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to run fping:", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start fping:", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "fping completed, processing results...")

	scanner := bufio.NewScanner(stdout)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []HostInfo
	sem := make(chan struct{}, 20) // limit concurrency

	fmt.Println("Starting SNMP scans...")

	for scanner.Scan() {
		ip := scanner.Text()
		sem <- struct{}{}
		wg.Add(1)
		go scanHost(ip, community, &results, &mu, &wg, sem)
	}

	wg.Wait()
	_ = cmd.Wait()

	fmt.Println("All scans completed, processing results...")

	// Output as a single JSON array
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(results)
}
