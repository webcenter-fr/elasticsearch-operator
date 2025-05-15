package elasticsearch

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// BuildPodMonitor permit to build pod monitor
// It return nil if prometheus monitoring is disabled
func buildPodMonitors(es *elasticsearchcrd.Elasticsearch) (podMonitors []*monitoringv1.PodMonitor, err error) {
	if !es.Spec.Monitoring.IsPrometheusMonitoring() {
		return nil, nil
	}

	podMonitors = []*monitoringv1.PodMonitor{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        GetPodMonitorName(es),
				Namespace:   es.Namespace,
				Labels:      getLabels(es),
				Annotations: getAnnotations(es),
			},
			Spec: monitoringv1.PodMonitorSpec{
				PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
					{
						Port:     ptr.To("exporter"),
						Interval: "10s",
					},
				},
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"exporter":      "true",
						"elasticsearch": "true",
					},
				},
			},
		},
	}

	return podMonitors, nil
}
