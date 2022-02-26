package controller

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

// syncToStdout 是控制器的业务逻辑实现
// 在此控制器中，它只是将有关 Pod 的信息打印到 stdout
// 如果发生错误，则简单地返回错误
// 此外重试逻辑不应成为业务逻辑的一部分。

// 检查是否发生错误，并确保我们稍后重试
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// 忘记每次成功同步时 key 的#AddRateLimited历史记录。
		// 这样可以确保不会因过时的错误历史记录而延迟此 key 更新的以后处理。
		c.workQueue.Forget(key)
		return
	}
	if errors.IsNotFound(err) {
		klog.Infof("svc not found %v: %v", key, err)
		klog.Infof("Error syncing svc %v: %v", key, err)
		return
	}
	if errors.IsAlreadyExists(err) {
		klog.Infof("object %s is already exists", key)
		return
	}
	//如果出现问题，此控制器将重试5次
	if c.workQueue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing svc %v: %v", key, err)
		// 重新加入 key 到限速队列
		// 根据队列上的速率限制器和重新入队历史记录，稍后将再次处理该 key
		c.workQueue.AddRateLimited(key)
		return
	}
	c.workQueue.Forget(key)
	// 多次重试，我们也无法成功处理该key
	runtime.HandleError(err)
	klog.Infof("Dropping object %q out of the queue: %v", key, err)
}

// Run 开始 watch 和同步
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// 停止控制器后关掉队列
	defer c.workQueue.ShutDown()

	klog.Info("Starting Pod controller")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping Pod controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
func (c *Controller) processNextItem() bool {
	// 等到工作队列中有一个新元素
	//klog.Info("Starting processing item from queue")

	key, quit := c.workQueue.Get()

	if quit {
		return false
	}

	// 告诉队列我们已经完成了处理此 key 的操作
	// 这将为其他 worker 解锁该 key
	// 这将确保安全的并行处理，因为永远不会并行处理具有相同 key 的两个Pod
	defer c.workQueue.Done(key)

	// 调用包含业务逻辑的方法
	strKey := key.(string)
	strSlice := strings.Split(strKey, "/")
	if len(strSlice) != 4 {
		klog.Info("key length is not %s", keyLengthShouldBe)
		return false
	}
	//klog.Info(strSlice)
	ns, name, eventType, kind := strSlice[0], strSlice[1], strSlice[2], strSlice[3]

	switch eventType {
	case AddEvent:
		switch kind {
		case v1.ResourceServices.String():
			c.handleErr(c.HandleSvcAddEvent(ns, name), key)
		case v1.ResourcePods.String():
		case EndPoints:
		}
	case UpdateEvent:
		switch kind {
		case v1.ResourceServices.String():
			c.handleErr(c.HandleSvcUpdateEvent(ns, name), key)
		}
	case DeleteEvent:
		switch kind {
		case v1.ResourceServices.String():
			return c.HandleSvcDeleteEvent(ns, name)
		}
	default:
		klog.Info("key %s should not appear here", strKey)
	}
	// TODO handle error enqueue
	// 如果在执行业务逻辑期间出现错误，则处理错误
	//c.handleErr(err, key)
	return true
}

func annotationIsValid(obj interface{}) bool {
	switch obj.(type) {
	case *v1.Service:
		if isEnable, ok := obj.(*v1.Service).Annotations[CNCFV1NetworkMultusEnable]; ok && isEnable == "true" || isEnable == "false" {
			return true
		}
	}
	/*
		case *v1.Pod:
			if configs, ok := obj.(*v1.Pod).Annotations[CNCFV1NetworkKey]; !ok || isEnable != "true" {
				return false
			}

		}

	*/
	return false
}
