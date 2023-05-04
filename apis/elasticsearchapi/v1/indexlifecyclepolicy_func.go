package v1

// GetIndexLifecyclePolicyName return the index lifecycle policy name
// If name is empty, it use the ressource name
func (o *IndexLifecyclePolicy) GetIndexLifecyclePolicyName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
