package logstash

import (
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicy(ls *logstashcrd.Logstash, listCaller []client.Object) (networkPolicy *networkingv1.NetworkPolicy, err error) {

	if len(listCaller) == 0 {
		return nil, nil
	}

	tcpProtocol := v1.ProtocolTCP
	var npp networkingv1.NetworkPolicyPeer
	ports := make([]networkingv1.NetworkPolicyPort, 0)
	npps := make([]networkingv1.NetworkPolicyPeer, 0, len(listCaller))

	// Compute to allow remote referer to access on Logstash
	for _, o := range listCaller {
		npp = networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": o.GetNamespace(),
				},
			},
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": o.GetName(),
				},
			},
		}

		switch o.GetObjectKind().GroupVersionKind().Kind {
		case "Filebeat":
			npp.PodSelector.MatchLabels[beatcrd.FilebeatAnnotationKey] = "true"
			fb := o.(*beatcrd.Filebeat)
			isFound := false
			for _, port := range ports {
				if fb.Spec.LogstashRef.ManagedLogstashRef.Port == int64(port.Port.IntValue()) {
					isFound = true
				}
			}
			if !isFound {
				ports = append(ports, networkingv1.NetworkPolicyPort{
					Protocol: &tcpProtocol,
					Port: &intstr.IntOrString{
						IntVal: int32(fb.Spec.LogstashRef.ManagedLogstashRef.Port),
					},
				})
			}
		}
		npps = append(npps, npp)
	}

	networkPolicy = &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNetworkPolicyName(ls),
			Namespace:   ls.Namespace,
			Labels:      getLabels(ls),
			Annotations: getAnnotations(ls),
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From:  npps,
					Ports: ports,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                         ls.Name,
					logstashcrd.LogstashAnnotationKey: "true",
				},
			},
		},
	}

	return networkPolicy, nil
}
