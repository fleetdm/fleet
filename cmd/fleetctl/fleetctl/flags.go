package fleetctl

import (
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/urfave/cli/v2"
)

const (
	apiOnlyFlagName          = "api-only"
	csvFlagName              = "csv"
	debugFlagName            = "debug"
	disableLogTopicsFlagName = "disable-log-topics"
	emailFlagName            = "email"
	enableLogTopicsFlagName  = "enable-log-topics"
	fleetCertificateFlagName = "fleet-certificate"
	fleetFlagName            = "fleet"
	globalRoleFlagName       = "global-role"
	mfaFlagName              = "mfa"
	nameFlagName             = "name"
	outfileFlagName          = "outfile"
	passwordFlagName         = "password"
	ssoFlagName              = "sso"
	stdoutFlagName           = "stdout"
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

func getEnabledLogTopics(c *cli.Context) string {
	return c.String(enableLogTopicsFlagName)
}

func disableLogTopicsFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    disableLogTopicsFlagName,
		EnvVars: []string{"FLEET_DISABLE_LOG_TOPICS"},
		Usage:   "Comma-separated log topics to disable",
	}
}

func getDisabledLogTopics(c *cli.Context) string {
	return c.String(disableLogTopicsFlagName)
}

// withLogTopicFlags adds enable/disable log topic flags and a Before hook
// to each subcommand. If a subcommand already has a Before hook, it is
// wrapped so that applyLogTopicFlags runs first.
func withLogTopicFlags(cmds []*cli.Command) []*cli.Command {
	for _, cmd := range cmds {
		cmd.Flags = append(cmd.Flags, enableLogTopicsFlag(), disableLogTopicsFlag())
		origBefore := cmd.Before
		cmd.Before = func(c *cli.Context) error {
			applyLogTopicFlags(c)
			if origBefore != nil {
				return origBefore(c)
			}
			return nil
		}
	}
	return cmds
}

// applyLogTopicFlags parses the enable/disable log topic flags and applies them.
// Enables run first, then disables, so disable wins on conflict.
func applyLogTopicFlags(c *cli.Context) {
	for _, topic := range str.SplitAndTrim(getEnabledLogTopics(c), ",", true) {
		logging.EnableTopic(topic)
	}
	for _, topic := range str.SplitAndTrim(getDisabledLogTopics(c), ",", true) {
		logging.DisableTopic(topic)
	}
}

// rawArgs returns the original command-line arguments (excluding the program
// name) from the App metadata. This is set by main and test helpers via
// StashRawArgs so that deprecation checks work correctly in both real
// invocations and tests (where os.Args points at the test binary).
func rawArgs(c *cli.Context) []string {
	if md := c.App.Metadata; md != nil {
		if args, ok := md["rawArgs"].([]string); ok {
			return args
		}
	}
	return os.Args[1:]
}

// StashRawArgs saves the raw invocation arguments (excluding the program name)
// into app.Metadata so that deprecation helpers can inspect the original
// command line. Call this before app.Run.
func StashRawArgs(app *cli.App, args []string) {
	if app.Metadata == nil {
		app.Metadata = map[string]any{}
	}
	if len(args) > 1 {
		app.Metadata["rawArgs"] = args[1:]
	}
}

// logDeprecatedCommandName checks if a deprecated command name was used in the
// invocation and prints a warning to stderr. Flag arguments (starting with "-")
// are skipped so that command names appearing after flags are still detected.
func logDeprecatedCommandName(c *cli.Context, deprecatedNames []string, newName string) {
	if !logging.TopicEnabled(logging.DeprecatedFieldTopic) {
		return
	}
	for _, arg := range rawArgs(c) {
		// Assume that command names appear before any flags, so stop checking once we see a flag.
		if strings.HasPrefix(arg, "-") {
			break
		}
		for _, dep := range deprecatedNames {
			if arg == dep {
				fmt.Fprintf(c.App.ErrWriter, "[!] 'fleetctl %s' is deprecated; use '%s' instead\n", dep, newName)
				return
			}
		}
	}
}

// logDeprecatedFlagName checks if a deprecated flag name was used in the
// invocation and prints a warning to stderr. It matches "--name",
// "-name", "--name=value", and "-name=value" forms.
func logDeprecatedFlagName(c *cli.Context, deprecatedName, newName string) {
	if !logging.TopicEnabled(logging.DeprecatedFieldTopic) {
		return
	}
	doubleDash := "--" + deprecatedName
	singleDash := "-" + deprecatedName
	for _, arg := range rawArgs(c) {
		if arg == doubleDash || strings.HasPrefix(arg, doubleDash+"=") ||
			arg == singleDash || strings.HasPrefix(arg, singleDash+"=") {
			fmt.Fprintf(c.App.ErrWriter, "[!] '--%s' is deprecated; use '--%s' instead\n", deprecatedName, newName)
			return
		}
	}
}

// logDeprecatedEnvVar checks if a deprecated environment variable is set and
// prints a warning to stderr suggesting the new name.
func logDeprecatedEnvVar(c *cli.Context, deprecatedEnv, newEnv string) {
	if !logging.TopicEnabled(logging.DeprecatedFieldTopic) {
		return
	}
	if _, ok := os.LookupEnv(deprecatedEnv); ok {
		fmt.Fprintf(c.App.ErrWriter, "[!] '%s' is deprecated; use '%s' instead\n", deprecatedEnv, newEnv)
	}
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
