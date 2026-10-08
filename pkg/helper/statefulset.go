package helper

import (
	appv1 "k8s.io/api/apps/v1"
)

// IsOnStatefulSetUpgradeState return false if statefulset not to be currently upgraded
func IsOnStatefulSetUpgradeState(o *appv1.StatefulSet) bool {
	if o == nil {
		return false
	}

	if o.Generation != o.Status.ObservedGeneration {
		return true
	}

	if o.Status.CurrentRevision != o.Status.UpdateRevision {
		return true
	}

	expectedReplica := o.Status.Replicas
	if o.Spec.Replicas != nil {
		expectedReplica = *o.Spec.Replicas
	}

	if expectedReplica != o.Status.ReadyReplicas {
		return true
	}

	if o.Status.UpdatedReplicas != expectedReplica {
		return true
	}

	return false
}
