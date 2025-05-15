package logstash

import (
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderServices permit to generate service
// It also generate service needed by ingress
// It inject the right selector for Logstash pods
func buildServices(ls *logstashcrd.Logstash) (services []*corev1.Service, err error) {
	services = make([]*corev1.Service, 0, len(ls.Spec.Services))
	var service *corev1.Service
	computedPort := make([]corev1.ServicePort, 0, len(ls.Spec.Services)+len(ls.Spec.Ingresses)+len(ls.Spec.Deployment.Ports))
	var isPortAlreadyUsed bool

	computedPort = append(computedPort, corev1.ServicePort{
		Name:       "http",
		Protocol:   corev1.ProtocolTCP,
		Port:       9600,
		TargetPort: intstr.FromString("http"),
	})

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
			logstashcrd.LogstashAnnotationKey: "true",
			"cluster":                         ls.Name,
		}

		services = append(services, service)

		for _, port := range service.Spec.Ports {
			isPortAlreadyUsed = false
			for _, portUsed := range computedPort {
				if port.Protocol == portUsed.Protocol && (port.Name == portUsed.Name || port.Port == portUsed.Port) {
					isPortAlreadyUsed = true
					break
				}
			}

			if !isPortAlreadyUsed {
				// To not add nodePort
				computedPort = append(computedPort, corev1.ServicePort{
					Name:        port.Name,
					Protocol:    port.Protocol,
					AppProtocol: port.AppProtocol,
					Port:        port.Port,
					TargetPort:  port.TargetPort,
				})
			}
		}
	}

	// Create specific services needed by ingress
	for _, i := range ls.Spec.Ingresses {
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
						Protocol:   i.ContainerPortProtocol,
						TargetPort: intstr.FromInt(int(i.ContainerPort)),
						Port:       int32(i.ContainerPort),
						Name:       i.Name,
					},
				},
				Selector: map[string]string{
					logstashcrd.LogstashAnnotationKey: "true",
					"cluster":                         ls.Name,
				},
			},
		}

		services = append(services, service)

		isPortAlreadyUsed = false
		for _, portUsed := range computedPort {
			if i.ContainerPortProtocol == portUsed.Protocol && (int32(i.ContainerPort) == portUsed.Port || i.Name == portUsed.Name) {
				isPortAlreadyUsed = true
				break
			}
		}

		if !isPortAlreadyUsed {
			computedPort = append(computedPort, corev1.ServicePort{
				Protocol:   i.ContainerPortProtocol,
				TargetPort: intstr.FromInt(int(i.ContainerPort)),
				Port:       int32(i.ContainerPort),
				Name:       i.Name,
			})
		}
	}

	// Create specific services needed by route
	for _, i := range ls.Spec.Routes {
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
						Protocol:   i.ContainerPortProtocol,
						TargetPort: intstr.FromInt(int(i.ContainerPort)),
						Port:       int32(i.ContainerPort),
						Name:       i.Name,
					},
				},
				Selector: map[string]string{
					logstashcrd.LogstashAnnotationKey: "true",
					"cluster":                         ls.Name,
				},
			},
		}

		services = append(services, service)

		isPortAlreadyUsed = false
		for _, portUsed := range computedPort {
			if i.ContainerPortProtocol == portUsed.Protocol && (int32(i.ContainerPort) == portUsed.Port || i.Name == portUsed.Name) {
				isPortAlreadyUsed = true
				break
			}
		}

		if !isPortAlreadyUsed {
			computedPort = append(computedPort, corev1.ServicePort{
				Protocol:   i.ContainerPortProtocol,
				TargetPort: intstr.FromInt(int(i.ContainerPort)),
				Port:       int32(i.ContainerPort),
				Name:       i.Name,
			})
		}
	}

	// Compute global service with custom container ports
	for _, port := range ls.Spec.Deployment.Ports {
		isPortAlreadyUsed = false
		for _, portUsed := range computedPort {
			if port.Protocol == portUsed.Protocol && (port.ContainerPort == portUsed.Port || port.Name == portUsed.Name) {
				isPortAlreadyUsed = true
				break
			}
		}

		if !isPortAlreadyUsed {
			computedPort = append(computedPort, corev1.ServicePort{
				Protocol:   port.Protocol,
				TargetPort: intstr.FromInt(int(port.ContainerPort)),
				Port:       port.ContainerPort,
				Name:       port.Name,
			})
		}
	}

	// Create global service with all ports
	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ls.Namespace,
			Name:        GetGlobalServiceName(ls),
			Labels:      getLabels(ls),
			Annotations: getAnnotations(ls),
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeClusterIP,
			Ports: computedPort,
			Selector: map[string]string{
				logstashcrd.LogstashAnnotationKey: "true",
				"cluster":                         ls.Name,
			},
			ClusterIP: "None",
		},
	}

	services = append(services, service)

	return services, nil
}
