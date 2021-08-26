package main

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func createConfigDumpCmd(configManager config.Manager) *cobra.Command {
	var configDumpCmd = &cobra.Command{
		Use:   "config_dump",
		Short: "Dump the parsed configuration in yaml format",
		Long: `
Dump the parsed configuration in yaml format.

Fleet retrieves configuration options from many locations, and it can be
useful to see the result of merging those configs.

The following precedence is used when reading configs:
1. CLI flags
2. Environment Variables
3. Config File
4. Default Values
`,
		Run: func(cmd *cobra.Command, args []string) {
			buf, err := yaml.Marshal(configManager.LoadConfig())
			if err != nil {
				initFatal(err, "marshalling config to yaml")
			}
			fmt.Println(string(buf))
		}}

	return configDumpCmd
}
