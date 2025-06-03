package snmp

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type SnmpRunner struct {
	sender SnmpHostsSender
}

type SnmpHost struct {
	IP       string `json:"ip_address"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	UUID     string `json:"uuid"`
}

func (sr *SnmpRunner) Run(oc *fleet.OrbitConfig) error {
	if !oc.Notifications.ScanNetwork {
		fmt.Fprintln(os.Stderr, "Network scan is disabled in Orbit configuration.")
		return nil
	}

	fmt.Fprintln(os.Stderr, "Starting SNMP network scan...")

	subnet := "10.211.55.0/24"
	community := "public"

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
	var results []SnmpHost
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

	if err := sr.sender.SendSnmpHostsResponse(results); err != nil {
		fmt.Fprintln(os.Stderr, "Error sending SNMP hosts response:", err)
	}

	return nil
}

func New(sender SnmpHostsSender) *SnmpRunner {
	return &SnmpRunner{
		sender: sender,
	}
}

type SnmpHostsSender interface {
	SendSnmpHostsResponse([]SnmpHost) error
}

func scanHost(ip, community string, results *[]SnmpHost, mu *sync.Mutex, wg *sync.WaitGroup, sem chan struct{}) {
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

	if hostname == "" && osVal == "" {
		fmt.Println("No valid SNMP data found for", ip)
		return
	}

	hash := sha256.Sum256([]byte(osVal))
	hashString := fmt.Sprintf("%x", hash)
	id := md5.Sum([]byte(fmt.Sprintf("%s:%s", ip, hashString)))

	info := SnmpHost{
		IP:       ip,
		Hostname: hostname,
		OS:       osVal,
		UUID:     string(id[:]),
	}

	mu.Lock()
	*results = append(*results, info)
	mu.Unlock()

	fmt.Println("scan completed for", ip)
}

func isSNMPPortOpen(ip string) (bool, error) {
	cmd := exec.Command("/opt/homebrew/bin/nmap", "-sU", "-p", "161", "--host-timeout", "2s", "--max-retries", "1", ip)
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

func snmpGet(ip, community, oid string) (string, error) {
	cmd := exec.Command("snmpget", "-v2c", "-c", community, "-t", "1", "-r", "0", "-Oqv", ip, oid)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}
