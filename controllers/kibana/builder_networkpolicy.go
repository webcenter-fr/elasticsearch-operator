package kibana

import (
	"os"

	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicies(kb *kibanacrd.Kibana) (networkPolicies []networkingv1.NetworkPolicy, err error) {

	networkPolicies = make([]networkingv1.NetworkPolicy, 0, 1)
	tcpProtocol := v1.ProtocolTCP

	// Compute network policy to allow operator to access on Kibana API
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNetworkPolicyName(kb),
			Namespace:   kb.Namespace,
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port: &intstr.IntOrString{
								IntVal: 5601,
							},
							Protocol: &tcpProtocol,
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": kb.Name,
					KibanaAnnotationKey: "true",
				},
			},
		},
	}

	_, found := os.LookupEnv("POD_NAME")
	if found {
		networkPolicy.Spec.Ingress[0].From[0].PodSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kibana-operator.k8s.webcenter.fr": "true",
			},
		}
	}

	namespace, found := os.LookupEnv("POD_NAMESPACE")
	if found {
		networkPolicy.Spec.Ingress[0].From[0].NamespaceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": namespace,
			},
		}
	}

	networkPolicies = append(networkPolicies, *networkPolicy)

	// Compute network policy to allow kibana to access on Elasticsearch
	// Only when it not on same namspace and Elasticsearch is managed by operator
	if kb.Spec.ElasticsearchRef.IsManaged() && kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace != "" {
		networkPolicy = &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetNetworkPolicyElasticsearchName(kb),
				Namespace:   kb.Namespace,
				Labels:      getLabels(kb),
				Annotations: getAnnotations(kb),
			},
			Spec: networkingv1.NetworkPolicySpec{
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace,
									},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cluster": kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name,
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
						"cluster": kb.Name,
						KibanaAnnotationKey: "true",
					},
				},
			},
		}

		networkPolicies = append(networkPolicies, *networkPolicy)
	}

	return networkPolicies, nil
}
