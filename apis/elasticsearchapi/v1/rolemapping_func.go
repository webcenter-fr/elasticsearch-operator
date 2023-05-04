package v1

// GetRoleMappingName return the role mapping name
// If name is empty, it use the ressource name
func (o *RoleMapping) GetRoleMappingName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
