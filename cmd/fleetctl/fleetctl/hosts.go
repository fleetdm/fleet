package fleetctl

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	hostsFlagName       = "hosts"
	labelFlagName       = "label"
	statusFlagName      = "status"
	searchQueryFlagName = "search_query"
)

func hostsCommand() *cli.Command {
	return &cli.Command{
		Name:  "hosts",
		Usage: "Manage Fleet hosts",
		Subcommands: []*cli.Command{
			transferCommand(),
			hostDebugLoggingCommand(),
		},
	}
}

func hostDebugLoggingCommand() *cli.Command {
	return &cli.Command{
		Name:      "debug-logging",
		Usage:     "Enable or disable orbit debug logging on a single host",
		UsageText: "Toggles runtime orbit debug logging for the given host. When enabled, orbit picks up the change on its next config poll (up to 30s) without restarting. Osquery is also flipped to verbose when debug is on. Requires admin or maintainer on the host's team.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "host",
				Usage:    "Host identifier (hostname, UUID, serial, or osquery host id)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "enable",
				Usage: "Enable debug logging on the host",
			},
			&cli.BoolFlag{
				Name:  "disable",
				Usage: "Disable debug logging and clear any active host override",
			},
			&cli.StringFlag{
				Name:        "duration",
				Usage:       "How long debug logging stays on (min 1m, max 7d). Only valid with --enable.",
				DefaultText: "24h",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			enable := c.Bool("enable")
			disable := c.Bool("disable")
			durationStr := c.String("duration")

			var duration time.Duration
			if durationStr != "" {
				d, err := parseDurationWithDays(durationStr)
				if err != nil {
					return fmt.Errorf("invalid --duration %q: %w", durationStr, err)
				}
				duration = d
			}

			switch {
			case enable == disable:
				return errors.New("exactly one of --enable or --disable is required")
			case disable && duration != 0:
				return errors.New("--duration cannot be used with --disable")
			case duration < 0:
				return errors.New("--duration must not be negative")
			}

			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			host, err := client.HostByIdentifier(c.String("host"))
			if err != nil {
				return fmt.Errorf("resolve host: %w", err)
			}

			until, err := client.SetHostOrbitDebugLogging(host.ID, enable, duration)
			if err != nil {
				return err
			}

			if enable {
				fmt.Fprintf(c.App.Writer, "Orbit debug logging enabled on host %d until %s (UTC).\n", host.ID, until.Format(time.RFC3339))
			} else {
				fmt.Fprintf(c.App.Writer, "Orbit debug logging disabled on host %d.\n", host.ID)
			}
			return nil
		},
	}
}

func transferCommand() *cli.Command {
	return &cli.Command{
		Name:      "transfer",
		Usage:     "Transfer one or more hosts to a fleet",
		UsageText: `This command will gather the set of hosts specified and transfer them to the fleet.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     fleetFlagName,
				Aliases:  []string{"team"},
				Usage:    "Fleet name hosts will be transferred to. Use '' for unassigned",
				Required: true,
			},
			&cli.StringFlag{
				Name:  hostsFlagName,
				Usage: "Comma-separated hostnames to transfer",
			},
			&cli.StringFlag{
				Name:  labelFlagName,
				Usage: "Label name to transfer",
			},
			&cli.StringFlag{
				Name:  statusFlagName,
				Usage: "Status to use when filtering hosts",
			},
			&cli.StringFlag{
				Name:  searchQueryFlagName,
				Usage: "A search query that returns matching hostnames to be transferred",
			},
			configFlag(),
			contextFlag(),
			yamlFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			team := c.String(fleetFlagName)
			var hosts []string
			if hostsFlag := c.String(hostsFlagName); hostsFlag != "" {
				hosts = strings.Split(hostsFlag, ",")
			}
			label := c.String(labelFlagName)
			status := c.String(statusFlagName)
			searchQuery := c.String(searchQueryFlagName)

			if hosts != nil {
				if label != "" || searchQuery != "" || status != "" {
					return errors.New("--hosts cannot be used along side any other flag")
				}
			} else {
				if label == "" && searchQuery == "" && status == "" {
					return errors.New("You need to define either --hosts, or one or more of --label, --status, --search_query")
				}
			}

			return client.TransferHosts(hosts, label, status, searchQuery, team)
		},
	}
}

// parseDurationWithDays extends time.ParseDuration with a "d" (days) unit so
// values like "7d" or "1d12h" are accepted. Each "Nd" or "N.Nd" span is
// rewritten to the equivalent number of hours before delegating to
// time.ParseDuration, which handles the rest of the units and signs.
func parseDurationWithDays(s string) (time.Duration, error) {
	if s == "" {
		return 0, errors.New("empty duration")
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		j := i
		if s[j] == '+' || s[j] == '-' {
			j++
		}
		start := j
		for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
			j++
		}
		if j > start && j < len(s) && s[j] == 'd' {
			days, err := strconv.ParseFloat(s[start:j], 64)
			if err != nil {
				return 0, err
			}
			if s[i] == '-' {
				days = -days
			}
			b.WriteString(strconv.FormatFloat(days*24, 'f', -1, 64))
			b.WriteByte('h')
			i = j + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return time.ParseDuration(b.String())
}
