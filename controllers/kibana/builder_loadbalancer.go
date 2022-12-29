package kibana

import (
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GenerateLoadbalancer permit to generate Loadbalancer throught service
// It return nil if Loadbalancer is disabled
func BuildLoadbalancer(kb *kibanaapi.Kibana) (service *corev1.Service, err error) {

	if !kb.IsLoadBalancerEnabled() {
		return nil, nil
	}

	selector := map[string]string{
		"cluster":           kb.Name,
		KibanaAnnotationKey: "true",
	}

	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetLoadBalancerName(kb),
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Spec: corev1.ServiceSpec{
			Type:            corev1.ServiceTypeLoadBalancer,
			SessionAffinity: corev1.ServiceAffinityNone,
			Selector:        selector,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       5601,
					TargetPort: intstr.FromInt(5601),
				},
			},
		},
	}

	return service, nil
}
