package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/tiggoins/kube-janitor/types"
)

func main() {
	checker := &types.OrphanChecker{
		Logger:     logrus.New(),
		Containers: make(map[string]types.ContainerInfo),
		SandboxMap: make(map[string]string),
		Options:    ParseOptions(),
	}
	checker.Logger.SetFormatter(&logrus.JSONFormatter{})
	checker.Logger.SetOutput(os.Stdout)
	checker.Logger.SetLevel(logrus.InfoLevel)

	checker.Runtime = detectRuntime()
	checker.K8sClient = getK8sClient(checker.Logger)
	checker.K8sPodUIDSet = checker.getK8sPodUIDSet()

	var containerList []ContainerInfo
	if checker.Runtime == Docker {
		containerList = listContainersDocker()
	} else {
		containerList = listContainersContainerd()
	}

	for _, c := range containerList {
		checker.Containers[c.ID] = c
		if c.IsPause {
			checker.SandboxMap[c.PodUID] = c.ID
		}
	}

	// 删除所有 K8s API 中存在的 PodUID，对剩下的内容进行 orphan 检查
	for podUID := range checker.K8sPodUIDSet {
		delete(checker.SandboxMap, podUID)
	}

	// 剩下 SandboxMap 中的是不在 K8s 中的 podUID
	for podUID, sandboxID := range checker.SandboxMap {
		found := false
		for _, c := range checker.Containers {
			if c.PodUID == podUID && !c.IsPause {
				found = true
				checker.logOrphan(c, OrphanAPI)
			}
		}
		if !found {
			checker.Logger.WithFields(logrus.Fields{
				"type":     OrphanPause,
				"podUID":   podUID,
				"sandboxID": sandboxID,
			}).Warn("Pause container without business container and not found in K8s")
		}
	}
}

func detectRuntime() ContainerRuntime {
	if _, err := os.Stat(dockerSocket); err == nil {
		return Docker
	}
	if _, err := os.Stat(containerdSocket); err == nil {
		return Containerd
	}
	logrus.Fatal("No supported container runtime socket found")
	return ""
}

func getK8sClient(logger *logrus.Logger) *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to get in-cluster config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create K8s client")
	}
	return clientset
}

func (o *OrphanChecker) getK8sPodUIDSet() map[string]bool {
	nodeName := os.Getenv("NODE_NAME")
	pods, err := o.K8sClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		o.Logger.WithError(err).Fatal("Failed to list pods from K8s API")
	}
	podUIDSet := make(map[string]bool)
	for _, pod := range pods.Items {
		podUIDSet[string(pod.UID)] = true
	}
	return podUIDSet
}

func (o *OrphanChecker) logOrphan(c ContainerInfo, orphanType OrphanType) {
	if c.ExitTime != nil && time.Since(*c.ExitTime) > 24*time.Hour {
		o.Logger.WithFields(logrus.Fields{
			"type":        orphanType,
			"containerID": c.ID,
			"podUID":      c.PodUID,
			"sandboxID":   c.SandboxID,
			"exitTime":    c.ExitTime.Format(time.RFC3339),
		}).Warn("Orphan container over 24h")
	} else {
		o.Logger.WithFields(logrus.Fields{
			"type":        orphanType,
			"containerID": c.ID,
			"podUID":      c.PodUID,
			"sandboxID":   c.SandboxID,
		}).Warn("Orphan container")
	}
}

