package v1

import "github.com/disaster37/operator-sdk-extra/v2/pkg/object"

// GetStatus return the status object
func (o *ComponentTemplate) GetStatus() object.RemoteObjectStatus {
	return &o.Status
}

// GetExternalName return the component template name
// If name is empty, it use the ressource name
func (o *ComponentTemplate) GetExternalName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}

// IsRawTemplate return true if raw template is set
func (o *ComponentTemplate) IsRawTemplate() bool {
	return o.Spec.RawTemplate != nil
}
