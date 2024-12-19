package v1

import "github.com/disaster37/operator-sdk-extra/pkg/object"

// GetStatus return the status object
func (o *RoleMapping) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetExternalName return the role mapping name
// If name is empty, it use the ressource name
func (o *RoleMapping) GetExternalName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
