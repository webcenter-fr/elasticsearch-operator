package metricbeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicies(mb *beatcrd.Metricbeat) (networkPolicies []networkingv1.NetworkPolicy, err error) {

	networkPolicies = make([]networkingv1.NetworkPolicy, 0)
	tcpProtocol := v1.ProtocolTCP
	var networkPolicy *networkingv1.NetworkPolicy

	// Compute network policy to allow metricbeat to access on Elasticsearch
	// Only when it not on same namspace and Elasticsearch is managed by operator
	if mb.Spec.ElasticsearchRef.IsManaged() && mb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		networkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetNetworkPolicyElasticsearchName(mb),
				Namespace:   mb.Namespace,
				Labels:      getLabels(mb),
				Annotations: getAnnotations(mb),
			},
			Spec: networkingv1.NetworkPolicySpec{
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": mb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace,
									},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cluster": mb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name,
										elasticsearchcontrollers.ElasticsearchAnnotationKey: "true",
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Port: &intstr.IntOrString{
									IntVal: 9200,
								},
								Protocol: &tcpProtocol,
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":               mb.Name,
						MetricbeatAnnotationKey: "true",
					},
				},
			},
		}

		networkPolicies = append(networkPolicies, *networkPolicy)
	}

	return networkPolicies, nil
}
