package elasticsearch

import (
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GenerateLoadbalancer permit to generate Loadbalancer throught service
// It return nil if Loadbalancer is disabled
func BuildLoadbalancer(es *elasticsearchcrd.Elasticsearch) (service *corev1.Service, err error) {

	if !es.IsLoadBalancerEnabled() {
		return nil, nil
	}

	selector := map[string]string{
		"cluster": es.Name,
		elasticsearchcrd.ElasticsearchAnnotationKey: "true",
	}
	if es.Spec.Endpoint.LoadBalancer.TargetNodeGroupName != "" {
		// Check the node group specified exist
		isFound := false
		for _, nodeGroup := range es.Spec.NodeGroups {
			if nodeGroup.Name == es.Spec.Endpoint.LoadBalancer.TargetNodeGroupName {
				isFound = true
				break
			}
		}
		if !isFound {
			return nil, errors.Errorf("The target group name '%s' not found", es.Spec.Endpoint.LoadBalancer.TargetNodeGroupName)
		}

		selector["nodeGroup"] = es.Spec.Endpoint.LoadBalancer.TargetNodeGroupName
	}

	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   es.Namespace,
			Name:        GetLoadBalancerName(es),
			Labels:      getLabels(es),
			Annotations: getAnnotations(es),
		},
		Spec: corev1.ServiceSpec{
			Type:            corev1.ServiceTypeLoadBalancer,
			SessionAffinity: corev1.ServiceAffinityNone,
			Selector:        selector,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       9200,
					TargetPort: intstr.FromInt(9200),
				},
			},
		},
	}

	return service, nil
}
