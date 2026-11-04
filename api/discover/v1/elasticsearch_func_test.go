package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/multiphase"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestElasticsearchGetStatus(t *testing.T) {
	status := ElasticsearchStatus{
		DefaultMultiPhaseObjectStatus: multiphase.DefaultMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestElasticsearchGetDiscoverStatus(t *testing.T) {
	status := ElasticsearchStatus{
		DiscoverStatus: DiscoverStatus{
			SecretEnvRef:  ptr.To("test-env"),
			SecretFileRef: ptr.To("test-file"),
		},
	}
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetDiscoverStatus())
}

func TestElasticsearchGetInternalName(t *testing.T) {
	// When custom name
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			Discover: Discover{
				Name: ptr.To("custom-name"),
			},
		},
	}
	assert.Equal(t, "custom-name", o.GetInternalName())

	// When name is nil
	o.Spec.Name = nil
	assert.Equal(t, "test", o.GetInternalName())

	// When name is empty
	o.Spec.Name = ptr.To("")
	assert.Equal(t, "test", o.GetInternalName())
}

func TestElasticsearchRefValidateField(t *testing.T) {
	var o *Elasticsearch
	var err *field.Error

	// Valid managed Elasticsearch
	o = &Elasticsearch{
		Spec: ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test-elasticsearch",
				},
			},
		},
	}
	assert.True(t, o.Spec.ElasticsearchRef.IsManaged())
	assert.False(t, o.Spec.ElasticsearchRef.IsExternal())
	err = o.Spec.ElasticsearchRef.ValidateField()
	assert.Nil(t, err)

	// Valid external Elasticsearch
	o = &Elasticsearch{
		Spec: ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://elasticsearch.example.com:9200"},
				},
			},
		},
	}
	assert.False(t, o.Spec.ElasticsearchRef.IsManaged())
	assert.True(t, o.Spec.ElasticsearchRef.IsExternal())
	err = o.Spec.ElasticsearchRef.ValidateField()
	assert.Nil(t, err)

	// Invalid when neither managed nor external
	o = &Elasticsearch{
		Spec: ElasticsearchSpec{},
	}
	err = o.Spec.ElasticsearchRef.ValidateField()
	assert.Error(t, err)

	// Invalid when external address is malformed
	o = &Elasticsearch{
		Spec: ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"elasticsearch.example.com:9200"},
				},
			},
		},
	}
	err = o.Spec.ElasticsearchRef.ValidateField()
	assert.Error(t, err)
}
