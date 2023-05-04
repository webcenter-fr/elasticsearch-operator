package kibana

import (
	"github.com/disaster37/k8sbuilder"
	"github.com/pkg/errors"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildIngress permit to generate Ingress object
// It return error if ingress spec is not provided
// It return nil if ingress is disabled
func BuildIngress(kb *kibanacrd.Kibana) (ingress *networkingv1.Ingress, err error) {
	if !kb.IsIngressEnabled() {
		return nil, nil
	}

	if kb.Spec.Endpoint.Ingress.Host == "" {
		return nil, errors.New("endpoint.ingress.host must be provided")
	}

	var tls []networkingv1.IngressTLS

	pathType := networkingv1.PathTypePrefix

	// Add default annotation
	defaultAnnotations := map[string]string{}
	if kb.IsTlsEnabled() {
		defaultAnnotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = "true"
		defaultAnnotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"
	}

	// Compute target service
	targetService := GetServiceName(kb)

	// Compute TLS
	if kb.Spec.Endpoint.Ingress.SecretRef != nil {
		tls = []networkingv1.IngressTLS{
			{
				Hosts:      []string{kb.Spec.Endpoint.Ingress.Host},
				SecretName: kb.Spec.Endpoint.Ingress.SecretRef.Name,
			},
		}
	}

	// Generate ingress
	ingress = &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetIngressName(kb),
			Labels:      getLabels(kb, kb.Spec.Endpoint.Ingress.Labels),
			Annotations: getAnnotations(kb, defaultAnnotations, kb.Spec.Endpoint.Ingress.Annotations),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: kb.Spec.Endpoint.Ingress.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: targetService,
											Port: networkingv1.ServiceBackendPort{Number: 5601},
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
	if err = k8sbuilder.MergeK8s(&ingress.Spec, ingress.Spec, kb.Spec.Endpoint.Ingress.IngressSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge ingress spec")
	}

	return ingress, nil
}
