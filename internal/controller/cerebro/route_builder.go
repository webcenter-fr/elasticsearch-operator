package cerebro

import (
	"emperror.dev/errors"
	"github.com/disaster37/k8sbuilder"
	routev1 "github.com/openshift/api/route/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// buildRoutes permit to generate Route object
// It return error if route spec is not provided
// It return nil if route is disabled
func buildRoutes(cb *cerebrocrd.Cerebro) (routes []*routev1.Route, err error) {
	if !cb.Spec.Endpoint.IsRouteEnabled() {
		return nil, nil
	}

	routes = make([]*routev1.Route, 0, 1)

	if cb.Spec.Endpoint.Route.Host == "" {
		return nil, errors.New("endpoint.route.host must be provided")
	}

	// Generate route
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cb.Namespace,
			Name:        GetIngressName(cb),
			Labels:      getLabels(cb, cb.Spec.Endpoint.Route.Labels),
			Annotations: getAnnotations(cb, cb.Spec.Endpoint.Route.Annotations),
		},
		Spec: routev1.RouteSpec{
			Host: cb.Spec.Endpoint.Route.Host,
			Path: "/",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: GetServiceName(cb),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
		},
	}

	// Enabled TLS
	if cb.Spec.Endpoint.Route.TlsEnabled != nil && *cb.Spec.Endpoint.Route.TlsEnabled {
		route.Spec.TLS = &routev1.TLSConfig{
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			Termination:                   routev1.TLSTerminationEdge,
		}
		if cb.Spec.Endpoint.Route.SecretRef != nil {
			route.Spec.TLS.ExternalCertificate = &routev1.LocalObjectReference{
				Name: cb.Spec.Endpoint.Route.SecretRef.Name,
			}
		}
	}

	// Merge expected route with custom route spec
	if err = k8sbuilder.MergeK8s(&route.Spec, route.Spec, cb.Spec.Endpoint.Route.RouteSpec); err != nil {
		return nil, errors.Wrap(err, "Error when merge route spec")
	}

	// Avoid to reset target service fater merge provided custom spec because of is not pointer
	if route.Spec.To.Name == "" {
		route.Spec.To = routev1.RouteTargetReference{
			Kind: "Service",
			Name: GetServiceName(cb),
		}
	}

	routes = append(routes, route)

	return routes, nil
}
