package v1

// IsProtected return true if user is protected
func (h *User) IsProtected() bool {
	if h.Spec.IsProtected != nil && *h.Spec.IsProtected {
		return true
	}

	return false
}

// GetUsername return the expected user name
// It take ressource name if username is empty
func (h *User) GetUsername() string {
	if h.Spec.Username == "" {
		return h.Name
	}

	return h.Spec.Username
}
