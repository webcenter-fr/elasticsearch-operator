package kibana

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPodMonitor permit to build pod monitor
// It return nil if prometheus monitoring is disabled
func buildPodMonitors(kb *kibanacrd.Kibana) (podMonitors []monitoringv1.PodMonitor, err error) {
	if !kb.Spec.Monitoring.IsPrometheusMonitoring() {
		return nil, nil
	}
	scheme := "https"
	if !kb.IsTlsEnabled() {
		scheme = "http"
	}

	podMonitors = []monitoringv1.PodMonitor{
		{
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
						BasicAuth: &monitoringv1.BasicAuth{
							Username: corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: GetSecretNameForCredentials(kb),
								},
								Key: "username",
							},
							Password: corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: GetSecretNameForCredentials(kb),
								},
								Key: "kibana_system",
							},
						},
						Scheme: scheme,
						TLSConfig: &monitoringv1.PodMetricsEndpointTLSConfig{
							SafeTLSConfig: monitoringv1.SafeTLSConfig{
								InsecureSkipVerify: true,
							},
						},
					},
				},
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                     kb.Name,
						kibanacrd.KibanaAnnotationKey: "true",
					},
				},
			},
		},
	}

	return podMonitors, nil
}
