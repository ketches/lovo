package consts

const (
	LovoStorageClassName           = "lovo"
	LovoPersistentVolumePathPrefix = "/var/lib/lovo"

	HostnameSelectorKey = "kubernetes.io/hostname"

	LovoPVCNamespaceLabel = "lovo.ketches.cn/pvc-namespace"
	LovoPVCNameLabel      = "lovo.ketches.cn/pvc-name"
	LovoPVNodeAnnotation  = "lovo.ketches.cn/node"
	LovoPVPathAnnotation  = "lovo.ketches.cn/path"
)
