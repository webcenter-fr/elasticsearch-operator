package filebeat

import (
	routev1 "github.com/openshift/api/route/v1"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// buildRoutes permit to generate Route object
// It will overwrite the target service
func buildRoutes(fb *beatcrd.Filebeat) (routes []routev1.Route, err error) {
	routes = make([]routev1.Route, 0, len(fb.Spec.Routes))
	var route *routev1.Route

	for _, i := range fb.Spec.Routes {

		route = &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   fb.Namespace,
				Name:        GetIngressName(fb, i.Name),
				Labels:      getLabels(fb, i.Labels),
				Annotations: getAnnotations(fb, i.Annotations),
			},
			Spec: *i.Spec.DeepCopy(),
		}

		route.Spec.To = routev1.RouteTargetReference{
			Kind: "Service",
			Name: GetServiceName(fb, i.Name),
		}
		route.Spec.Port = &routev1.RoutePort{
			TargetPort: intstr.FromInt(int(i.ContainerPort)),
		}

		routes = append(routes, *route)

	}

	return routes, nil
}
