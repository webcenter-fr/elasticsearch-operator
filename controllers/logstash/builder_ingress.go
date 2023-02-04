package logstash

import (
	"fmt"
	"strings"

	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildIngresses permit to generate Ingresses object
// It will overwrite the target service
func BuildIngresses(ls *logstashcrd.Logstash) (ingresses []networkingv1.Ingress, err error) {

	ingresses = make([]networkingv1.Ingress, 0, len(ls.Spec.Ingresses))
	var (
		ingress *networkingv1.Ingress
	)

	for _, i := range ls.Spec.Ingresses {

		if !i.IsOneForEachLogstashInstance() {
			// Global ingress
			ingress = &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   ls.Namespace,
					Name:        GetIngressName(ls, i.Name),
					Labels:      getLabels(ls, i.Labels),
					Annotations: getAnnotations(ls, i.Annotations),
				},
				Spec: *i.Spec.DeepCopy(),
			}

			for indexRule, rule := range ingress.Spec.Rules {
				if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
					for indexPath, path := range rule.HTTP.Paths {
						path.Backend = networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: GetServiceName(ls, i.Name),
								Port: networkingv1.ServiceBackendPort{
									Number: int32(i.ContainerPort),
								},
							},
						}
						rule.HTTP.Paths[indexPath] = path
					}
				}
				if rule.IngressRuleValue.HTTP != nil && len(rule.IngressRuleValue.HTTP.Paths) > 0 {
					for indexPath, path := range rule.IngressRuleValue.HTTP.Paths {
						path.Backend = networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: GetServiceName(ls, i.Name),
								Port: networkingv1.ServiceBackendPort{
									Number: int32(i.ContainerPort),
								},
							},
						}
						rule.IngressRuleValue.HTTP.Paths[indexPath] = path
					}
				}

				ingress.Spec.Rules[indexRule] = rule
			}

			ingresses = append(ingresses, *ingress)

		} else {
			// Ingress for each pod

			for indexPod := 0; indexPod < int(ls.Spec.Deployment.Replicas); indexPod++ {
				ingress = &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ls.Namespace,
						Name:        fmt.Sprintf("%s-%d", GetIngressName(ls, i.Name), indexPod),
						Labels:      getLabels(ls, i.Labels),
						Annotations: getAnnotations(ls, i.Annotations),
					},
					Spec: *i.Spec.DeepCopy(),
				}

				for indexRule, rule := range ingress.Spec.Rules {
					if strings.Contains(rule.Host, "%d") {
						rule.Host = fmt.Sprintf(rule.Host, indexPod)
					} else {
						rule.Host = fmt.Sprintf("%d-"+rule.Host, indexPod)
					}

					if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
						for indexPath, path := range rule.HTTP.Paths {
							path.Backend = networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: fmt.Sprintf("%s-%d", GetServiceName(ls, i.Name), indexPod),
									Port: networkingv1.ServiceBackendPort{
										Number: int32(i.ContainerPort),
									},
								},
							}
							rule.HTTP.Paths[indexPath] = path
						}
					}
					if rule.IngressRuleValue.HTTP != nil && len(rule.IngressRuleValue.HTTP.Paths) > 0 {
						for indexPath, path := range rule.IngressRuleValue.HTTP.Paths {
							path.Backend = networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: fmt.Sprintf("%s-%d", GetServiceName(ls, i.Name), indexPod),
									Port: networkingv1.ServiceBackendPort{
										Number: int32(i.ContainerPort),
									},
								},
							}
							rule.IngressRuleValue.HTTP.Paths[indexPath] = path
						}
					}

					ingress.Spec.Rules[indexRule] = rule
				}

				for indexTls, tls := range ingress.Spec.TLS {
					for indexHost, host := range tls.Hosts {
						if strings.Contains(host, "%d") {
							tls.Hosts[indexHost] = fmt.Sprintf(host, indexPod)
						} else {
							tls.Hosts[indexHost] = fmt.Sprintf("%d-"+host, indexPod)
						}

					}

					ingress.Spec.TLS[indexTls] = tls
				}

				ingresses = append(ingresses, *ingress)
			}
		}

	}

	return ingresses, nil
}
