package kibana

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPodMonitor permit to build pod monitor
// It return nil if prometheus monitoring is disabled
func BuildPodMonitor(kb *kibanacrd.Kibana) (podMonitor *monitoringv1.PodMonitor, err error) {
	if !kb.IsPrometheusMonitoring() {
		return nil, nil
	}

	podMonitor = &monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetPodMonitorName(kb),
			Namespace:   kb.Namespace,
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Spec: monitoringv1.PodMonitorSpec{
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
				{
					Port:     "http",
					Interval: "10s",
					Path:     "_prometheus/metrics",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":           kb.Name,
					KibanaAnnotationKey: "true",
				},
			},
		},
	}

	return podMonitor, nil
}
