package fleetctl

import (
	"strings"

	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/urfave/cli/v2"
)

const (
	outfileFlagName          = "outfile"
	debugFlagName            = "debug"
	fleetCertificateFlagName = "fleet-certificate"
	stdoutFlagName           = "stdout"
	enableLogTopicsFlagName  = "enable-log-topics"
	disableLogTopicsFlagName = "disable-log-topics"
)

func outfileFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    outfileFlagName,
		Value:   "",
		EnvVars: []string{"OUTFILE"},
		Usage:   "Path to output file",
	}
}

func getOutfile(c *cli.Context) string {
	return c.String(outfileFlagName)
}

func debugFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:    debugFlagName,
		EnvVars: []string{"DEBUG"},
		Usage:   "Enable debug http request logging",
	}
}

func getDebug(c *cli.Context) bool {
	return c.Bool(debugFlagName)
}

func fleetCertificateFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    fleetCertificateFlagName,
		EnvVars: []string{"FLEET_CERTIFICATE"},
		Usage:   "Path of the TLS fleet certificate, can be used to provide additional connection debugging information",
	}
}

func getFleetCertificate(c *cli.Context) string {
	return c.String(fleetCertificateFlagName)
}

func stdoutFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:    stdoutFlagName,
		EnvVars: []string{"STDOUT"},
		Usage:   "Print contents to stdout",
	}
}

func getStdout(c *cli.Context) bool {
	return c.Bool(stdoutFlagName)
}

func enableLogTopicsFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    enableLogTopicsFlagName,
		EnvVars: []string{"FLEET_ENABLE_LOG_TOPICS"},
		Usage:   "Comma-separated log topics to enable",
	}
}

func getEnableLogTopics(c *cli.Context) string {
	return c.String(enableLogTopicsFlagName)
}

func disableLogTopicsFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    disableLogTopicsFlagName,
		EnvVars: []string{"FLEET_DISABLE_LOG_TOPICS"},
		Usage:   "Comma-separated log topics to disable",
	}
}

func getDisableLogTopics(c *cli.Context) string {
	return c.String(disableLogTopicsFlagName)
}

// applyLogTopicFlags parses the enable/disable log topic flags and applies them.
// Enables run first, then disables, so disable wins on conflict.
func applyLogTopicFlags(c *cli.Context) {
	for _, topic := range parseLogTopicsList(getEnableLogTopics(c)) {
		logging.EnableTopic(topic)
	}
	for _, topic := range parseLogTopicsList(getDisableLogTopics(c)) {
		logging.DisableTopic(topic)
	}
}

// parseLogTopicsList splits a comma-separated string into trimmed, non-empty topic names.
func parseLogTopicsList(s string) []string {
	if s == "" {
		return nil
	}
	var topics []string
	for t := range strings.SplitSeq(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			topics = append(topics, t)
		}
	}
	return topics
}

func byHostIdentifier() cli.Flag {
	return &cli.StringFlag{
		Name:  "host",
		Usage: "Filter MDM commands by host specified by hostname, UUID, or serial number.",
	}
}

func byMDMCommandRequestType() cli.Flag {
	return &cli.StringFlag{
		Name:  "type",
		Usage: "Filter MDM commands by type.",
	}
}

func withMDMCommandStatusFilter() cli.Flag {
	return &cli.StringFlag{
		Name:  "command_status",
		Usage: "Filter MDM commands by command status in a comma-separated list. Valid values are 'pending', 'ran', and 'failed'. ",
	}
}
