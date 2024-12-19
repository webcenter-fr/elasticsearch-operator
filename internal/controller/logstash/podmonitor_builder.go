package logstash

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPodMonitor permit to build pod monitor
// It return nil if prometheus monitoring is disabled
func buildPodMonitors(ls *logstashcrd.Logstash) (podMonitors []monitoringv1.PodMonitor, err error) {
	if !ls.Spec.Monitoring.IsPrometheusMonitoring() {
		return nil, nil
	}

	podMonitors = make([]monitoringv1.PodMonitor, 0, 1)
	scrapInterval := "10s"
	if ls.Spec.Monitoring.Prometheus.ScrapInterval != nil && *ls.Spec.Monitoring.Prometheus.ScrapInterval != "" {
		scrapInterval = *ls.Spec.Monitoring.Prometheus.ScrapInterval
	}

	podMonitor := monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetPodMonitorName(ls),
			Namespace:   ls.Namespace,
			Labels:      getLabels(ls),
			Annotations: getAnnotations(ls),
		},
		Spec: monitoringv1.PodMonitorSpec{
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
				{
					Port:     "exporter",
					Interval: monitoringv1.Duration(scrapInterval),
					Path:     "/metrics",
					Scheme:   "http",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                         ls.Name,
					logstashcrd.LogstashAnnotationKey: "true",
				},
			},
		},
	}

	podMonitors = append(podMonitors, podMonitor)

	return podMonitors, nil
}
