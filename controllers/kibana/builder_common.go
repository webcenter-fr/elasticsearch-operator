package kibana

import (
	"fmt"

	"github.com/thoas/go-funk"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
)

const (
	defaultImage            = "docker.elastic.co/kibana/kibana"
	defaultPrometheusPlugin = "https://github.com/pjhampton/kibana-prometheus-exporter/releases/download/%s/kibanaPrometheusExporter-%s.zip"
)

// GetConfigMapName permit to get the configMap name that store the config of Kibana
func GetConfigMapName(kb *kibanacrd.Kibana) (configMapName string) {
	return fmt.Sprintf("%s-config-kb", kb.Name)
}

// GetServiceName permit to get the service name for Kibana
func GetServiceName(kb *kibanacrd.Kibana) (serviceName string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetSecretNameForTls permit to get the secret name that store all certificates for Kibana
// It return the secret name as string
func GetSecretNameForTls(kb *kibanacrd.Kibana) (secretName string) {

	if !kb.IsSelfManagedSecretForTls() {
		return kb.Spec.Tls.CertificateSecretRef.Name
	}

	return fmt.Sprintf("%s-tls-kb", kb.Name)
}

// GetSecretNameForCAElasticsearch permit to get the secret name that store all Elasticsearch CA
// It return the secret name as string
func GetSecretNameForCAElasticsearch(kb *kibanacrd.Kibana) (secretName string) {

	return fmt.Sprintf("%s-ca-es-kb", kb.Name)
}

// GetSecretNameForPki permit to get the secret name that store PKI
// It return the secret name as string
func GetSecretNameForPki(kb *kibanacrd.Kibana) (secretName string) {
	return fmt.Sprintf("%s-pki-kb", kb.Name)
}

// GetSecretNameForKeystore permit to get the secret name that store the secret
// It will inject each key on keystore
// It return empty string if not secret provided
func GetSecretNameForKeystore(kb *kibanacrd.Kibana) (secretName string) {

	if kb.Spec.KeystoreSecretRef != nil {
		return kb.Spec.KeystoreSecretRef.Name
	}

	return ""
}

// GetLoadBalancerName permit to get the load balancer name
func GetLoadBalancerName(kb *kibanacrd.Kibana) (serviceName string) {
	return fmt.Sprintf("%s-lb-kb", kb.Name)
}

// GetIngressName permit to get the ingress name
func GetIngressName(kb *kibanacrd.Kibana) (ingressName string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetPDBName permit to get the pdb name
func GetPDBName(kb *kibanacrd.Kibana) (serviceName string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetDeploymentName permit to get the deployement name
func GetDeploymentName(kb *kibanacrd.Kibana) (name string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetContainerImage permit to get the image name
func GetContainerImage(kb *kibanacrd.Kibana) string {
	version := "latest"
	if kb.Spec.Version != "" {
		version = kb.Spec.Version
	}

	image := defaultImage
	if kb.Spec.Image != "" {
		image = kb.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, version)
}

// getLabels permit to return global label must be set on all resources
func getLabels(kb *kibanacrd.Kibana, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":                     kb.Name,
		kibanacrd.KibanaAnnotationKey: "true",
	}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}

	labels = funk.UnionStringMap(labels, kb.Labels)

	return labels
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(kb *kibanacrd.Kibana, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		kibanacrd.KibanaAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}

// GetSecretNameForCredentials permit to get the secret name that store the credentials
func GetSecretNameForCredentials(kb *kibanacrd.Kibana) (secretName string) {
	return fmt.Sprintf("%s-credential-kb", kb.Name)
}

// GetNetworkPolicyName return the name for network policy
func GetNetworkPolicyName(kb *kibanacrd.Kibana) string {
	return fmt.Sprintf("%s-allow-api-kb", kb.Name)
}

// GetNetworkPolicyName return the name for network policy
func GetNetworkPolicyElasticsearchName(kb *kibanacrd.Kibana) string {
	return fmt.Sprintf("%s-allow-es-kb", kb.Name)
}

// GetExporterUrl permit to get the URL to download Kibana plugin for prometheus exporter
func GetExporterUrl(kb *kibanacrd.Kibana) string {
	if kb.Spec.Monitoring.Prometheus != nil && kb.Spec.Monitoring.Prometheus.Url != "" {
		return kb.Spec.Monitoring.Prometheus.Url
	}

	if kb.Spec.Version != "" && kb.Spec.Version != "latest" {
		return fmt.Sprintf(defaultPrometheusPlugin, kb.Spec.Version, kb.Spec.Version)
	}

	return fmt.Sprintf(defaultPrometheusPlugin, "8.6.0", "8.6.0")
}

// GetPodMonitorName return the name for podMonitor
func GetPodMonitorName(kb *kibanacrd.Kibana) string {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetMetricbeatName return the metricbeat namme
func GetMetricbeatName(kb *kibanacrd.Kibana) (name string) {
	return fmt.Sprintf("%s-metricbeat-kb", kb.Name)
}
