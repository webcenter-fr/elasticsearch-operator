package elasticsearch

import (
	"os"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildNetworkPolicy permit to generate Network policy object
func BuildNetworkPolicy(es *elasticsearchcrd.Elasticsearch, listCaller []client.Object) (networkPolicy *networkingv1.NetworkPolicy, err error) {
	var npp networkingv1.NetworkPolicyPeer

	npps := make([]networkingv1.NetworkPolicyPeer, 0, len(listCaller)+1)

	// Compute to allow remote referer to access on Elasticsearch
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
		case "Logstash":
			npp.PodSelector.MatchLabels[logstashcrd.LogstashAnnotationKey] = "true"
		case "Filebeat":
			npp.PodSelector.MatchLabels[beatcrd.FilebeatAnnotationKey] = "true"
		case "Metricbeat":
			npp.PodSelector.MatchLabels[beatcrd.MetricbeatAnnotationKey] = "true"
		case "Kibana":
			npp.PodSelector.MatchLabels[kibanacrd.KibanaAnnotationKey] = "true"
		}
		npps = append(npps, npp)
	}

	// Compute to allow operator to access on Elasticsearch
	npp = networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{},
	}

	_, found := os.LookupEnv("POD_NAME")
	if found {
		npp.PodSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"elasticsearch-operator.k8s.webcenter.fr": "true",
			},
		}
	}

	namespace, found := os.LookupEnv("POD_NAMESPACE")
	if found {
		npp.NamespaceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": namespace,
			},
		}
	}

	npps = append(npps, npp)

	tcpProtocol := v1.ProtocolTCP
	networkPolicy = &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNetworkPolicyName(es),
			Namespace:   es.Namespace,
			Labels:      getLabels(es),
			Annotations: getAnnotations(es),
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: npps,
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
				networkingv1.PolicyTypeIngress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster": es.Name,
					elasticsearchcrd.ElasticsearchAnnotationKey: "true",
				},
			},
		},
	}

	return networkPolicy, nil
}
