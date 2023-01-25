package elasticsearch

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPodMonitor permit to build pod monitor
// It return nil if prometheus monitoring is disabled
func BuildPodMonitor(es *elasticsearchcrd.Elasticsearch) (podMonitor *monitoringv1.PodMonitor, err error) {
	if !es.IsPrometheusMonitoring() {
		return nil, nil
	}

	podMonitor = &monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetExporterDeployementName(es),
			Namespace:   es.Namespace,
			Labels:      getLabels(es),
			Annotations: getAnnotations(es),
		},
		Spec: monitoringv1.PodMonitorSpec{
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
				{
					Port: "exporter",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                  es.Name,
					ElasticsearchAnnotationKey: "true",
				},
			},
		},
	}

	return podMonitor, nil
}
