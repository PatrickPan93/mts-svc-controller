package controller

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"strings"
)

func (c *Controller) HandleSvcAddEvent(namespace, name string) bool {

	svc, err := c.svcLister.Services(namespace).Get(name)
	if err != nil {
		klog.Errorf("Error while handling svc object: namespace(%s) name(%s) %s", namespace, name, err)
		return false
	}
	if svc.Annotations[CNCFV1NetworkMultusEnable] == "false" {
		return true
	}

	multusSvc := generateMultusSvcBaseOnOriginSvc(svc)

	subFixSet := c.getDeviceNamesBySvc(svc)

	if len(subFixSet) == 0 {
		klog.Errorf("Error while get subFix by svc selector(subFixSet not found): namespace(%s) name(%s)", namespace, name)
		return false
	}

	ownerReferences := make([]metav1.OwnerReference, 0)

	var yes = true

	ownerReference := metav1.OwnerReference{
		APIVersion:         v1.SchemeGroupVersion.String(),
		Kind:               Service,
		Name:               svc.Name,
		UID:                svc.UID,
		Controller:         &yes,
		BlockOwnerDeletion: &yes,
	}

	ownerReferences = append(ownerReferences, ownerReference)

	subFixSet = removeDuplicationSlice(subFixSet)

	originName := multusSvc.GetName()

	multusSvc.SetOwnerReferences(ownerReferences)

	for _, subFix := range subFixSet {
		multusSvc.Name = originName + "-" + subFix
		_, err = c.clientSet.CoreV1().Services(namespace).Create(context.Background(), multusSvc, metav1.CreateOptions{})
		if errors.IsAlreadyExists(err) {
			klog.Infof("svc %s exists", multusSvc.Name)
			return true
		}
		if err != nil {
			klog.Errorf(err.Error())
			return false
		}
		klog.Infof("svc %s created", multusSvc.Name)
	}

	return true
}

func (c *Controller) HandleSvcUpdateEvent(namespace, name string) bool {
	svc, err := c.svcLister.Services(namespace).Get(name)
	if err != nil {
		klog.Errorf("Error while handling svc object: namespace(%s) name(%s) %s", namespace, name, err)
		return false
	}

	if svc.Annotations[CNCFV1NetworkMultusEnable] == "false" {
		devices := c.getDeviceNamesBySvc(svc)
		for _, device := range devices {
			multusSvcName := SvcNamePrefix + name + "-" + device
			err := c.clientSet.CoreV1().Services(namespace).Delete(context.Background(), multusSvcName, metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			if err != nil {
				klog.Errorf("Error while deleting multus svc: %s", multusSvcName)
				return false
			}

		}
		return true
	}

	newSvc := svc.DeepCopy()

	multusSvc := generateMultusSvcBaseOnOriginSvc(newSvc)

	oldMultusSvc, err := c.svcLister.Services(namespace).Get(multusSvc.GetName())

	if errors.IsNotFound(err) {
		return c.HandleSvcAddEvent(namespace, name)
	}

	if err != nil {
		klog.Errorf("Error while handling svc object: namespace(%s) name(%s) %s", namespace, name, err)
		return false
	}

	multusSvc.SetUID(oldMultusSvc.GetUID())
	multusSvc.SetResourceVersion(oldMultusSvc.GetResourceVersion())

	_, err = c.clientSet.CoreV1().Services(namespace).Update(context.Background(), multusSvc, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Error while handling svc object: namespace(%s) name(%s) %s", namespace, name, err)
		return false
	}
	klog.Infof("svc %s updated", multusSvc.Name)
	return true
}

func (c *Controller) HandleSvcDeleteEvent(namespace, name string) bool {

	svc, err := c.svcLister.Services(namespace).Get(name)
	if err != nil {
		klog.Errorf("Error while handling svc object: namespace(%s) name(%s) %s", namespace, name, err)
		return false
	}

	subFixSet := c.getDeviceNamesBySvc(svc)
	for _, subFix := range subFixSet {
		var multusSvcName string
		multusSvcName = SvcNamePrefix + "-" + name + "-" + subFix

		_, err := c.svcLister.Services(namespace).Get(multusSvcName)
		if errors.IsNotFound(err) {
			return true
		}

		err = c.clientSet.CoreV1().Services(namespace).Delete(context.Background(), multusSvcName, metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("Error while handling svc object: namespace(%s) name(%s) %s", namespace, name, err)
			return false
		}
		klog.Infof("svc %s deleted", multusSvcName)
	}
	return true
}

func generateMultusSvcBaseOnOriginSvc(service *v1.Service) *v1.Service {
	multusSvc := service.DeepCopy()
	multusSvc.Name = SvcNamePrefix + service.GetName()
	delete(multusSvc.Annotations, CNCFV1NetworkMultusEnable)
	multusSvc.Spec.ClusterIP = ""
	multusSvc.Spec.ClusterIPs = nil
	multusSvc.ResourceVersion = ""
	multusSvc.Spec.Selector = nil
	if multusSvc.Spec.Type == v1.ServiceTypeNodePort {
		klog.Infof("NodePort is not supported now, try to change into ClusterIP type: %s/%s", multusSvc.GetNamespace(), multusSvc.GetName())
		multusSvc.Spec.Type = v1.ServiceTypeClusterIP
		for i, _ := range multusSvc.Spec.Ports {

			if multusSvc.Spec.Ports[i].NodePort != 0 {
				multusSvc.Spec.Ports[i].NodePort = 0
			}
		}

		multusSvc.Spec.ExternalTrafficPolicy = ""
	}

	return multusSvc
}

func (c *Controller) getDeviceNamesBySvc(service *v1.Service) []string {

	/*
		_, err = c.epsLister.Endpoints(namespace).Get(multusSvc.Name)
		if errors.IsNotFound(err) {
			selector := labels.SelectorFromSet(multusSvc.Spec.Selector)
			pods, err := c.podLister.Pods(namespace).List(selector)
			if err != nil {
				klog.Errorf("Error while matching pods by label %s", selector.String())
			}

	*/
	name := service.GetName()
	ns := service.GetNamespace()

	selector := labels.SelectorFromSet(service.Spec.Selector)

	pods, err := c.podLister.Pods(ns).List(selector)

	if err != nil {
		klog.Errorf("Error while get deviceNames by svc selector: namespace(%s) name(%s) %s", ns, name, err)
		return nil
	}

	subFixSet := make([]string, 0)

	for _, pod := range pods {
		annotations := pod.GetAnnotations()
		if configs, ok := annotations[CNCFV1NetworkKey]; !ok || len(configs) == 0 {
			klog.Infof("Pod %s annotations value %s is not valid: %s", pod.Name, CNCFV1NetworkKey, configs)
			continue
		}
		configs := strings.Split(annotations[CNCFV1NetworkKey], ",")
		for _, config := range configs {
			configSlice := strings.Split(config, "@")
			if len(configSlice) < 2 {
				klog.Infof("the config format expect net-attach-def@device, but current is %s, ignored..", configSlice)
				continue
			}
			subFixSet = append(subFixSet, strings.Split(config, "@")[1])
		}
	}
	return subFixSet
}

func removeDuplicationSlice(arr []string) []string {
	set := make(map[string]struct{}, len(arr))
	j := 0
	for _, v := range arr {
		_, ok := set[v]
		if ok {
			continue
		}
		set[v] = struct{}{}
		arr[j] = v
		j++
	}

	return arr[:j]
}
