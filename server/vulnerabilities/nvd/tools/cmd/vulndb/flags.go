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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/debug"
)

var (
	// See https://github.com/go-sql-driver/mysql#dsn-data-source-name for DSN.
	// And libfb/go/fbmysql for fbmysql DSN.
	gFlagMySQL = os.Getenv("MYSQL")

	// General purpose flags.

	gFlagOwner       = os.Getenv("USER")
	gFlagCollector   = ""
	gFlagProvider    = ""
	gFlagMetadata    = ""
	gFlagFormat      = "csv"
	gFlagDeadline    deadlineFlag
	gFlagDeleteAll   bool
	gFlagCSVNoHeader = false
)

func init() {
	fs := RootCmd.PersistentFlags()
	fs.VarP(&debug.Level, "debug", "v", "set verbosity level")
}

func addRequiredFlags(cmd *cobra.Command, names ...string) {
	addFlags(cmd, true, names...)
}

func addOptionalFlags(cmd *cobra.Command, names ...string) {
	addFlags(cmd, false, names...)
}

func addFlags(cmd *cobra.Command, required bool, names ...string) {
	for _, name := range names {
		f, exists := supportedFlags[name]
		if !exists {
			panic("unsupported flag: " + name)
		}
		f(cmd.Flags())
		if required {
			// This requires calling fs.Set for flags that
			// have a default value.
			cmd.MarkFlagRequired(name)
		}
	}
}

var supportedFlags = map[string]func(*pflag.FlagSet){
	"mysql": func(fs *pflag.FlagSet) {
		fs.StringVar(&gFlagMySQL, "mysql", gFlagMySQL, "set mysql dsn (or use $MYSQL)")
		if gFlagMySQL != "" {
			fs.Set("mysql", gFlagMySQL)
		}
	},
	"owner": func(fs *pflag.FlagSet) {
		fs.StringVar(&gFlagOwner, "owner", gFlagOwner, "set owner of the records")
		fs.Set("owner", gFlagOwner)
	},
	"collector": func(fs *pflag.FlagSet) {
		fs.StringVar(&gFlagCollector, "collector", gFlagCollector, "set unique name of the data collector")
	},
	"provider": func(fs *pflag.FlagSet) {
		fs.StringVar(&gFlagProvider, "provider", gFlagProvider, "set short name of the data provider")
	},
	"metadata": func(fs *pflag.FlagSet) {
		fs.StringVar(&gFlagMetadata, "metadata", gFlagMetadata, "set metadata")
	},
	"format": func(fs *pflag.FlagSet) {
		fs.StringVar(&gFlagFormat, "format", gFlagFormat, "set output format (csv or nvdcvejson)")
		fs.Set("format", gFlagFormat)
	},
	"deadline": func(fs *pflag.FlagSet) {
		fs.Var(&gFlagDeadline, "deadline", fmt.Sprintf("set deadline in absolute time or duration (e.g. %s or 24h, 30d)", vulndb.TimeLayout))
	},
	"delete_all": func(fs *pflag.FlagSet) {
		fs.BoolVarP(&gFlagDeleteAll, "all", "a", gFlagDeleteAll, "delete all records from database")
	},
	"csv_noheader": func(fs *pflag.FlagSet) {
		fs.BoolVarP(&gFlagCSVNoHeader, "csvnoheader", "n", gFlagCSVNoHeader, "omit csv header in output")
	},
}

// deadlineFlag implements the pflag.Value interface.
type deadlineFlag struct {
	Time time.Time
}

func (d *deadlineFlag) Type() string {
	return "string"
}

func (d *deadlineFlag) String() string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.String()
}

// Set sets v as the deadline's time. Takes same input as time.ParseDuration
// but supports using 'd' (e.g. 30d) for representing days as d*24h.
func (d *deadlineFlag) Set(v string) error {
	t, err := vulndb.ParseTime(v)
	if err == nil {
		d.Time = t
		return nil
	}
	dd, err := time.ParseDuration(v)
	if err == nil {
		d.Time = time.Now().Add(dd)
		return nil
	}
	idx := strings.Index(v, "d")
	if idx < 1 {
		return fmt.Errorf("invalid deadline: %q", v)
	}
	n, err := strconv.Atoi(v[0:idx])
	if err != nil {
		return fmt.Errorf("invalid deadline: %q", v)
	}
	dd, _ = time.ParseDuration(strconv.Itoa(n*24) + "h")
	d.Time = time.Now().Add(dd)
	return nil
}
