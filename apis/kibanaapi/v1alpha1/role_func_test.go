package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetRoleName(t *testing.T) {
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

	assert.Equal(t, "test2", o.GetRoleName())

	// When name isn't set
	o = &Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: RoleSpec{},
	}

	assert.Equal(t, "test", o.GetRoleName())
}
