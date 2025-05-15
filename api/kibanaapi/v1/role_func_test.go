package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleGetStatus(t *testing.T) {
	status := RoleStatus{
		DefaultRemoteObjectStatus: remote.DefaultRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestRoleGetExternalName(t *testing.T) {
	var o *Role

	// When name is set
	o = &Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: RoleSpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: RoleSpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
