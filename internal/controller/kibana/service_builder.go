package kibana

import (
	"fmt"

	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderService permit to generate service
func buildServices(kb *kibanacrd.Kibana) (services []*corev1.Service, err error) {
	services = []*corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: kb.Namespace,
				Name:      GetServiceName(kb),
				Labels: getLabels(kb, map[string]string{
					fmt.Sprintf("%s/service", kibanacrd.KibanaAnnotationKey): "true",
				}),
				Annotations: getAnnotations(kb),
			},
			Spec: corev1.ServiceSpec{
				Type:            corev1.ServiceTypeClusterIP,
				SessionAffinity: corev1.ServiceAffinityNone,
				Selector: map[string]string{
					"cluster":                     kb.Name,
					kibanacrd.KibanaAnnotationKey: "true",
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   corev1.ProtocolTCP,
						Port:       5601,
						TargetPort: intstr.FromInt(5601),
					},
				},
			},
		},
	}

	return services, nil
}
