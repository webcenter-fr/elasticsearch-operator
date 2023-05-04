package v1

// IsBasicLicense return true if is basic license
func (h *License) IsBasicLicense() bool {
	if (h.Spec.Basic == nil && h.Spec.SecretRef != nil) || (h.Spec.Basic != nil && !*h.Spec.Basic) {
		return false
	}

	return true
}
