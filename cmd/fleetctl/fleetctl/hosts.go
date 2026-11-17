package fleetctl

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
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

			err = client.TransferHosts(hosts, label, status, searchQuery, team)
			var sce kithttp.StatusCoder
			if hosts != nil && errors.As(err, &sce) && sce.StatusCode() == http.StatusForbidden {
				// GitOps users can't look up hosts by hostname, so they're denied before the transfer request.
				if me, meErr := client.Me(); meErr == nil && isGitOpsUser(me) {
					return errors.New(fleet.GitOpsTransferHostErrorMessage)
				}
			}
			return err
		},
	}
}
