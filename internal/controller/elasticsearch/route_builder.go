package elasticsearch

import (
	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	routev1 "github.com/openshift/api/route/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// buildRoutes permit to generate Route object
// It return error if route spec is not provided
// It return nil if route is disabled
func buildRoutes(es *elasticsearchcrd.Elasticsearch, secretTlsApi *corev1.Secret) (routes []routev1.Route, err error) {
	if !es.IsRouteEnabled() {
		return nil, nil
	}

	routes = make([]routev1.Route, 0, 1)

	if es.Spec.Endpoint.Route.Host == "" {
		return nil, errors.New("endpoint.route.host must be provided")
	}

	// Compute target service
	targetService := GetGlobalServiceName(es)
	if es.Spec.Endpoint.Route.TargetNodeGroupName != "" {
		// Check the node group specified exist
		isFound := false
		for _, nodeGroup := range es.Spec.NodeGroups {
			if nodeGroup.Name == es.Spec.Endpoint.Route.TargetNodeGroupName {
				isFound = true
				break
			}
		}
		if !isFound {
			return nil, errors.Errorf("The target group name '%s' not found", es.Spec.Endpoint.Route.TargetNodeGroupName)
		}

		targetService = GetNodeGroupServiceName(es, es.Spec.Endpoint.Route.TargetNodeGroupName)
	}

	// Generate route
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   es.Namespace,
			Name:        GetIngressName(es),
			Labels:      getLabels(es, es.Spec.Endpoint.Route.Labels),
			Annotations: getAnnotations(es, es.Spec.Endpoint.Route.Annotations),
		},
		Spec: routev1.RouteSpec{
			Host: es.Spec.Endpoint.Route.Host,
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: targetService,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
		},
	}

	// Enabled TLS
	if es.Spec.Tls.IsTlsEnabled() || (es.Spec.Endpoint.Route.TlsEnabled != nil && *es.Spec.Endpoint.Route.TlsEnabled) {
		route.Spec.TLS = &routev1.TLSConfig{
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
		}
		if es.Spec.Endpoint.Route.SecretRef != nil {
			route.Spec.TLS.ExternalCertificate = &routev1.LocalObjectReference{
				Name: es.Spec.Endpoint.Route.SecretRef.Name,
			}
		}
		if es.Spec.Tls.IsTlsEnabled() {
			route.Spec.TLS.Termination = routev1.TLSTerminationReencrypt
		} else {
			route.Spec.TLS.Termination = routev1.TLSTerminationEdge
		}
		if secretTlsApi != nil && len(secretTlsApi.Data["ca.crt"]) > 0 {
			route.Spec.TLS.DestinationCACertificate = string(secretTlsApi.Data["ca.crt"])
		}
	}

	// Merge expected route with custom route spec
	if err = k8sbuilder.MergeK8s(&route.Spec, route.Spec, es.Spec.Endpoint.Route.RouteSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge route spec")
	}

	// Avoid to reset target service fater merge provided custom spec because of is not pointer
	if route.Spec.To.Name == "" {
		route.Spec.To = routev1.RouteTargetReference{
			Kind: "Service",
			Name: targetService,
		}
	}

	routes = append(routes, *route)

	return routes, nil
}
