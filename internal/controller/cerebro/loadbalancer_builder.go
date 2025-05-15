package cerebro

import (
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GenerateLoadbalancer permit to generate Loadbalancer throught service
// It return nil if Loadbalancer is disabled
func buildLoadbalancers(cb *cerebrocrd.Cerebro) (services []*corev1.Service, err error) {
	if !cb.Spec.Endpoint.IsLoadBalancerEnabled() {
		return nil, nil
	}

	selector := map[string]string{
		"cluster":                       cb.Name,
		cerebrocrd.CerebroAnnotationKey: "true",
	}

	services = []*corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   cb.Namespace,
				Name:        GetLoadBalancerName(cb),
				Labels:      getLabels(cb),
				Annotations: getAnnotations(cb),
			},
			Spec: corev1.ServiceSpec{
				Type:            corev1.ServiceTypeLoadBalancer,
				SessionAffinity: corev1.ServiceAffinityNone,
				Selector:        selector,
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   corev1.ProtocolTCP,
						Port:       9000,
						TargetPort: intstr.FromInt(9000),
					},
				},
			},
		},
	}

	return services, nil
}
