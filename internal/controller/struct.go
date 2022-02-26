package controller

import (
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type networkStatus struct {
	Name    string   `json:"name"`
	Ips     []string `json:"ips"`
	Default bool     `json:"default,omitempty"`
	Dns     struct {
	} `json:"dns"`
	Interface string `json:"interface,omitempty"`
	Mac       string `json:"mac,omitempty"`
}

type Controller struct {
	podLister   v1.PodLister
	svcLister   v1.ServiceLister
	epsLister   v1.EndpointsLister
	podInformer cache.SharedInformer
	svcInformer cache.SharedInformer
	epsInformer cache.SharedInformer

	workQueue workqueue.RateLimitingInterface
	clientSet *kubernetes.Clientset
}
