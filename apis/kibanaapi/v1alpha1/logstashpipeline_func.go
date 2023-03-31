package v1alpha1

// GetRoleName return the role name
// If name is empty, it use the ressource name
func (o *LogstashPipeline) GetPipelineName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
