package shared

import corev1 "k8s.io/api/core/v1"

type MetricbeatMonitoringSpec struct {
	// Enabled permit to enable Metricbeat monitoring
	// It will deploy metricbeat to collect metric
	// Default to false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef ElasticsearchRef `json:"elasticsearchRef"`

	// Resources permit to set resources on metricbeat
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"initContainerResources,omitempty"`

	// RefreshPeriod permit to set the time to collect data
	// Defaullt to  `10s`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	RefreshPeriod string `json:"refreshPeriod,omitempty"`
}