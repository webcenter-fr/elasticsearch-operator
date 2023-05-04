package elasticsearch

import (
	"fmt"

	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderServices permit to generate services
// It generate one for all cluster and for each node group
// For each node groups, it also generate headless services
func BuildServices(es *elasticsearchcrd.Elasticsearch) (services []corev1.Service, err error) {
	services = make([]corev1.Service, 0, (1+len(es.Spec.NodeGroups))*2)
	var (
		service         *corev1.Service
		headlessService *corev1.Service
	)

	defaultHeadlessAnnotations := map[string]string{
		"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
	}

	// Generate cluster service
	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: es.Namespace,
			Name:      GetGlobalServiceName(es),
			Labels: getLabels(es, map[string]string{
				fmt.Sprintf("%s/service", elasticsearchcrd.ElasticsearchAnnotationKey): "true",
			}),
			Annotations: getAnnotations(es),
		},
		Spec: corev1.ServiceSpec{
			Type:            corev1.ServiceTypeClusterIP,
			SessionAffinity: corev1.ServiceAffinityNone,
			Selector: map[string]string{
				"cluster": es.Name,
				elasticsearchcrd.ElasticsearchAnnotationKey: "true",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       9200,
					TargetPort: intstr.FromInt(9200),
				},
				{
					Name:       "transport",
					Protocol:   corev1.ProtocolTCP,
					Port:       9300,
					TargetPort: intstr.FromInt(9300),
				},
			},
		},
	}

	services = append(services, *service)

	// Generate service for each node group
	for _, nodeGroup := range es.Spec.NodeGroups {
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: es.Namespace,
				Name:      GetNodeGroupServiceName(es, nodeGroup.Name),
				Labels: getLabels(es, map[string]string{
					"nodeGroup": nodeGroup.Name,
					fmt.Sprintf("%s/service", elasticsearchcrd.ElasticsearchAnnotationKey): "true",
				}),
				Annotations: getAnnotations(es),
			},
			Spec: corev1.ServiceSpec{
				Type:            corev1.ServiceTypeClusterIP,
				SessionAffinity: corev1.ServiceAffinityNone,
				Selector: map[string]string{
					"cluster":   es.Name,
					"nodeGroup": nodeGroup.Name,
					elasticsearchcrd.ElasticsearchAnnotationKey: "true",
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   corev1.ProtocolTCP,
						Port:       9200,
						TargetPort: intstr.FromInt(9200),
					},
					{
						Name:       "transport",
						Protocol:   corev1.ProtocolTCP,
						Port:       9300,
						TargetPort: intstr.FromInt(9300),
					},
				},
			},
		}

		headlessService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: es.Namespace,
				Name:      GetNodeGroupServiceNameHeadless(es, nodeGroup.Name),
				Labels: getLabels(es, map[string]string{
					"nodeGroup": nodeGroup.Name,
					fmt.Sprintf("%s/service", elasticsearchcrd.ElasticsearchAnnotationKey): "true",
				}),
				Annotations: getAnnotations(es, defaultHeadlessAnnotations),
			},
			Spec: corev1.ServiceSpec{
				ClusterIP:                "None",
				PublishNotReadyAddresses: true,
				Type:                     corev1.ServiceTypeClusterIP,
				SessionAffinity:          corev1.ServiceAffinityNone,
				Selector: map[string]string{
					"cluster":   es.Name,
					"nodeGroup": nodeGroup.Name,
					elasticsearchcrd.ElasticsearchAnnotationKey: "true",
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   corev1.ProtocolTCP,
						Port:       9200,
						TargetPort: intstr.FromInt(9200),
					},
					{
						Name:       "transport",
						Protocol:   corev1.ProtocolTCP,
						Port:       9300,
						TargetPort: intstr.FromInt(9300),
					},
				},
			},
		}

		services = append(services, *service, *headlessService)
	}

	return services, nil
}
