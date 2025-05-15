package v1

import "github.com/disaster37/operator-sdk-extra/v2/pkg/object"

// GetStatus return the status object
func (o *License) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetExternalName permit to get the license name
func (h *License) GetExternalName() string {
	return h.Name
}

// IsBasicLicense return true if is basic license
func (h *License) IsBasicLicense() bool {
	if (h.Spec.Basic == nil && h.Spec.SecretRef != nil) || (h.Spec.Basic != nil && !*h.Spec.Basic) {
		return false
	}

	return true
}
