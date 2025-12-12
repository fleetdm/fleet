package test

import (
	"encoding/json"
	"fmt"
)

func PrettyPrintJSON(prefix string, v any) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("pretty-print JSON error: %v\n", err)
	}
	fmt.Println(prefix + ": " + string(bytes))
}
