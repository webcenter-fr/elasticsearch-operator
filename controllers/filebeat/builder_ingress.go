package filebeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildIngresses permit to generate Ingresses object
// It will overwrite the target service
func BuildIngresses(fb *beatcrd.Filebeat) (ingresses []networkingv1.Ingress, err error) {

	ingresses = make([]networkingv1.Ingress, 0, len(fb.Spec.Ingresses))
	var (
		ingress *networkingv1.Ingress
	)

	for _, i := range fb.Spec.Ingresses {

		ingress = &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   fb.Namespace,
				Name:        GetIngressName(fb, i.Name),
				Labels:      getLabels(fb, i.Labels),
				Annotations: getAnnotations(fb, i.Annotations),
			},
			Spec: *i.Spec.DeepCopy(),
		}

		for indexRule, rule := range ingress.Spec.Rules {
			if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
				for indexPath, path := range rule.HTTP.Paths {
					path.Backend = networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: GetServiceName(fb, i.Name),
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
							Name: GetServiceName(fb, i.Name),
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

	}

	return ingresses, nil
}
