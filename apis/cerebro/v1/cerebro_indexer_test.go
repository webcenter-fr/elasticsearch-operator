package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/stretchr/testify/assert"
)

func (t *TestSuite) TestSetupCerebroIndexer() {
	// Add indexers
	err := controller.SetupIndexerWithManager(
		t.k8sManager,
		SetupCerebroIndexer,
	)

	assert.NoError(t.T(), err)
}
