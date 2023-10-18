package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupLicenseIndexer() {
	// Add license to force indexer execution

	license := &License{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: LicenseSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			SecretRef: &v1.LocalObjectReference{
				Name: "test",
			},
		},
	}

	err := t.k8sClient.Create(context.Background(), license)
	assert.NoError(t.T(), err)

}
