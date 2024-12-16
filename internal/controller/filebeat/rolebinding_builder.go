package filebeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildRoleBindings permit to generate RoleBinding object
// It return nil if we are not on Openshift
// We only need roleBinding on Openshift because of we need to binding it with scc
func buildRoleBindings(fb *beatcrd.Filebeat, isOpenshift bool) (rolesBindings []rbacv1.RoleBinding, err error) {
	if !isOpenshift {
		return nil, nil
	}

	rolesBindings = []rbacv1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   fb.Namespace,
				Name:        GetServiceAccountName(fb),
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
			},
			Subjects: []rbacv1.Subject{
				{
					Name:      GetServiceAccountName(fb),
					Namespace: fb.Namespace,
					Kind:      "ServiceAccount",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     "system:openshift:scc:anyuid",
			},
		},
	}

	return rolesBindings, nil
}
