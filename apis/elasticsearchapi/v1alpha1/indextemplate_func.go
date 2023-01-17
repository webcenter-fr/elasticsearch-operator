package v1alpha1

// GetIndexTemplateName return the index template name
// If name is empty, it use the ressource name
func (o *IndexTemplate) GetIndexTemplateName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
