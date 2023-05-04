package filebeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuilderServices permit to generate service
// It also generate service needed by ingress
// It inject the right selector for Logstash pods
func BuildServices(fb *beatcrd.Filebeat) (services []corev1.Service, err error) {

	services = make([]corev1.Service, 0, len(fb.Spec.Services))
	var service *corev1.Service
	computedPort := make([]corev1.ServicePort, 0, len(fb.Spec.Services)+len(fb.Spec.Ingresses)+len(fb.Spec.Deployment.Ports))
	var isPortAlreadyUsed bool

	computedPort = append(computedPort, corev1.ServicePort{
		Name:       "http",
		Protocol:   corev1.ProtocolTCP,
		Port:       5066,
		TargetPort: intstr.FromString("http"),
	})

	// Create regular services
	for _, s := range fb.Spec.Services {
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   fb.Namespace,
				Name:        GetServiceName(fb, s.Name),
				Labels:      getLabels(fb, s.Labels),
				Annotations: getAnnotations(fb, s.Annotations),
			},
			Spec: *s.Spec.DeepCopy(),
		}

		service.Spec.Selector = map[string]string{
			beatcrd.FilebeatAnnotationKey: "true",
			"cluster":                     fb.Name,
		}

		services = append(services, *service)

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
	for _, i := range fb.Spec.Ingresses {

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   fb.Namespace,
				Name:        GetServiceName(fb, i.Name),
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
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
					beatcrd.FilebeatAnnotationKey: "true",
					"cluster":                     fb.Name,
				},
			},
		}

		services = append(services, *service)

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
	for _, port := range fb.Spec.Deployment.Ports {
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
			Namespace:   fb.Namespace,
			Name:        GetGlobalServiceName(fb),
			Labels:      getLabels(fb),
			Annotations: getAnnotations(fb),
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeClusterIP,
			Ports: computedPort,
			Selector: map[string]string{
				beatcrd.FilebeatAnnotationKey: "true",
				"cluster":                     fb.Name,
			},
			ClusterIP: "None",
		},
	}

	services = append(services, *service)

	return services, nil
}
