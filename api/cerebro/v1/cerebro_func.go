package v1

import (
	"github.com/disaster37/operator-sdk-extra/v2/pkg/object"
)

// GetStatus implement the object.MultiPhaseObject
func (h *Cerebro) GetStatus() object.MultiPhaseObjectStatus {
	return &h.Status
}
