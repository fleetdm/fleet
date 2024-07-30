/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package main

import (
	"fmt"

	"github.com/Uptycs/kubequery/internal/k8s/tables"
)

func main() {
	tbls := tables.GetTables()
	fmt.Printf("{\n  \"tables\": [")
	for j, t := range tbls {
		fmt.Printf("    {\n      \"name\": \"%s\",\n      \"columns\": [\n", t.Name)
		for i, c := range t.Columns {
			fmt.Printf("        {\n          \"name\": \"%s\",\n          \"type\": \"%s\"\n", c.Name, c.Type)
			if i < len(t.Columns)-1 {
				fmt.Printf("        },\n")
			} else {
				fmt.Printf("        }\n")
			}
		}
		fmt.Printf("      ]\n")
		if j < len(tbls)-1 {
			fmt.Printf("    },\n")
		} else {
			fmt.Printf("    }\n")
		}
	}
	fmt.Printf("  ]\n}")
}
