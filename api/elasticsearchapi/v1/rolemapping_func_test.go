package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleMappingGetStatus(t *testing.T) {
	status := RoleMappingStatus{
		BasicRemoteObjectStatus: apis.BasicRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestRoleMappingGetExternalName(t *testing.T) {
	var o *RoleMapping

	// When name is set
	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: RoleMappingSpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &RoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: RoleMappingSpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
