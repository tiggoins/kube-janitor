package docker

import (
	"context"
	"time"
	"strings"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
)

func listContainersDocker() []ContainerInfo {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil
	}
	var results []ContainerInfo
	for _, cont := range containers {
		labels := cont.Labels
		podUID := labels["io.kubernetes.pod.uid"]
		sandboxID := labels["io.kubernetes.sandbox.id"]
		isPause := strings.Contains(cont.Image, "pause") || labels["io.kubernetes.container.name"] == "POD"

		var exitTime *time.Time
		if cont.State == "exited" {
			inspect, err := cli.ContainerInspect(ctx, cont.ID)
			if err == nil {
				t := inspect.State.FinishedAt
				parsed, err := time.Parse(time.RFC3339Nano, t)
				if err == nil {
					exitTime = &parsed
				}
			}
		}

		results = append(results, ContainerInfo{
			ID:        cont.ID,
			Labels:    labels,
			Runtime:   string(Docker),
			IsPause:   isPause,
			SandboxID: sandboxID,
			PodUID:    podUID,
			ExitTime:  exitTime,
		})
	}
	return results
}
