package common

// KubernetesCapability descripte the capability of the current Kubernetes cluster
type KubernetesCapability struct {
	HasRoute      bool
	HasPrometheus bool
}
