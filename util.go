package main

import (
	eksauth "github.com/chankh/eksutil/pkg/auth"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	controllerutils "k8s.io/kubernetes/pkg/controller"
)

type EksHandler struct {
	ClusterName string
	cs          *clientset.Clientset
}

func NewEksHandler(clusterName string) (*EksHandler, error) {
	config := &eksauth.ClusterConfig{
		ClusterName: clusterName,
	}

	cs, err := eksauth.NewAuthClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get EKS authenticated client")
	}

	return &EksHandler{
		ClusterName: clusterName,
		cs:          cs,
	}, nil
}

func (h *EksHandler) GetNodes() (*v1.NodeList, error) {
	// Call Kubernetes API here
	clientset := h.cs
	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Fatal("Error listing pods")
	}

	var results []string

	for i, node := range nodes.Items {
		log.Infof("[%d] %s", i, node.Name)
		results = append(results, node.Name)
	}
	log.Infof("got all results: %v", results)
	return nodes, err
}

func (h *EksHandler) GetPods() (*v1.PodList, error) {
	// Call Kubernetes API here
	clientset := h.cs
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Fatal("Error listing pods")
	}

	var results []string

	for i, pod := range pods.Items {
		log.Infof("[%d] %s", i, pod.Name)
		results = append(results, pod.Name)
	}
	log.Infof("got all results: %v", results)
	return pods, err
}

func (h *EksHandler) TaintNode(t *v1.Taint, nodeName string) error {
	log.Infof("Tainting on node %s", nodeName)
	return controllerutils.AddOrUpdateTaintOnNode(h.cs, nodeName, t)
}
