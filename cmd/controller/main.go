package main

import (
	"flag"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"mts-svc-controller/internal/controller"
	"path/filepath"
	"time"
)

func initClient() (*kubernetes.Clientset, error) {
	var err error
	var config *rest.Config
	// inCluster（Pod）、KubeConfig（kubectl）
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(可选) kubeconfig 文件的绝对路径")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig 文件的绝对路径")
	}
	flag.Parse()

	// 首先使用 inCluster 模式(需要去配置对应的 RBAC 权限，默认的sa是default->是没有获取deployments的List权限)
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 KubeConfig 文件创建集群配置 Config 对象
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	// 已经获得了 rest.Config 对象
	// 创建 Clientset 对象
	return kubernetes.NewForConfig(config)
}

func main() {

	clientSet, err := initClient()
	if err != nil {
		klog.Fatal(err)
	}

	factory := informers.NewSharedInformerFactory(clientSet, 600*time.Second)

	ctl := controller.NewController(factory, clientSet)
	ctl.AddEventHandlers()

	stopCh := make(chan struct{})
	defer close(stopCh)

	// Start & wait for cache sync
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	// start controller
	go ctl.Run(1, stopCh)

	select {}

}
