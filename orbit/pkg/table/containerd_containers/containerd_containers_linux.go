//go:build linux
// +build linux

package containerd_containers

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("namespace"),
		table.TextColumn("id"),
		table.TextColumn("image"),
		table.TextColumn("image_digest"),
		table.TextColumn("state"),
		table.BigIntColumn("created"),
		table.TextColumn("runtime"),
		table.TextColumn("command"),
		table.BigIntColumn("pid"),
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
			info, err := container.Info(nsCtx)
			if err != nil {
				log.Printf("Failed to get info for container %s: %v", container.ID(), err)
				continue
			}

			// Get image digest if possible
			imageDigest := ""
			img, err := container.Image(nsCtx)
			if err == nil {
				imageDigest = img.Target().Digest.String()
			}

			// Get state and pid from task if possible
			state := "unknown"
			pid := ""
			command := ""
			task, err := container.Task(nsCtx, cio.Load)
			if err == nil {
				status, err := task.Status(nsCtx)
				if err == nil {
					state = string(status.Status)
				}
				taskPid := task.Pid()
				if taskPid > 0 {
					pid = fmt.Sprintf("%d", taskPid)
				}
				// Try to get the command from the process spec
				spec, err := container.Spec(nsCtx)
				if err == nil && spec.Process != nil && len(spec.Process.Args) > 0 {
					command = strings.Join(spec.Process.Args, " ")
				}
			} else {
				state = "stopped"
			}

			row := map[string]string{
				"namespace":    namespace,
				"id":           info.ID,
				"image":        info.Image,
				"image_digest": imageDigest,
				"state":        state,
				"created":      fmt.Sprintf("%d", info.CreatedAt.Unix()),
				"runtime":      info.Runtime.Name,
				"pid":          pid,
				"command":      command,
			}
			rows = append(rows, row)
		}
	}

	return rows, nil
}
