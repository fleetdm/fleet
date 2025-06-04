package snmp

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
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
		log.Error().Msg("Network scan: is disabled in Orbit configuration.")
		return nil
	}

	log.Error().Msg("Network scan: Starting SNMP network scan...")

	subnet := "10.211.55.0/24"
	community := "public"

	log.Error().Msg("Network scan: Running fping to detect live hosts...")

	cmd := exec.Command("fping", "-a", "-g", subnet)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Network scan: Failed to run fping")
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Network scan: Failed to start fping")
		os.Exit(1)
	}

	log.Error().Msg("Network scan: fping completed, processing results...")

	scanner := bufio.NewScanner(stdout)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []SnmpHost
	sem := make(chan struct{}, 20) // limit concurrency

	log.Error().Msg("Network scan: Starting SNMP scans...")

	for scanner.Scan() {
		ip := scanner.Text()
		sem <- struct{}{}
		wg.Add(1)
		go scanHost(ip, community, &results, &mu, &wg, sem)
	}

	wg.Wait()
	_ = cmd.Wait()

	if err := sr.sender.SendSnmpHostsResponse(results); err != nil {
		log.Error().Err(err).Msg("Network scan: Error sending SNMP hosts response")
		return nil
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
		log.Error().Err(err).Msg("Network scan: Error checking SNMP port for " + ip)
		return
	}
	if !open {
		log.Error().Msg("Network scan: SNMP port not open for " + ip)
		return
	}

	log.Error().Msg("Network scan: Scanning host " + ip)
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
		log.Error().Msg("Network scan: No valid SNMP data found for " + ip)
		return
	}

	hash := sha256.Sum256([]byte(osVal))
	hashString := hex.EncodeToString(hash[:])

	info := SnmpHost{
		IP:       ip,
		Hostname: hostname,
		OS:       osVal,
		UUID:     hashString,
	}

	mu.Lock()
	*results = append(*results, info)
	mu.Unlock()

	log.Error().Msg("Network scan: scan completed for " + ip)
}

func isSNMPPortOpen(ip string) (bool, error) {
	cmd := exec.Command("/opt/homebrew/bin/nmap", "-sU", "-p", "161", "--host-timeout", "2s", "--max-retries", "1", ip)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		// Non-zero exit can still include useful output
		return false, fmt.Errorf("Network scan: nmap error: %v\noutput:\n%s", err, out.String())
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
