package v1

import "github.com/disaster37/operator-sdk-extra/pkg/object"

// GetStatus return the status object
func (o *User) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// IsProtected return true if user is protected
func (h *User) IsProtected() bool {
	if h.Spec.IsProtected != nil && *h.Spec.IsProtected {
		return true
	}

	return false
}

// GetExternalName return the expected user name
// It take ressource name if username is empty
func (h *User) GetExternalName() string {
	if h.Spec.Username == "" {
		return h.Name
	}

	return h.Spec.Username
}
