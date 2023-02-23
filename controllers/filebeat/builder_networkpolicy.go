package filebeat

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/logstash"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicies(fb *beatcrd.Filebeat) (networkPolicies []networkingv1.NetworkPolicy, err error) {

	networkPolicies = make([]networkingv1.NetworkPolicy, 0)
	tcpProtocol := v1.ProtocolTCP
	var networkPolicy *networkingv1.NetworkPolicy

	// Compute network policy to allow filebeat to access on Elasticsearch
	// Only when it not on same namspace and Elasticsearch is managed by operator
	if fb.Spec.ElasticsearchRef.IsManaged() && fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		networkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetNetworkPolicyElasticsearchName(fb),
				Namespace:   fb.Namespace,
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
			},
			Spec: networkingv1.NetworkPolicySpec{
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace,
									},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cluster": fb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name,
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
						"cluster":             fb.Name,
						FilebeatAnnotationKey: "true",
					},
				},
			},
		}

		networkPolicies = append(networkPolicies, *networkPolicy)
	}

	// Compute network policy to allow filebeat to access on Logstash
	// Only when it not on same namspace and Logstash is managed by operator
	if fb.Spec.LogstashRef.IsManaged() && fb.Spec.LogstashRef.ManagedLogstashRef.Namespace != "" {
		networkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetNetworkPolicyLogstashName(fb),
				Namespace:   fb.Namespace,
				Labels:      getLabels(fb),
				Annotations: getAnnotations(fb),
			},
			Spec: networkingv1.NetworkPolicySpec{
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": fb.Spec.LogstashRef.ManagedLogstashRef.Namespace,
									},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cluster": fb.Spec.LogstashRef.ManagedLogstashRef.Name,
										logstashcontrollers.LogstashAnnotationKey: "true",
									},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Port: &intstr.IntOrString{
									IntVal: int32(fb.Spec.LogstashRef.ManagedLogstashRef.Port),
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
						"cluster":             fb.Name,
						FilebeatAnnotationKey: "true",
					},
				},
			},
		}

		networkPolicies = append(networkPolicies, *networkPolicy)
	}

	return networkPolicies, nil
}
