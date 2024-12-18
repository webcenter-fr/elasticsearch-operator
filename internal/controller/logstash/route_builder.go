package logstash

import (
	routev1 "github.com/openshift/api/route/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// buildRoutes permit to generate Route object
// It will overwrite the target service
func buildRoutes(ls *logstashcrd.Logstash) (routes []routev1.Route, err error) {
	routes = make([]routev1.Route, 0, len(ls.Spec.Routes))
	var route *routev1.Route

	for _, i := range ls.Spec.Routes {

		route = &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ls.Namespace,
				Name:        GetIngressName(ls, i.Name),
				Labels:      getLabels(ls, i.Labels),
				Annotations: getAnnotations(ls, i.Annotations),
			},
			Spec: *i.Spec.DeepCopy(),
		}

		route.Spec.To = routev1.RouteTargetReference{
			Kind: "Service",
			Name: GetServiceName(ls, i.Name),
		}
		route.Spec.Port = &routev1.RoutePort{
			TargetPort: intstr.FromInt(int(i.ContainerPort)),
		}

		routes = append(routes, *route)

	}

	return routes, nil
}
