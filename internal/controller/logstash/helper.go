package logstash

import (
	"fmt"

	"github.com/thoas/go-funk"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
)

const (
	defaultImage           = "docker.elastic.co/logstash/logstash"
	defaultExporterImage   = "ghcr.io/kuskoman/logstash-exporter"
	defaultExporterVersion = "latest"
)

// GetConfigMapConfigName permit to get the configMap name that store the config
func GetConfigMapConfigName(ls *logstashcrd.Logstash) (configMapName string) {
	return fmt.Sprintf("%s-config-ls", ls.Name)
}

// GetConfigMapPipelineName permit to get the configMap name that store the piepline
func GetConfigMapPipelineName(ls *logstashcrd.Logstash) (configMapName string) {
	return fmt.Sprintf("%s-pipeline-ls", ls.Name)
}

// GetConfigMapPatternName permit to get the configMap name that store the pattern
func GetConfigMapPatternName(ls *logstashcrd.Logstash) (configMapName string) {
	return fmt.Sprintf("%s-pattern-ls", ls.Name)
}

// GetServiceName permit to get the service name
func GetServiceName(ls *logstashcrd.Logstash, serviceName string) string {
	return fmt.Sprintf("%s-%s-ls", ls.Name, serviceName)
}

// GetGlobalServiceName pemrit to get the global service name
func GetGlobalServiceName(ls *logstashcrd.Logstash) string {
	return fmt.Sprintf("%s-headless-ls", ls.Name)
}

// GetSecretNameForCAElasticsearch permit to get the secret name that store all Elasticsearch CA
// It return the secret name as string
func GetSecretNameForCAElasticsearch(ls *logstashcrd.Logstash) (secretName string) {
	return fmt.Sprintf("%s-ca-es-ls", ls.Name)
}

// GetSecretNameForKeystore permit to get the secret name that store the secret
// It will inject each key on keystore
// It return empty string if not secret provided
func GetSecretNameForKeystore(ls *logstashcrd.Logstash) (secretName string) {
	if ls.Spec.KeystoreSecretRef != nil {
		return ls.Spec.KeystoreSecretRef.Name
	}

	return ""
}

// GetPDBName permit to get the pdb name
func GetPDBName(ls *logstashcrd.Logstash) (serviceName string) {
	return fmt.Sprintf("%s-ls", ls.Name)
}

// GetStatefulsetName permit to get the statefulset name
func GetStatefulsetName(ls *logstashcrd.Logstash) (name string) {
	return fmt.Sprintf("%s-ls", ls.Name)
}

// GetContainerImage permit to get the image name
func GetContainerImage(ls *logstashcrd.Logstash) string {
	version := "latest"
	if ls.Spec.Version != "" {
		version = ls.Spec.Version
	}

	image := defaultImage
	if ls.Spec.Image != "" {
		image = ls.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, version)
}

// GetSecretNameForPki permit to get the secret name that store PKI
// It return the secret name as string
func GetSecretNameForPki(ls *logstashcrd.Logstash) (secretName string) {
	return fmt.Sprintf("%s-pki-ls", ls.Name)
}

// GetSecretNameForTls permit to get the secret name that store all certificates for Logstash
// It return the secret name as string
func GetSecretNameForTls(ls *logstashcrd.Logstash) (secretName string) {
	return fmt.Sprintf("%s-tls-ls", ls.Name)
}

// getLabels permit to return global label must be set on all resources
func getLabels(ls *logstashcrd.Logstash, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":                         ls.Name,
		logstashcrd.LogstashAnnotationKey: "true",
	}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}

	labels = funk.UnionStringMap(labels, ls.Labels)

	return labels
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(ls *logstashcrd.Logstash, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		logstashcrd.LogstashAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}

// GetSecretNameForCredentials permit to get the secret name that store the credentials
func GetSecretNameForCredentials(ls *logstashcrd.Logstash) (secretName string) {
	return fmt.Sprintf("%s-credential-ls", ls.Name)
}

// GetNetworkPolicyName return the name for network policy
func GetNetworkPolicyName(ls *logstashcrd.Logstash) string {
	return fmt.Sprintf("%s-allow-ls", ls.Name)
}

// GetPodMonitorName return the name for podMonitor
func GetPodMonitorName(ls *logstashcrd.Logstash) string {
	return fmt.Sprintf("%s-ls", ls.Name)
}

// GetIngressName permit to get the ingress name
func GetIngressName(ls *logstashcrd.Logstash, ingressName string) string {
	return fmt.Sprintf("%s-%s-ls", ls.Name, ingressName)
}

// GetMetricbeatName return the metricbeat namme
func GetMetricbeatName(ls *logstashcrd.Logstash) (name string) {
	return fmt.Sprintf("%s-metricbeat-ls", ls.Name)
}

// GetServiceAccountName return the service account name
func GetServiceAccountName(ls *logstashcrd.Logstash) string {
	return fmt.Sprintf("%s-ls", ls.Name)
}

// GetConfigMapExporterName permit to get the configMap name that store the config for exporter
func GetConfigMapExporterName(ls *logstashcrd.Logstash) (configMapName string) {
	return fmt.Sprintf("%s-exporter-ls", ls.Name)
}
