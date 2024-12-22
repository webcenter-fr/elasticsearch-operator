package metricbeat

import (
	"fmt"

	"github.com/thoas/go-funk"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
)

const (
	defaultImage = "docker.elastic.co/beats/metricbeat"
)

// GetConfigMapConfigName permit to get the configMap name that store the config
func GetConfigMapConfigName(mb *beatcrd.Metricbeat) (configMapName string) {
	return fmt.Sprintf("%s-config-mb", mb.Name)
}

// GetConfigMapModuleName permit to get the configMap name that store the modules settings
func GetConfigMapModuleName(mb *beatcrd.Metricbeat) (configMapName string) {
	return fmt.Sprintf("%s-module-mb", mb.Name)
}

// GetGlobalServiceName permit to get the global service name
func GetGlobalServiceName(mb *beatcrd.Metricbeat) string {
	return fmt.Sprintf("%s-headless-mb", mb.Name)
}

// GetSecretNameForCAElasticsearch permit to get the secret name that store all Elasticsearch CA
// It return the secret name as string
func GetSecretNameForCAElasticsearch(mb *beatcrd.Metricbeat) (secretName string) {
	return fmt.Sprintf("%s-ca-es-mb", mb.Name)
}

// GetPDBName permit to get the pdb name
func GetPDBName(mb *beatcrd.Metricbeat) (serviceName string) {
	return fmt.Sprintf("%s-mb", mb.Name)
}

// GetStatefulsetName permit to get the statefulset name
func GetStatefulsetName(mb *beatcrd.Metricbeat) (name string) {
	return fmt.Sprintf("%s-mb", mb.Name)
}

// GetContainerImage permit to get the image name
func GetContainerImage(mb *beatcrd.Metricbeat) string {
	version := "latest"
	if mb.Spec.Version != "" {
		version = mb.Spec.Version
	}

	image := defaultImage
	if mb.Spec.Image != "" {
		image = mb.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, version)
}

// getLabels permit to return global label must be set on all resources
func getLabels(mb *beatcrd.Metricbeat, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":                       mb.Name,
		beatcrd.MetricbeatAnnotationKey: "true",
	}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}

	labels = funk.UnionStringMap(labels, mb.Labels)

	return labels
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(mb *beatcrd.Metricbeat, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		beatcrd.MetricbeatAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}

// GetSecretNameForCredentials permit to get the secret name that store the credentials
func GetSecretNameForCredentials(mb *beatcrd.Metricbeat) (secretName string) {
	return fmt.Sprintf("%s-credential-mb", mb.Name)
}

// GetNetworkPolicyElasticsearchName return the name for network policy to access on Elasticsearch
func GetNetworkPolicyElasticsearchName(mb *beatcrd.Metricbeat) string {
	return fmt.Sprintf("%s-allow-es-mb", mb.Name)
}

// GetServiceAccountName return the service account name
func GetServiceAccountName(mb *beatcrd.Metricbeat) string {
	return fmt.Sprintf("%s-mb", mb.Name)
}
