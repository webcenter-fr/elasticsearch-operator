package kibana

import (
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// buildServiceAccounts permit to generate ServiceAccount object
// It return nil if we are not on Openshift
// We only need service account on Openshift because of we need to binding it with scc
func buildServiceAccounts(kb *kibanacrd.Kibana, isOpenshift bool) (serviceAccounts []*corev1.ServiceAccount, err error) {
	if !isOpenshift {
		return nil, nil
	}

	serviceAccounts = []*corev1.ServiceAccount{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   kb.Namespace,
				Name:        GetServiceAccountName(kb),
				Labels:      getLabels(kb),
				Annotations: getAnnotations(kb),
			},
			AutomountServiceAccountToken: ptr.To[bool](false),
		},
	}

	return serviceAccounts, nil
}
