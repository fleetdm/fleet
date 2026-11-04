package fleetctl

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

func deleteCommand() *cli.Command {
	var flFilename string
	return &cli.Command{
		Name:      "delete",
		Usage:     "Specify files to declaratively batch delete osquery configurations",
		UsageText: `fleetctl delete [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "f",
				EnvVars:     []string{"FILENAME"},
				Value:       "",
				Destination: &flFilename,
				Usage:       "A file to apply",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if flFilename == "" {
				return errors.New("-f must be specified")
			}

			b, err := os.ReadFile(flFilename)
			if err != nil {
				return err
			}

			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			specs, err := spec.GroupFromBytes(b)
			if err != nil {
				return err
			}

			for _, query := range specs.Queries {
				fmt.Printf("[+] deleting query %q\n", query.Name)
				if err := fleet.DeleteQuery(query.Name); err != nil {
					root := ctxerr.Cause(err)
					switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
					case service.NotFoundErr:
						fmt.Printf("[!] query %q doesn't exist\n", query.Name)
						continue
					}
					return err
				}
			}

			for _, pack := range specs.Packs {
				fmt.Printf("[+] deleting pack %q\n", pack.Name)
				if err := fleet.DeletePack(pack.Name); err != nil {
					root := ctxerr.Cause(err)
					switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
					case service.NotFoundErr:
						fmt.Printf("[!] pack %q doesn't exist\n", pack.Name)
						continue
					}
					return err
				}
			}

			labelIDs := newLabelIDLookup(fleet)
			for _, label := range specs.Labels {
				fmt.Printf("[+] deleting label %q\n", label.Name)
				id, err := labelIDs.find(label.Name, label.TeamID)
				if err != nil {
					return err
				}
				if id == 0 {
					fmt.Printf("[!] label %q doesn't exist\n", label.Name)
					continue
				}
				if err := fleet.DeleteLabelByID(id); err != nil {
					root := ctxerr.Cause(err)
					switch root.(type) { //nolint:gocritic // ignore singleCaseSwitch
					case service.NotFoundErr:
						fmt.Printf("[!] label %q doesn't exist\n", label.Name)
						continue
					}
					return err
				}
			}

			return nil
		},
	}
}

// Labels are deleted by ID, and fleets can share label names, so a spec's label is looked up by name among its own
// fleet's labels (or the global labels).
type labelIDLookup struct {
	client  *service.Client
	byFleet map[uint][]*fleet.LabelSpec
}

func newLabelIDLookup(client *service.Client) *labelIDLookup {
	return &labelIDLookup{client: client, byFleet: make(map[uint][]*fleet.LabelSpec)}
}

// find returns 0 if there's no such label. Names compare case-insensitively, like in Fleet.
func (l *labelIDLookup) find(name string, fleetID *uint) (uint, error) {
	id := ptr.ValOrZero(fleetID)
	labels, ok := l.byFleet[id]
	if !ok {
		var err error
		if labels, err = l.client.GetLabels(id); err != nil {
			return 0, err
		}
		l.byFleet[id] = labels
	}
	for _, label := range labels {
		if strings.EqualFold(label.Name, name) && ptr.ValOrZero(label.TeamID) == id {
			return label.ID, nil
		}
	}
	return 0, nil
}
