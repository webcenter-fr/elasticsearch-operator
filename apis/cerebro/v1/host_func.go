package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Host) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}
