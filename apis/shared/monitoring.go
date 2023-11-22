package shared

import corev1 "k8s.io/api/core/v1"

// MonitoringSpec permit to set monitoring
type MonitoringSpec struct {

	// Prometheus permit to monitor cluster with Prometheus and graphana (via exporter)
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Prometheus *MonitoringPrometheusSpec `json:"prometheus,omitempty"`

	// Metricbeat permit to monitor cluster with metricbeat and to dedicated monitoring cluster
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Metricbeat *MonitoringMetricbeatSpec `json:"metricbeat,omitempty"`
}

// MonitoringPrometheusSpec permit to set Prometheus
type MonitoringPrometheusSpec struct {

	// Enabled permit to enable Prometheus monitoring
	// It will deploy exporter and add podMonitor policy
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Url is the plugin URL where to download exporter
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Url string `json:"url,omitempty"`

	// The docker Image if not URL
	ImageSpec `json:",inline"`

	// Version is the exporter version to use
	// Default is use the latest
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`

	// Resources permit to set ressources on Prometheus expporter container
	// If not defined, it will use the default requirements
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// MonitoringMetricbeatSpec permit to set metricbeat
type MonitoringMetricbeatSpec struct {
	// Enabled permit to enable Metricbeat monitoring
	// It will deploy metricbeat to collect metric
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ElasticsearchRef ElasticsearchRef `json:"elasticsearchRef"`

	// Resources permit to set resources on metricbeat
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// RefreshPeriod permit to set the time to collect data
	// Defaullt to  `10s`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default="10s"
	RefreshPeriod string `json:"refreshPeriod,omitempty"`

	// Version is the Metricbeat version to use
	// Default it use the same version of the product that it monitor
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Version string `json:"version,omitempty"`

	// NumberOfReplica is the number of replica to set on metricbeat setting when it create templates
	// Default is set to 0
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=0
	NumberOfReplica int64 `json:"numberOfReplicas,omitempty"`
}

// IsPrometheusMonitoring return true if Prometheus monitoring is enabled
func (h MonitoringSpec) IsPrometheusMonitoring() bool {

	if h.Prometheus != nil && h.Prometheus.Enabled {
		return true
	}

	return false
}

// IsMetricbeatMonitoring return true if Metricbeat monitoring is enabled
func (h *MonitoringSpec) IsMetricbeatMonitoring(numberOfReplicas int32) bool {

	if h.Metricbeat != nil && h.Metricbeat.Enabled && numberOfReplicas > 0 {
		return true
	}

	return false
}
