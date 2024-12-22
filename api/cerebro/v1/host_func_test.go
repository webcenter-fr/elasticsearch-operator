package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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

func TestHostIsManaged(t *testing.T) {
	var o ElasticsearchRef

	// When managed
	o = ElasticsearchRef{
		ManagedElasticsearchRef: &v1.LocalObjectReference{
			Name: "test",
		},
	}
	assert.True(t, o.IsManaged())

	// When not managed
	o = ElasticsearchRef{
		ManagedElasticsearchRef: &v1.LocalObjectReference{},
	}
	assert.False(t, o.IsManaged())

	o = ElasticsearchRef{}
	assert.False(t, o.IsManaged())
}

func TestHostIsExternal(t *testing.T) {
	var o ElasticsearchRef

	// When external
	o = ElasticsearchRef{
		ExternalElasticsearchRef: &ElasticsearchExternalRef{
			Name:    "test",
			Address: "test",
		},
	}
	assert.True(t, o.IsExternal())

	// When not managed
	o = ElasticsearchRef{
		ExternalElasticsearchRef: &ElasticsearchExternalRef{},
	}
	assert.False(t, o.IsExternal())

	o = ElasticsearchRef{}
	assert.False(t, o.IsExternal())
}
