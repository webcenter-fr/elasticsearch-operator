package v1

// GetUserSpaceID return the user space ID
// If name is empty, it use the ressource name
func (o *UserSpace) GetUserSpaceID() string {
	if o.Spec.ID == "" {
		return o.Name
	}

	return o.Spec.ID
}

// IsIncludeReferences return true if include reference
func (o KibanaUserSpaceCopy) IsIncludeReference() bool {
	if o.IncludeReferences == nil || *o.IncludeReferences {
		return true
	}

	return false
}

// IsOverwrite is true if overwrite
func (o KibanaUserSpaceCopy) IsOverwrite() bool {
	if o.Overwrite == nil || *o.Overwrite {
		return true
	}

	return false
}

// IsCreateNewCopy is true if create new copy
func (o KibanaUserSpaceCopy) IsCreateNewCopy() bool {
	if o.CreateNewCopies != nil && *o.CreateNewCopies {
		return true
	}

	return false
}

// IsForceUpdate is true if force update
func (o KibanaUserSpaceCopy) IsForceUpdate() bool {
	if o.ForceUpdateWhenReconcile != nil && *o.ForceUpdateWhenReconcile {
		return true
	}

	return false
}
