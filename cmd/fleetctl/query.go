package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

func queryCommand() *cli.Command {
	var (
		flHosts, flLabels, flQuery, flQueryName, flResultsQuery string
		flQuiet, flExit, flPretty                               bool
		flTimeout                                               time.Duration
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
			&cli.StringFlag{
				Name:        "results-query",
				Destination: &flResultsQuery,
				Usage:       "Query to run on results of query",
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

			if flQueryName != "" {
				q, err := client.GetQuery(flQueryName)
				if err != nil {
					return fmt.Errorf("Query '%s' not found", flQueryName)
				}
				flQuery = q.Query
			}

			if flQuery == "" {
				return errors.New("Query must be specified with --query or --query-name")
			}

			var writer outputWriter
			if flPretty {
				writer = newPrettyWriter()
			} else {
				writer = newJsonWriter(c.App.Writer)
			}

			hosts := strings.Split(flHosts, ",")
			labels := strings.Split(flLabels, ",")

			res, err := client.LiveQuery(flQuery, labels, hosts)
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
			}

			// collect results
			var results []fleet.DistributedQueryResult

		OUTER:
			for {
				select {
				// Print a result
				case result := <-res.Results():
					results = append(results, result)
					s.Stop()

					if err := writer.WriteResult(result); err != nil {
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
						break OUTER
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
						break OUTER
					}

				// Check for timeout expiring
				case <-timeoutChan:
					s.Stop()
					if !flQuiet {
						fmt.Fprintln(os.Stderr, s.Suffix+"\nStopped by timeout")
					}
					break OUTER
				}
			}

			// run a query on the results
			if flResultsQuery != "" {
				db, err := sqlx.Open("sqlite3", ":memory:")
				if err != nil {
					return err
				}

				// create a table for results
				row := results[0].Rows[0]

				var columns []string
				for column, _ := range row {
					columns = append(columns, column)
				}
				sort.Strings(columns)

				sql := `CREATE TABLE results (host_id TEXT, `
				for i, column := range columns {
					if i > 0 {
						sql += ` TEXT, `
					}
					sql += column
				}
				sql += ` TEXT)`

				_, err = db.Exec(sql)
				if err != nil {
					return err
				}

				// insert the results
				sql = `INSERT INTO results (host_id, `
				sql += strings.Join(columns, ", ")
				sql += `) VALUES `

				placeholders := `(` + strings.Repeat("?, ", len(columns)+1)
				placeholders = placeholders[:len(placeholders)-2] + ")"

				var args []interface{}
				for i, result := range results {
					for j, row := range result.Rows {
						if i+j > 0 {
							sql += `, `
						}
						sql += placeholders
						args = append(args, result.Host.ID)
						for _, column := range columns {
							args = append(args, row[column])
						}
					}
				}

				_, err = db.Exec(sql, args...)
				if err != nil {
					return err
				}

				fmt.Println("--------")

				// finally run the query on results
				rows, err := db.Queryx(flResultsQuery)
				if err != nil {
					return err
				}
				defer rows.Close()

				for rows.Next() {
					results := make(map[string]interface{})
					err := rows.MapScan(results)
					if err != nil {
						return err
					}
					resultsJSON, err := json.Marshal(results)
					if err != nil {
						return err
					}
					fmt.Println(string(resultsJSON))
				}
				if err := rows.Err(); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
