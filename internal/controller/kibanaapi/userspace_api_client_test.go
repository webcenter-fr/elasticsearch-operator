package kibanaapi

import (
	"testing"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	"github.com/stretchr/testify/assert"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUserSpaceBuild(t *testing.T) {
	var (
		o             *kibanaapicrd.UserSpace
		space         *kbapi.KibanaSpace
		expectedSpace *kbapi.KibanaSpace
		err           error
		client        *userSpaceApiClient
	)

	client = &userSpaceApiClient{}

	// Normale case
	o = &kibanaapicrd.UserSpace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanaapicrd.UserSpaceSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Name: "my space",
		},
	}

	expectedSpace = &kbapi.KibanaSpace{
		ID:   "test",
		Name: "my space",
	}

	space, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSpace, space)

	// With all parameters
	o = &kibanaapicrd.UserSpace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: kibanaapicrd.UserSpaceSpec{
			KibanaRef: shared.KibanaRef{
				ManagedKibanaRef: &shared.KibanaManagedRef{
					Name: "test",
				},
			},
			Name:        "my space",
			Description: "my description",
			DisabledFeatures: []string{
				"dev_tools",
			},
			Initials: "tt",
			Color:    "#aabbcc",
		},
	}

	expectedSpace = &kbapi.KibanaSpace{
		ID:          "test",
		Name:        "my space",
		Description: "my description",
		DisabledFeatures: []string{
			"dev_tools",
		},
		Initials: "tt",
		Color:    "#aabbcc",
	}

	space, err = client.Build(o)
	assert.NoError(t, err)
	assert.Equal(t, expectedSpace, space)
}
