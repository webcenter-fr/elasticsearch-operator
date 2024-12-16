package metricbeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// buildServiceAccounts permit to generate ServiceAccount object
// It return nil if we are not on Openshift
// We only need service account on Openshift because of we need to binding it with scc
func buildServiceAccounts(mb *beatcrd.Metricbeat, isOpenshift bool) (serviceAccounts []corev1.ServiceAccount, err error) {
	if !isOpenshift {
		return nil, nil
	}

	serviceAccounts = []corev1.ServiceAccount{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   mb.Namespace,
				Name:        GetServiceAccountName(mb),
				Labels:      getLabels(mb),
				Annotations: getAnnotations(mb),
			},
			AutomountServiceAccountToken: ptr.To[bool](false),
		},
	}

	return serviceAccounts, nil
}
