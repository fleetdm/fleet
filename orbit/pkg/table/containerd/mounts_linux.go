//go:build linux

package containerd

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// MountsColumns is the schema of the containerd_mounts table.
func MountsColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("namespace"),
		table.TextColumn("container_id"),
		table.TextColumn("type"),
		table.TextColumn("source"),
		table.TextColumn("destination"),
		table.TextColumn("options"),
	}
}

// GenerateMounts is called to return the results for the containerd_mounts table at query time.
// Constraints for generating can be retrieved from the queryContext.
func GenerateMounts(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
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
					row := map[string]string{
						"namespace":    namespace,
						"container_id": container.ID(),
						"type":         mount.Type,
						"source":       mount.Source,
						"destination":  mount.Destination,
						"options":      strings.Join(mount.Options, ","),
					}
					rows = append(rows, row)
				}
			}
		}
	}

	return rows, nil
}
