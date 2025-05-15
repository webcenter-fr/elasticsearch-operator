package v1

import "github.com/disaster37/operator-sdk-extra/v2/pkg/object"

// GetStatus return the status object
func (o *LogstashPipeline) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetRoleName return the role name
// If name is empty, it use the ressource name
func (o *LogstashPipeline) GetExternalName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
