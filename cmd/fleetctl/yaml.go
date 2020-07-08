package main

import "strings"

// splitYaml splits a text file into separate yaml documents divided by ---
func splitYaml(in string) []string {
	var out []string
	for _, chunk := range yamlSeparator.Split(in, -1) {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		out = append(out, chunk)
	}
	return out
}
