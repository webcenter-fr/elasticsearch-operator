package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupUserSpaceIndexer() {
	// Add UserSpace to force  indexer execution

	o := &UserSpace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: UserSpaceSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Name: "test",
		},
	}

	err := t.k8sClient.Create(context.Background(), o)
	assert.NoError(t.T(), err)
}
