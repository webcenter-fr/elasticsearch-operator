package filebeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildRoleBindings(t *testing.T) {
	var (
		err          error
		roleBindings []*rbacv1.RoleBinding
		o            *beatcrd.Filebeat
	)

	// With default values
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	roleBindings, err = buildRoleBindings(o, false)
	assert.NoError(t, err)
	assert.Empty(t, roleBindings)

	// When on Openshift
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	roleBindings, err = buildRoleBindings(o, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(roleBindings))
	test.EqualFromYamlFile[*rbacv1.RoleBinding](t, "testdata/rolebinding_default.yml", roleBindings[0], scheme.Scheme)
}
