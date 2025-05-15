package v1

import "github.com/disaster37/operator-sdk-extra/v2/pkg/object"

// GetStatus return the status object
func (o *SnapshotRepository) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetExternalName get the snapshot repository name
// If name is empty, it use the ressource name
func (o *SnapshotRepository) GetExternalName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
