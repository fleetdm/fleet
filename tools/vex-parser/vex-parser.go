package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OpenVEXDocument struct {
	Context    string      `json:"context"`
	Statements []Statement `json:"statements"`
	Author     string      `json:"author"`
}

type Statement struct {
	Vulnerability Vulnerability `json:"vulnerability"`
	Status        string        `json:"status"`
	StatusNotes   string        `json:"status_notes"`
	Products      []Product     `json:"products"`
	Justification string        `json:"justification,omitempty"`
	Timestamp     string        `json:"timestamp,omitempty"`
}

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

func generateMarkdown(vex *OpenVEXDocument) string {
	var sb strings.Builder
	for _, stmt := range vex.Statements {
		sb.WriteString(fmt.Sprintf("### %s\n", stmt.Vulnerability.Name))
		sb.WriteString(fmt.Sprintf("- **Author:** %s\n", vex.Author))
		sb.WriteString(fmt.Sprintf("- **Status:** `%s`\n", stmt.Status))
		sb.WriteString(fmt.Sprintf("- **Status notes:** %s\n", stmt.StatusNotes))
		if len(stmt.Products) > 0 {
			sb.WriteString("- **Products:**\n")
			for _, product := range stmt.Products {
				sb.WriteString(fmt.Sprintf("  - `%s`\n", product.ID))
			}
		}
		if stmt.Justification != "" {
			sb.WriteString(fmt.Sprintf("- **Justification:** `%s`\n", stmt.Justification))
		}
		if stmt.Timestamp != "" {
			sb.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", stmt.Timestamp))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func outputMarkdown(vexPath string) {
	vex, err := parseOpenVEX(vexPath)
	if err != nil {
		fmt.Printf("Error parsing OpenVEX file %q: %s\n", vexPath, err)
		return
	}

	md := generateMarkdown(vex)
	fmt.Print(md)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go /path/to/directory/with/vex.json/files/")
		return
	}

	vexPath := os.Args[1]
	vexPaths, err := filepath.Glob(filepath.Join(vexPath, "*.vex.json"))
	if err != nil {
		fmt.Printf("Error processing directory %q: %v\n", vexPath, err)
	}
	if len(vexPaths) == 0 {
		fmt.Printf("No vulnerabilities tracked at the moment.\n\n")
		return
	}

	for _, vexPath := range vexPaths {
		outputMarkdown(vexPath)
	}
}
