package logstash

import (
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicies(ls *logstashcrd.Logstash) (networkPolicies []networkingv1.NetworkPolicy, err error) {

	networkPolicies = make([]networkingv1.NetworkPolicy, 0)
	tcpProtocol := v1.ProtocolTCP
	var networkPolicy *networkingv1.NetworkPolicy

	// Compute network policy to allow logstash to access on Elasticsearch
	// Only when it not on same namspace and Elasticsearch is managed by operator
	if ls.Spec.ElasticsearchRef.IsManaged() && ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		networkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetNetworkPolicyElasticsearchName(ls),
				Namespace:   ls.Namespace,
				Labels:      getLabels(ls),
				Annotations: getAnnotations(ls),
			},
			Spec: networkingv1.NetworkPolicySpec{
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"name": ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace,
									},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cluster": ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name,
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
			},
		}

		networkPolicies = append(networkPolicies, *networkPolicy)
	}

	return networkPolicies, nil
}
