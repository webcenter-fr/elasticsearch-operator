package v1

import "github.com/disaster37/operator-sdk-extra/pkg/object"

// GetStatus return the status object
func (o *Watch) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetExternalName return the watch name
// If name is empty, it use the ressource name
func (o *Watch) GetExternalName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
