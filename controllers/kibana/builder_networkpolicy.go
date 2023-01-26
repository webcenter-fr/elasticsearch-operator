package kibana

import (
	"os"

	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicy(kb *kibanacrd.Kibana) (networkPolicy *networkingv1.NetworkPolicy, err error) {

	tcpProtocol := v1.ProtocolTCP
	networkPolicy = &networkingv1.NetworkPolicy{
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
							PodSelector: &metav1.LabelSelector{},
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
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
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
				"name": namespace,
			},
		}
	}

	return networkPolicy, nil
}
