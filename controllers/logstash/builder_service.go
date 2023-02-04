package logstash

import (
	"fmt"

	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderServices permit to generate service
// It also generate service needed by ingress
// It inject the right selector for Logstash pods
func BuildServices(ls *logstashcrd.Logstash) (services []corev1.Service, err error) {

	services = make([]corev1.Service, 0, len(ls.Spec.Services))
	var service *corev1.Service

	// Create regular services
	for _, s := range ls.Spec.Services {
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetServiceName(ls, s.Name),
				Labels:      getLabels(ls, s.Labels),
				Annotations: getAnnotations(ls, s.Annotations),
			},
			Spec: *s.Spec.DeepCopy(),
		}

		service.Spec.Selector = map[string]string{
			LogstashAnnotationKey: "true",
			"cluster":             ls.Name,
		}

		services = append(services, *service)
	}

	// Create specific services needed by ingress
	for _, i := range ls.Spec.Ingresses {
		if !i.IsOneForEachLogstashInstance() {
			// Global ingress

			service = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   ls.Namespace,
					Name:        GetServiceName(ls, i.Name),
					Labels:      getLabels(ls),
					Annotations: getAnnotations(ls),
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Protocol: i.ContainerPortProtocol,
							TargetPort: intstr.IntOrString{
								IntVal: int32(i.ContainerPort),
							},
							Port: int32(i.ContainerPort),
						},
					},
					Selector: map[string]string{
						LogstashAnnotationKey: "true",
						"cluster":             ls.Name,
					},
				},
			}

			services = append(services, *service)
		} else {
			for indexPod := 0; indexPod < int(ls.Spec.Deployment.Replicas); indexPod++ {
				service = &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ls.Namespace,
						Name:        fmt.Sprintf("%s-%d", GetServiceName(ls, i.Name), indexPod),
						Labels:      getLabels(ls),
						Annotations: getAnnotations(ls),
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{
							{
								Protocol: i.ContainerPortProtocol,
								TargetPort: intstr.IntOrString{
									IntVal: int32(i.ContainerPort),
								},
								Port: int32(i.ContainerPort),
							},
						},
						Selector: map[string]string{
							LogstashAnnotationKey:                "true",
							"cluster":                            ls.Name,
							"statefulset.kubernetes.io/pod-name": fmt.Sprintf("%s-%d", GetStatefulsetName(ls), indexPod),
						},
					},
				}

				services = append(services, *service)
			}
		}
	}

	return services, nil
}
