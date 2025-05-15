package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/multiphase"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreberoGetStatus(t *testing.T) {
	status := CerebroStatus{
		DefaultMultiPhaseObjectStatus: multiphase.DefaultMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}
