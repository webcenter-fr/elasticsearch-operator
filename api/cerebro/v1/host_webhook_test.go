package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestHostWebhook() {
	var (
		o   *Host
		err error
	)

	// Need failed when not specify target Elasticsearch cluster
	o = &Host{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: HostSpec{
			CerebroRef: HostCerebroRef{
				Name:      "test",
				Namespace: "default",
			},
			ElasticsearchRef: ElasticsearchRef{},
		},
	}
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
