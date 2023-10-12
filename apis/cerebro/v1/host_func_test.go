package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHostGetStatus(t *testing.T) {
	status := HostStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Host{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}
