package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
)

func queryCommand() *cli.Command {
	var (
		flHosts, flLabels, flQuery, flQueryName string
		flQuiet, flExit, flPretty               bool
		flTimeout                               time.Duration
	)
	return &cli.Command{
		Name:      "query",
		Usage:     "Run a live query",
		UsageText: `fleetctl query [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "hosts",
				EnvVars:     []string{"HOSTS"},
				Value:       "",
				Destination: &flHosts,
				Usage:       "Comma-separated hosts to target. Hosts can be specified by hostname, UUID, or serial number.",
			},
			&cli.StringFlag{
				Name:        "labels",
				EnvVars:     []string{"LABELS"},
				Value:       "",
				Destination: &flLabels,
				Usage:       "Comma-separated label names to target",
			},
			&cli.BoolFlag{
				Name:        "quiet",
				EnvVars:     []string{"QUIET"},
				Destination: &flQuiet,
				Usage:       "Only print results (no status information)",
			},
			&cli.BoolFlag{
				Name:        "exit",
				EnvVars:     []string{"EXIT"},
				Destination: &flExit,
				Usage:       "Exit when 100% of online hosts have results returned",
			},
			&cli.StringFlag{
				Name:        "query",
				EnvVars:     []string{"QUERY"},
				Value:       "",
				Destination: &flQuery,
				Usage:       "Query to run",
			},
			&cli.StringFlag{
				Name:        "query-name",
				EnvVars:     []string{"QUERYNAME"},
				Value:       "",
				Destination: &flQueryName,
				Usage:       "Name of saved query to run",
			},
			&cli.BoolFlag{
				Name:        "pretty",
				EnvVars:     []string{"PRETTY"},
				Destination: &flPretty,
				Usage:       "Enable pretty-printing",
			},
			&cli.DurationFlag{
				Name:        "timeout",
				EnvVars:     []string{"TIMEOUT"},
				Destination: &flTimeout,
				Usage:       "How long to run query before exiting (10s, 1h, etc.)",
			},
			&cli.UintFlag{
				Name:  teamFlagName,
				Usage: "ID of the team where the named query belongs to (0 means global)",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if flHosts == "" && flLabels == "" {
				return errors.New("No hosts or labels targeted. Please provide either --hosts or --labels.")
			}

			if flQuery != "" && flQueryName != "" {
				return errors.New("--query and --query-name must not be provided together")
			}

			var queryID *uint
			if flQueryName != "" {
				var teamID *uint
				if tid := c.Uint(teamFlagName); tid != 0 {
					teamID = &tid
				}
				queries, err := client.GetQueries(teamID, &flQueryName)
				if err != nil || len(queries) == 0 {
					return fmt.Errorf("Query '%s' not found", flQueryName)
				}
				// For backwards compatibility with older fleet server, we explicitly find the query in the result array
				for _, query := range queries {
					if query.Name == flQueryName {
						id := query.ID // making an explicit copy of ID
						queryID = &id
						break
					}
				}
				if queryID == nil {
					return fmt.Errorf("Query '%s' not found", flQueryName)
				}
			} else if flQuery == "" {
				return errors.New("Query must be specified with --query or --query-name")
			}

			var output outputWriter
			if flPretty {
				output = newPrettyWriter()
			} else {
				output = newJsonWriter(c.App.Writer)
			}

			hostIdentifiers := strings.Split(flHosts, ",")
			labels := strings.Split(flLabels, ",")

			res, err := client.LiveQuery(flQuery, queryID, labels, hostIdentifiers)
			if err != nil {
				if strings.Contains(err.Error(), "no hosts targeted") {
					return errors.New(fleet.NoHostsTargetedErrMsg)
				}
				if strings.Contains(err.Error(), fleet.InvalidLabelSpecifiedErrMsg) {
					regex := regexp.MustCompile(`(\[.*?\])`)
					match := regex.FindString(err.Error())
					return errors.New(fmt.Sprintf("%s %s", match, fleet.InvalidLabelSpecifiedErrMsg))
				}
				return err
			}

			tick := time.NewTicker(100 * time.Millisecond)
			defer tick.Stop()

			// See charsets at
			// https://godoc.org/github.com/briandowns/spinner#pkg-variables
			s := spinner.New(spinner.CharSets[24], 200*time.Millisecond)
			s.Writer = os.Stderr
			if flQuiet {
				s.Writer = io.Discard
			}
			s.Start()

			var timeoutChan <-chan time.Time
			if flTimeout > 0 {
				timeoutChan = time.After(flTimeout)
			} else {
				// Channel that never fires (so that we can
				// read from the channel in the below select
				// statement without panicking)
				timeoutChan = make(chan time.Time)
			}

			for {
				select {
				// Print a result
				case hostResult := <-res.Results():
					s.Stop()

					if err := output.WriteResult(hostResult); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing result: %s\n", err)
					}

					s.Start()

				// Print an error
				case err := <-res.Errors():
					fmt.Fprintf(os.Stderr, "Error talking to server: %s\n", err.Error())

				// Update status message on interval
				case <-tick.C:
					status := res.Status()
					totals := res.Totals()
					var percentTotal, percentOnline float64
					var responded, total, online uint
					if status != nil && totals != nil {
						total = totals.Total
						online = totals.Online
						responded = status.ActualResults
						if total > 0 {
							percentTotal = 100 * float64(responded) / float64(total)
						}
						if online > 0 {
							percentOnline = 100 * float64(responded) / float64(online)
						}
					}

					msg := fmt.Sprintf(" %.f%% responded (%.f%% online) | %d/%d targeted hosts (%d/%d online)", percentTotal, percentOnline, responded, total, responded, online)

					s.Lock()
					s.Suffix = msg
					s.Unlock()

					if total == responded && status != nil {
						s.Stop()
						if !flQuiet {
							fmt.Fprintln(os.Stderr, msg)
						}
						return nil
					}

					if status != nil && totals != nil && responded >= online && flExit {
						s.Stop()
						if !flQuiet {
							fmt.Fprintln(os.Stderr, msg)
						}
						return nil
					}

				// Check for timeout expiring
				case <-timeoutChan:
					s.Stop()
					if !flQuiet {
						fmt.Fprintln(os.Stderr, s.Suffix+"\nStopped by timeout")
					}
					return nil
				}
			}
		},
	}
}
