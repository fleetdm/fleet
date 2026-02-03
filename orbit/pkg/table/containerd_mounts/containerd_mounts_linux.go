//go:build linux
// +build linux

package containerd_mounts

import (
	"context"
	"fmt"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("namespace"),
		table.TextColumn("container_id"),
		table.TextColumn("type"),
		table.TextColumn("source"),
		table.TextColumn("destination"),
		table.TextColumn("options"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to containerd: %v", err)
	}
	defer client.Close()

	// Get all namespaces so we can iterate over them
	namespacesList, err := client.NamespaceService().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to list namespaces: %v", err)
	}

	rows := []map[string]string{}
	for _, namespace := range namespacesList {
		nsCtx := namespaces.WithNamespace(ctx, namespace)

		containers, err := client.Containers(nsCtx)
		if err != nil {
			return nil, fmt.Errorf("Failed to list containers: %v", err)
		}

		for _, container := range containers {
			// Get the container's spec to access mount information
			spec, err := container.Spec(nsCtx)
			if err != nil {
				log.Printf("Failed to get spec for container %s: %v", container.ID(), err)
				continue
			}

			// Iterate through mounts in the spec
			if spec.Mounts != nil {
				for _, mount := range spec.Mounts {
					// Join options into a comma-separated string
					options := ""
					if len(mount.Options) > 0 {
						for i, opt := range mount.Options {
							if i > 0 {
								options += ","
							}
							options += opt
						}
					}

					row := map[string]string{
						"namespace":    namespace,
						"container_id": container.ID(),
						"type":         mount.Type,
						"source":       mount.Source,
						"destination":  mount.Destination,
						"options":      options,
					}
					rows = append(rows, row)
				}
			}
		}
	}

	return rows, nil
}
