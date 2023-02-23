package helper

import (
	appv1 "k8s.io/api/apps/v1"
)

// IsOnStatefulSetUpgradeState return false if statefulset not to be currently upgraded
func IsOnStatefulSetUpgradeState(o *appv1.StatefulSet) bool {
	if o == nil {
		return false
	}

	if o.ObjectMeta.Generation != o.Status.ObservedGeneration {
		return true
	}

	if o.Status.CurrentRevision != o.Status.UpdateRevision {
		return true
	}

	if o.Status.Replicas != o.Status.ReadyReplicas {
		return true
	}

	return false

}
