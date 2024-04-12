// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/rustsec"
)

func init() {
	flog.AddFlags(flag.CommandLine, nil)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: rustsec2nvd <rustsec-crates-dir>")
		fmt.Println("Example:")
		fmt.Println("git clone https://github.com/RustSec/advisory-db")
		fmt.Println("rustsec2nvd advisory-db/crates > rustsec.cve.json")
		os.Exit(1)
	}

	feed, err := rustsec.Convert(os.Args[1])
	if err != nil {
		flog.Fatal(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(feed)
	if err != nil {
		flog.Fatal(err)
	}
}
