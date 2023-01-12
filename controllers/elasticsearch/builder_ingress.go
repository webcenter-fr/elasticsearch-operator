package elasticsearch

import (
	"github.com/disaster37/k8sbuilder"
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildIngress permit to generate Ingress object
// It return error if ingress spec is not provided
// It return nil if ingress is disabled
func BuildIngress(es *elasticsearchcrd.Elasticsearch) (ingress *networkingv1.Ingress, err error) {
	if !es.IsIngressEnabled() {
		return nil, nil
	}

	if es.Spec.Endpoint.Ingress.Host == "" {
		return nil, errors.New("endpoint.ingress.host must be provided")
	}

	var tls []networkingv1.IngressTLS

	pathType := networkingv1.PathTypePrefix

	// Add default annotation
	defaultAnnotations := map[string]string{}
	if es.IsTlsApiEnabled() {
		defaultAnnotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = "true"
		defaultAnnotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"
	}

	// Compute target service
	targetService := GetGlobalServiceName(es)
	if es.Spec.Endpoint.Ingress.TargetNodeGroupName != "" {
		// Check the node group specified exist
		isFound := false
		for _, nodeGroup := range es.Spec.NodeGroups {
			if nodeGroup.Name == es.Spec.Endpoint.Ingress.TargetNodeGroupName {
				isFound = true
				break
			}
		}
		if !isFound {
			return nil, errors.Errorf("The target group name '%s' not found", es.Spec.Endpoint.Ingress.TargetNodeGroupName)
		}

		targetService = GetNodeGroupServiceName(es, es.Spec.Endpoint.Ingress.TargetNodeGroupName)
	}

	// Compute TLS
	if es.Spec.Endpoint.Ingress.SecretRef != nil {
		tls = []networkingv1.IngressTLS{
			{
				Hosts:      []string{es.Spec.Endpoint.Ingress.Host},
				SecretName: es.Spec.Endpoint.Ingress.SecretRef.Name,
			},
		}
	}

	// Generate ingress
	ingress = &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   es.Namespace,
			Name:        GetIngressName(es),
			Labels:      getLabels(es, es.Spec.Endpoint.Ingress.Labels),
			Annotations: getAnnotations(es, defaultAnnotations, es.Spec.Endpoint.Ingress.Annotations),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: es.Spec.Endpoint.Ingress.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: targetService,
											Port: networkingv1.ServiceBackendPort{Number: 9200},
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
	if err = k8sbuilder.MergeK8s(&ingress.Spec, ingress.Spec, es.Spec.Endpoint.Ingress.IngressSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge ingress spec")
	}

	return ingress, nil
}
