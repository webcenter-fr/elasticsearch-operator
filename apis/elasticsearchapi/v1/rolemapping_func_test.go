package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
