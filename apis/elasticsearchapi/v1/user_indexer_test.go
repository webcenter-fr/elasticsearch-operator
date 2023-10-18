package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupUserIndexer() {
	// Add user to force  indexer execution

	user := &User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: UserSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Username: "test",
			SecretRef: &UserSecret{
				Name: "test",
				Key:  "password",
			},
		},
	}

	err := t.k8sClient.Create(context.Background(), user)
	assert.NoError(t.T(), err)
}
