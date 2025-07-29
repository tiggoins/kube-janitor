package types

import (
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type ContainerRuntime string

type OrphanType string

const (
	Docker     ContainerRuntime = "docker"
	Containerd ContainerRuntime = "containerd"

	OrphanBusiness OrphanType = "orphan-business"
	OrphanAPI      OrphanType = "orphan-api"
	OrphanPause    OrphanType = "orphan-pause"
)

var (
	dockerSocket     = "/var/run/docker.sock"
	containerdSocket = "/run/containerd/containerd.sock"
)

type ContainerInfo struct {
	ID        string
	Labels    map[string]string
	Runtime   string
	IsPause   bool
	SandboxID string
	PodUID    string
	ExitTime  *time.Time
}

type OrphanChecker struct {
	Runtime       ContainerRuntime
	K8sClient     *kubernetes.Clientset
	Logger        *logrus.Logger
	Containers    map[string]ContainerInfo // containerID -> info
	SandboxMap    map[string]string        // podUID -> sandboxID
	K8sPodUIDSet  map[string]bool
	Options       Options
}
