package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (t *TestSuite) TestSetupLicenseWebhook() {
	var (
		o   *License
		err error
	)

	// Need failed when set multiple license
	// Check we can update it
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test2",
				},
			},
			Basic: ptr.To(true),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
	err = t.k8sClient.Update(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook2",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test2",
				},
			},
			Basic: ptr.To(true),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when create mutiple license on same external cluster
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook3",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Basic: ptr.To(true),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)

	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook4",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test.local"},
				},
			},
			Basic: ptr.To(true),
		},
	}

	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not specify target Elasticsearch cluster
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
			Basic:            ptr.To(true),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when specify basic license and provide secret
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test2.local"},
				},
			},
			Basic: ptr.To(true),
			SecretRef: &v1.LocalObjectReference{
				Name: "test",
			},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)

	// Need failed when not basic and not provide secret
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook5",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{"https://test3.local"},
				},
			},
			Basic: ptr.To(false),
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
