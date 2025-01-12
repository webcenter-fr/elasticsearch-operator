package elasticsearch

import (
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildRoleBindings permit to generate RoleBinding object
// It return nil if we are not on Openshift
// We only need roleBinding on Openshift because of we need to binding it with scc
func buildRoleBindings(es *elasticsearchcrd.Elasticsearch, isOpenshift bool) (rolesBindings []rbacv1.RoleBinding, err error) {
	if !isOpenshift {
		return nil, nil
	}

	clusterRoleName := "system:openshift:scc:anyuid"
	if es.IsSetVMMaxMapCount() {
		clusterRoleName = "system:openshift:scc:privileged"
	}

	rolesBindings = []rbacv1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   es.Namespace,
				Name:        GetServiceAccountName(es),
				Labels:      getLabels(es),
				Annotations: getAnnotations(es),
			},
			Subjects: []rbacv1.Subject{
				{
					Name:      GetServiceAccountName(es),
					Namespace: es.Namespace,
					Kind:      "ServiceAccount",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     clusterRoleName,
			},
		},
	}

	return rolesBindings, nil
}
