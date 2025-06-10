package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type OpenVEXDocument struct {
	Context    string      `json:"context"`
	Statements []Statement `json:"statements"`
	Author     string      `json:"author"`
}

type Statement struct {
	Vulnerability   Vulnerability `json:"vulnerability"`
	Status          string        `json:"status"`
	StatusNotes     string        `json:"status_notes"`
	Products        []Product     `json:"products"`
	Justification   string        `json:"justification"`
	Aliases         []string      `json:"aliases"`
	ActionStatement string        `json:"action_statement"`
	Timestamp       string        `json:"timestamp"`
}

const timeFormat = "2006-01-02T15:04:05.999999Z07:00"

type Vulnerability struct {
	Name string `json:"name"`
}

type Product struct {
	ID string `json:"@id"`
}

func parseOpenVEX(filePath string) (*OpenVEXDocument, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var vex OpenVEXDocument
	err = json.Unmarshal(data, &vex)
	if err != nil {
		return nil, err
	}
	return &vex, nil
}

func generateMarkdown(vex *OpenVEXDocument) (string, error) {
	var sb strings.Builder
	cve := vex.Statements[0].Vulnerability.Name
	for _, stmt := range vex.Statements[1:] {
		if stmt.Vulnerability.Name != cve {
			return "", fmt.Errorf("VEX statement does not match CVE: %s vs %s", stmt.Vulnerability.Name, cve)
		}
	}
	sb.WriteString(fmt.Sprintf("### [%s](https://nvd.nist.gov/vuln/detail/%s)\n", cve, cve))
	sort.Slice(vex.Statements, func(i, j int) bool {
		ti, _ := time.Parse(timeFormat, vex.Statements[i].Timestamp)
		tj, _ := time.Parse(timeFormat, vex.Statements[j].Timestamp)
		return ti.After(tj)
	})
	multipleStatements := len(vex.Statements) > 1
	for _, stmt := range vex.Statements {
		if multipleStatements {
			sb.WriteString("#### Statement:\n")
		}
		if len(stmt.Aliases) > 0 {
			sb.WriteString(fmt.Sprintf("- **Aliases:** %s", strings.Join(stmt.Aliases, ",")))
		}
		sb.WriteString(fmt.Sprintf("- **Author:** %s\n", vex.Author))
		sb.WriteString(fmt.Sprintf("- **Status:** `%s`\n", stmt.Status))
		if stmt.StatusNotes != "" {
			statusNotes := stmt.StatusNotes
			if !strings.HasSuffix(statusNotes, ".") {
				statusNotes += "."
			}
			sb.WriteString(fmt.Sprintf("- **Status notes:** %s\n", statusNotes))
		}
		sb.WriteString("- **Products:**: ")
		var ids []string
		for _, product := range stmt.Products {
			ids = append(ids, "`"+product.ID+"`")
		}
		sb.WriteString(strings.Join(ids, ",") + "\n")
		if stmt.Justification != "" {
			sb.WriteString(fmt.Sprintf("- **Justification:** `%s`\n", stmt.Justification))
		}
		if stmt.ActionStatement != "" {
			sb.WriteString(fmt.Sprintf("- **Action statement:** `%s`\n", stmt.ActionStatement))
		}
		if stmt.Timestamp != "" {
			t, err := time.Parse(timeFormat, stmt.Timestamp)
			if err != nil {
				return "", fmt.Errorf("parsing timestamp %s for %s: %s", stmt.Timestamp, stmt.Vulnerability.Name, err)
			}
			sb.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", t.Format(time.DateTime)))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func outputMarkdown(vexPath string) error {
	vex, err := parseOpenVEX(vexPath)
	if err != nil {
		return err
	}

	md, err := generateMarkdown(vex)
	if err != nil {
		return err
	}
	fmt.Print(md)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go /path/to/directory/with/vex.json/files/")
		os.Exit(1)
	}

	vexPath := os.Args[1]
	vexPaths, err := filepath.Glob(filepath.Join(vexPath, "*.vex.json"))
	if err != nil {
		fmt.Printf("Error processing directory %q: %v\n", vexPath, err)
		os.Exit(1)
	}
	if len(vexPaths) == 0 {
		fmt.Printf("No vulnerabilities tracked at the moment.\n\n")
		return
	}

	sort.Slice(vexPaths, func(i, j int) bool {
		return vexPaths[i] > vexPaths[j]
	})
	for _, vexPath := range vexPaths {
		if err := outputMarkdown(vexPath); err != nil {
			fmt.Printf("Error parsing OpenVEX file %q: %s\n", vexPath, err)
			os.Exit(1)
		}
	}
}
