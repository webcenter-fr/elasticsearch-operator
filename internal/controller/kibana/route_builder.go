package kibana

import (
	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	routev1 "github.com/openshift/api/route/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// buildRoutes permit to generate Route object
// It return error if route spec is not provided
// It return nil if route is disabled
func buildRoutes(kb *kibanacrd.Kibana, secretTlsApi *corev1.Secret) (routes []routev1.Route, err error) {
	if !kb.Spec.Endpoint.IsRouteEnabled() {
		return nil, nil
	}

	routes = make([]routev1.Route, 0, 1)

	if kb.Spec.Endpoint.Route.Host == "" {
		return nil, errors.New("endpoint.route.host must be provided")
	}

	// Generate route
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetIngressName(kb),
			Labels:      getLabels(kb, kb.Spec.Endpoint.Route.Labels),
			Annotations: getAnnotations(kb, kb.Spec.Endpoint.Route.Annotations),
		},
		Spec: routev1.RouteSpec{
			Host: kb.Spec.Endpoint.Route.Host,
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: GetServiceName(kb),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
		},
	}

	// Enabled TLS
	if kb.Spec.Tls.IsTlsEnabled() || (kb.Spec.Endpoint.Route.TlsEnabled != nil && *kb.Spec.Endpoint.Route.TlsEnabled) {
		route.Spec.TLS = &routev1.TLSConfig{
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
		}
		if kb.Spec.Endpoint.Route.SecretRef != nil {
			route.Spec.TLS.ExternalCertificate = &routev1.LocalObjectReference{
				Name: kb.Spec.Endpoint.Route.SecretRef.Name,
			}
		}
		if kb.Spec.Tls.IsTlsEnabled() {
			route.Spec.TLS.Termination = routev1.TLSTerminationReencrypt
		} else {
			route.Spec.TLS.Termination = routev1.TLSTerminationEdge
		}
		if secretTlsApi != nil && len(secretTlsApi.Data["ca.crt"]) > 0 {
			route.Spec.TLS.DestinationCACertificate = string(secretTlsApi.Data["ca.crt"])
		}
	}

	// Merge expected route with custom route spec
	if err = k8sbuilder.MergeK8s(&route.Spec, route.Spec, kb.Spec.Endpoint.Route.RouteSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge route spec")
	}

	// Avoid to reset target service fater merge provided custom spec because of is not pointer
	if route.Spec.To.Name == "" {
		route.Spec.To = routev1.RouteTargetReference{
			Kind: "Service",
			Name: GetServiceName(kb),
		}
	}

	routes = append(routes, *route)

	return routes, nil
}
