package logstash

import (
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// buildServiceAccounts permit to generate ServiceAccount object
// It return nil if we are not on Openshift
// We only need service account on Openshift because of we need to binding it with scc
func buildServiceAccounts(ls *logstashcrd.Logstash, isOpenshift bool) (serviceAccounts []corev1.ServiceAccount, err error) {
	if !isOpenshift {
		return nil, nil
	}

	serviceAccounts = []corev1.ServiceAccount{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetServiceAccountName(ls),
				Labels:      getLabels(ls),
				Annotations: getAnnotations(ls),
			},
			AutomountServiceAccountToken: ptr.To[bool](false),
		},
	}

	return serviceAccounts, nil
}
