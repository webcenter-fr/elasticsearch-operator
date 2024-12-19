package metricbeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildRoleBindings permit to generate RoleBinding object
// It return nil if we are not on Openshift
// We only need roleBinding on Openshift because of we need to binding it with scc
func buildRoleBindings(mb *beatcrd.Metricbeat, isOpenshift bool) (rolesBindings []rbacv1.RoleBinding, err error) {
	if !isOpenshift {
		return nil, nil
	}

	rolesBindings = []rbacv1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   mb.Namespace,
				Name:        GetServiceAccountName(mb),
				Labels:      getLabels(mb),
				Annotations: getAnnotations(mb),
			},
			Subjects: []rbacv1.Subject{
				{
					Name:      GetServiceAccountName(mb),
					Namespace: mb.Namespace,
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
