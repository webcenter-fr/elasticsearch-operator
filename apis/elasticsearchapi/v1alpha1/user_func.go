package v1alpha1

// IsProtected return true if user is protected
func (h *User) IsProtected() bool {
	if h.Spec.IsProtected != nil && *h.Spec.IsProtected {
		return true
	}

	return false
}
