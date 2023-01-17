package v1alpha1

// GetComponentTemplateName return the component template name
// If name is empty, it use the ressource name
func (o *ComponentTemplate) GetComponentTemplateName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
