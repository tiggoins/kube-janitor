package containerd

import (
	"context"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
)

func listContainersContainerd() []ContainerInfo {
	client, err := containerd.New(containerdSocket)
	if err != nil {
		return nil
	}
	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")
	containers, err := client.Containers(ctx)
	if err != nil {
		return nil
	}
	var results []ContainerInfo
	for _, c := range containers {
		info, err := c.Info(ctx)
		if err != nil {
			continue
		}
		labels := info.Labels
		podUID := labels["io.kubernetes.pod.uid"]
		sandboxID := labels["io.kubernetes.sandbox.id"]
		isPause := strings.Contains(info.Image, "pause") || labels["io.kubernetes.container.name"] == "POD"

		var exitTime *time.Time
		task, err := c.Task(ctx, nil)
		if err == nil {
			status, err := task.Status(ctx)
			if err == nil && status.Status == containerd.Stopped {
				exitTime = status.ExitTimePtr()
			}
		}
		results = append(results, ContainerInfo{
			ID:        c.ID(),
			Labels:    labels,
			Runtime:   string(Containerd),
			IsPause:   isPause,
			SandboxID: sandboxID,
			PodUID:    podUID,
			ExitTime:  exitTime,
		})
	}
	return results
}
