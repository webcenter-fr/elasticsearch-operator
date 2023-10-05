package v1

import "github.com/disaster37/operator-sdk-extra/pkg/object"

// GetStatus return the status object
func (o *SnapshotLifecyclePolicy) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetExternalName return the snapshot lifecycle policy name
// If name is empty, it use the ressource name
func (o *SnapshotLifecyclePolicy) GetExternalName() string {
	if o.Spec.SnapshotLifecyclePolicyName == "" {
		return o.Name
	}

	return o.Spec.SnapshotLifecyclePolicyName
}
