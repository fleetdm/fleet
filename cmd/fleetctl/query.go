package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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
				Usage:       "Comma separated hostnames to target",
			},
			&cli.StringFlag{
				Name:        "labels",
				EnvVars:     []string{"LABELS"},
				Value:       "",
				Destination: &flLabels,
				Usage:       "Comma separated label names to target",
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
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if flHosts == "" && flLabels == "" {
				return errors.New("No hosts or labels targeted")
			}

			if flQuery != "" && flQueryName != "" {
				return errors.New("--query and --query-name must not be provided together")
			}

			if flQueryName != "" {
				q, err := fleet.GetQuery(flQueryName)
				if err != nil {
					return fmt.Errorf("Query '%s' not found", flQueryName)
				}
				flQuery = q.Query
			}

			if flQuery == "" {
				return errors.New("Query must be specified with --query or --query-name")
			}

			var output outputWriter
			if flPretty {
				output = newPrettyWriter()
			} else {
				output = newJsonWriter(c.App.Writer)
			}

			hosts := strings.Split(flHosts, ",")
			labels := strings.Split(flLabels, ",")

			res, err := fleet.LiveQuery(flQuery, labels, hosts)
			if err != nil {
				return err
			}

			tick := time.NewTicker(100 * time.Millisecond)
			defer tick.Stop()

			// See charsets at
			// https://godoc.org/github.com/briandowns/spinner#pkg-variables
			s := spinner.New(spinner.CharSets[24], 200*time.Millisecond)
			s.Writer = os.Stderr
			if flQuiet {
				s.Writer = ioutil.Discard
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

					if responded >= online && flExit {
						return nil
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
