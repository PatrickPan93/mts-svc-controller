package controller

const (
	CNCFV1NetworkKey          = "k8s.v1.cni.cncf.io/networks"
	CNCFV1NetworkStatusKey    = "k8s.v1.cni.cncf.io/networks-status"
	CNCFV1NetworkMultusEnable = "k8s.v1.cni.cncf.io/multus-service-enable"
	AddEvent                  = "add"
	UpdateEvent               = "update"
	DeleteEvent               = "delete"
	EndPoints                 = "endpoints"
	Service                   = "Service"
	keyLengthShouldBe         = 4
	SvcNamePrefix             = "multus-service-"
)
