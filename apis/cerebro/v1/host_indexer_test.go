package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupHostIndexer() {
	// Add host to force indexer

	host := &Host{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: HostSpec{
			CerebroRef: HostCerebroRef{
				Name:      "test",
				Namespace: "default",
			},
			ElasticsearchRef: "test",
		},
	}

	err := t.k8sClient.Create(context.Background(), host)
	assert.NoError(t.T(), err)

}
