package logstash

import (
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/discover"
)

// GetSecretNameForFile permit to get secret name for file
func GetSecretNameForFile(kk *discovercrd.Logstash) string {
	return discover.GetSecretNameForFile(kk.Name, kk.Spec.TargetSecretFileRef)
}

// GetSecretNameForEnv permit to get secret name for env
func GetSecretNameForEnv(kk *discovercrd.Logstash) string {
	return discover.GetSecretNameForEnv(kk.Name, kk.Spec.TargetSecretEnvRef)
}

// getLabels permit to return global label must be set on all resources
func getLabels(kk *discovercrd.Logstash, customLabels ...map[string]string) (labels map[string]string) {
	return discover.GetLabels(discovercrd.LogstashAnnotationKey, kk.Name, kk.Labels, customLabels...)
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(kk *discovercrd.Logstash, customAnnotations ...map[string]string) (annotations map[string]string) {
	return discover.GetAnnotations(discovercrd.LogstashAnnotationKey, kk.GetInternalName(), customAnnotations...)
}

// getEnvSuffix permit to return env suffix from Logstash name
func getEnvSuffix(kk *discovercrd.Logstash) string {
	return discover.GetEnvSuffix(kk.GetInternalName())
}
