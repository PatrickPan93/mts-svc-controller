package controller

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

func NewController(factory informers.SharedInformerFactory, clientSet *kubernetes.Clientset) *Controller {
	return &Controller{
		// Lister
		podLister: factory.Core().V1().Pods().Lister(),
		svcLister: factory.Core().V1().Services().Lister(),
		epsLister: factory.Core().V1().Endpoints().Lister(),
		// Informer
		svcInformer: factory.Core().V1().Services().Informer(),
		podInformer: factory.Core().V1().Pods().Informer(),
		epsInformer: factory.Core().V1().Endpoints().Informer(),
		// 创建队列
		workQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		clientSet: clientSet,
	}
}

func (c *Controller) firstHandleAndEnqueue(obj interface{}, eventType string, resourceType string) {

	key, err := cache.MetaNamespaceKeyFunc(obj)
	klog.Infof("svc %s event: %s", eventType, key)
	if err != nil {
		klog.Errorf("svc %s event: %s", eventType, err)
		return
	}
	if !annotationIsValid(obj.(*v1.Service)) {
		klog.Infof("drop svc %s event due to annotation is not valid", eventType)
		return
	}
	customKey := key + "/" + eventType + "/" + resourceType
	klog.Infof("svc %s enqueue", key)

	c.workQueue.Add(customKey)
}

func (c *Controller) AddEventHandlers() {

	c.svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.firstHandleAndEnqueue(obj, AddEvent, v1.ResourceServices.String())

		},
		UpdateFunc: func(old interface{}, new interface{}) {

			if objectChanged(old, new) {
				c.firstHandleAndEnqueue(new, UpdateEvent, v1.ResourceServices.String())
			}
		},
		DeleteFunc: func(obj interface{}) {
			klog.Info("multus service %s would be deleted base on ownReference", obj.(*v1.Service).Name)
		},
	})

	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})
	c.epsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})
}

func objectChanged(previous, current interface{}) bool {
	prev := previous.(metav1.Object)
	cur := current.(metav1.Object)

	return prev.GetResourceVersion() != cur.GetResourceVersion()
}
