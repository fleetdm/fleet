package fleetctl

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/AbGuthrie/goquery/v2"
	gqconfig "github.com/AbGuthrie/goquery/v2/config"
	gqhosts "github.com/AbGuthrie/goquery/v2/hosts"
	gqmodels "github.com/AbGuthrie/goquery/v2/models"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

type activeQuery struct {
	status  string
	results []map[string]string
}

type goqueryClient struct {
	client       *service.Client
	queryCounter int
	queries      map[string]activeQuery
	// goquery passes the UUID, while we need the hostname (or ID) to
	// query against Fleet. Keep a mapping so that we know how to target
	// the host.
	hostnameByUUID map[string]string
}

func newGoqueryClient(fleetClient *service.Client) *goqueryClient {
	return &goqueryClient{
		client:         fleetClient,
		queryCounter:   0,
		queries:        make(map[string]activeQuery),
		hostnameByUUID: make(map[string]string),
	}
}

func (c *goqueryClient) CheckHost(query string) (gqhosts.Host, error) {
	res, err := c.client.SearchTargets(query, nil, nil)
	if err != nil {
		return gqhosts.Host{}, err
	}

	var host *fleet.Host
	for _, h := range res.Hosts {
		// We allow hosts to be looked up by hostname in addition to UUID
		if query == h.UUID || query == h.Hostname || query == h.ComputerName {
			host = h
			break
		}
	}

	if host == nil {
		return gqhosts.Host{}, fmt.Errorf("host %s not found", query)
	}

	c.hostnameByUUID[host.UUID] = host.Hostname

	return gqhosts.Host{
		UUID:         host.UUID,
		ComputerName: host.ComputerName,
		Platform:     host.Platform,
		Version:      host.OsqueryVersion,
	}, nil
}

func (c *goqueryClient) ScheduleQuery(uuid, query string) (string, error) {
	c.queryCounter++
	queryName := strconv.Itoa(c.queryCounter)

	hostname, ok := c.hostnameByUUID[uuid]
	if !ok {
		return "", errors.New("could not lookup host")
	}

	res, err := c.client.LiveQuery(query, nil, []string{}, []string{hostname})
	if err != nil {
		return "", err
	}

	c.queries[queryName] = activeQuery{status: "Pending"}

	// We need to start a separate thread due to goquery expecting
	// scheduling a query and retrieving results to be separate
	// operations.
	go func() {
		select {
		case hostResult := <-res.Results():
			c.queries[queryName] = activeQuery{status: "Completed", results: hostResult.Rows}

			// Print an error
		case err := <-res.Errors():
			c.queries[queryName] = activeQuery{status: "error: " + err.Error()}
		}
	}()

	gqhosts.AddQueryToHost(uuid, gqhosts.Query{Name: queryName, SQL: query})
	return queryName, nil
}

func (c *goqueryClient) FetchResults(queryName string) (gqmodels.Rows, string, error) {
	res, ok := c.queries[queryName]
	if !ok {
		return nil, "", fmt.Errorf("Unknown query %s", queryName)
	}

	return res.results, res.status, nil
}

func goqueryCommand() *cli.Command {
	return &cli.Command{
		Name:  "goquery",
		Usage: "Start the goquery interface",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			yamlFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			goquery.Run(newGoqueryClient(fleet), gqconfig.Config{})
			return nil
		},
	}
}
