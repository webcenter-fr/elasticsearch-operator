package v1alpha1

// GetSnapshotLifecyclePolicyName return the snapshot lifecycle policy name
// If name is empty, it use the ressource name
func (o *SnapshotLifecyclePolicy) GetSnapshotLifecyclePolicyName() string {
	if o.Spec.SnapshotLifecyclePolicyName == "" {
		return o.Name
	}

	return o.Spec.SnapshotLifecyclePolicyName
}
