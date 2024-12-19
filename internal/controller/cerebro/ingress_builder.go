package cerebro

import (
	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildIngress permit to generate Ingress object
// It return error if ingress spec is not provided
// It return nil if ingress is disabled
func buildIngresses(cb *cerebrocrd.Cerebro) (ingresses []networkingv1.Ingress, err error) {
	if !cb.Spec.Endpoint.IsIngressEnabled() {
		return nil, nil
	}

	ingresses = make([]networkingv1.Ingress, 0, 1)

	if cb.Spec.Endpoint.Ingress.Host == "" {
		return nil, errors.New("endpoint.ingress.host must be provided")
	}

	var tls []networkingv1.IngressTLS

	pathType := networkingv1.PathTypePrefix

	// Compute target service
	targetService := GetServiceName(cb)

	// Compute TLS
	if cb.Spec.Endpoint.Ingress.SecretRef != nil {
		tls = []networkingv1.IngressTLS{
			{
				Hosts:      []string{cb.Spec.Endpoint.Ingress.Host},
				SecretName: cb.Spec.Endpoint.Ingress.SecretRef.Name,
			},
		}
	}

	// Generate ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cb.Namespace,
			Name:        GetIngressName(cb),
			Labels:      getLabels(cb, cb.Spec.Endpoint.Ingress.Labels),
			Annotations: getAnnotations(cb, cb.Spec.Endpoint.Ingress.Annotations),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: cb.Spec.Endpoint.Ingress.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: targetService,
											Port: networkingv1.ServiceBackendPort{Number: 9000},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: tls,
		},
	}

	// Merge expected ingress with custom ingress spec
	if err = k8sbuilder.MergeK8s(&ingress.Spec, ingress.Spec, cb.Spec.Endpoint.Ingress.IngressSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge ingress spec")
	}

	ingresses = append(ingresses, *ingress)

	return ingresses, nil
}
