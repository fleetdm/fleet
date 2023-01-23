package table

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func KolideVulnerabilities(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("name"),
		table.IntegerColumn("vulnerable"),
		table.TextColumn("details"),
	}
	return table.NewPlugin("kolide_vulnerabilities", columns, generateKolideVulnerabilities(client, logger))
}

var generateFuncs = []func(log log.Logger) map[string]string{
	generateCVE_2017_7149,
}

func generateKolideVulnerabilities(client *osquery.ExtensionManagerClient, logger log.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		results := []map[string]string{}

		for _, f := range generateFuncs {
			results = append(results, f(logger))
		}

		return results, nil
	}
}

func generateCVE_2017_7149(logger log.Logger) map[string]string {
	row := map[string]string{"name": "CVE-2017-7149"}
	volumes, err := getEncryptedAPFSVolumes()
	if err != nil {
		level.Error(logger).Log("err", fmt.Errorf("getting encrypted APFS volumes: %w", err))
		return row
	}

	details := struct {
		Vulnerable []string `json:"vulnerable"`
	}{}
	for _, vol := range volumes {
		vulnerable, err := checkVolumeVulnerability(vol)
		if err != nil {
			level.Error(logger).Log("err", fmt.Errorf("checking volume %s vulnerability: %w", vol, err))
			continue
		}

		if vulnerable {
			details.Vulnerable = append(details.Vulnerable, vol)
		}
	}

	if len(details.Vulnerable) == 0 {
		row["vulnerable"] = "0"
		return row
	}

	row["vulnerable"] = "1"

	detailJSON, err := json.Marshal(details)
	if err != nil {
		level.Error(logger).Log("err", fmt.Errorf("marshalling CVE_2017_7149 details: %w", err))
		return row
	}
	row["details"] = string(detailJSON)

	return row
}

// getEncryptedAPFSVolumes returns the list of volume names that are encrypted
// APFS volumes.
func getEncryptedAPFSVolumes() ([]string, error) {
	cmd := exec.Command("diskutil", "apfs", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("execing diskutil apfs list: %w", err)
	}

	volumeSection := regexp.MustCompile(`(?s)Volume .+? Encrypted:\s+(Yes|No)`)
	isEncrypted := regexp.MustCompile(`Encrypted:\s+Yes`)
	volumeName := regexp.MustCompile(`Volume (\S+)`)

	volumes := []string{}
	for _, section := range volumeSection.FindAllString(string(out), -1) {
		if !isEncrypted.MatchString(section) {
			// Not an encrypted volume
			continue
		}

		matches := volumeName.FindStringSubmatch(section)
		if len(matches) != 2 {
			continue
		}

		volumes = append(volumes, matches[1])
	}

	return volumes, nil
}

func checkVolumeVulnerability(volume string) (bool, error) {
	cmd := exec.Command("diskutil", "apfs", "listCryptoUsers", volume)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("execing diskutil apfs listCryptoUsers %s: %w", volume, err)
	}

	userSectionWithHint := regexp.MustCompile(`(?s) (\S+-\S+-\S+-\S+-\S+).+? Hint: ([^\n]+)`)
	for _, matches := range userSectionWithHint.FindAllStringSubmatch(string(out), -1) {
		if len(matches) != 3 {
			continue
		}
		uuid := matches[1]
		passHint := matches[2]

		if testVolumeUser(volume, uuid, passHint) {
			return true, nil
		}
	}

	return false, nil
}

func testVolumeUser(volume, uuid, passHint string) bool {
	cmd := exec.Command("diskutil", "apfs", "unlockVolume", volume, "-verify", "-user", uuid, "-passphrase", passHint)
	err := cmd.Run()
	// If cmd exits zero, the password hint was the password
	return err == nil
}
