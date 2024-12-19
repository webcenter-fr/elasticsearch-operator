package cerebro

import (
	"fmt"

	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderService permit to generate service
func buildServices(cb *cerebrocrd.Cerebro) (services []corev1.Service, err error) {
	services = []corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cb.Namespace,
				Name:      GetServiceName(cb),
				Labels: getLabels(cb, map[string]string{
					fmt.Sprintf("%s/service", cerebrocrd.CerebroAnnotationKey): "true",
				}),
				Annotations: getAnnotations(cb),
			},
			Spec: corev1.ServiceSpec{
				Type:            corev1.ServiceTypeClusterIP,
				SessionAffinity: corev1.ServiceAffinityNone,
				Selector: map[string]string{
					"cluster":                       cb.Name,
					cerebrocrd.CerebroAnnotationKey: "true",
				},
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
