package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/stretchr/testify/assert"
)

func (t *TestSuite) TestSetupKibanaIndexer() {
	// Add indexers
	err := controller.SetupIndexerWithManager(
		t.k8sManager,
		SetupKibanaIndexer,
	)

	assert.NoError(t.T(), err)
}
