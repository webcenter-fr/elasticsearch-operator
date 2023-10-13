package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/stretchr/testify/assert"
)

func (t *TestSuite) TestSetupElasticsearchIndexer() {
	// Add indexers
	err := controller.SetupIndexerWithManager(
		t.k8sManager,
		SetupElasticsearchIndexer,
	)

	assert.NoError(t.T(), err)
}
