package metricbeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderServices permit to generate service
func buildServices(mb *beatcrd.Metricbeat) (services []corev1.Service, err error) {
	services = make([]corev1.Service, 0, 1)

	// Create global service with all ports
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mb.Namespace,
			Name:        GetGlobalServiceName(mb),
			Labels:      getLabels(mb),
			Annotations: getAnnotations(mb),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       5066,
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: map[string]string{
				beatcrd.MetricbeatAnnotationKey: "true",
				"cluster":                       mb.Name,
			},
			ClusterIP: "None",
		},
	}

	services = append(services, *service)

	return services, nil
}
