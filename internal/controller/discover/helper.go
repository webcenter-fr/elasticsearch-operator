package discover

import (
	"fmt"
	"strings"

	"github.com/thoas/go-funk"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
)

// GetSecretNameForFile permit to get secret name for file
func GetSecretNameForFile(name string, targetSecretFileRef *corev1.LocalObjectReference) string {
	if targetSecretFileRef != nil && targetSecretFileRef.Name != "" {
		return targetSecretFileRef.Name
	}

	return fmt.Sprintf("%s-file", name)
}

// GetSecretNameForEnv permit to get secret name for env
func GetSecretNameForEnv(name string, targetSecretEnvRef *corev1.LocalObjectReference) string {
	if targetSecretEnvRef != nil && targetSecretEnvRef.Name != "" {
		return targetSecretEnvRef.Name
	}

	return fmt.Sprintf("%s-env", name)
}

// GetLabels permit to return global labels must be set on all resources of a discover type
func GetLabels(discoverTypeKey, name string, srcLabels map[string]string, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}
	labels = funk.UnionStringMap(labels, srcLabels)

	// Operator labels always win over user labels to keep the discover
	// controllers and the workload watchers consistent.
	return funk.UnionStringMap(labels, map[string]string{
		"discoverName":                    name,
		discoverTypeKey:                   "true",
		discovercrd.DiscoverAnnotationKey: "true",
	})
}

// GetAnnotations permit to return global annotations must be set on all resources of a discover type
func GetAnnotations(discoverTypeKey, internalName string, customAnnotations ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		discoverTypeKey:                   "true",
		discovercrd.DiscoverAnnotationKey: "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): internalName,
	}
	for _, annotation := range customAnnotations {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}

// GetEnvSuffix permit to return env suffix from discover internal name
func GetEnvSuffix(internalName string) string {
	return strings.ToUpper(strings.ReplaceAll(internalName, "-", "_"))
}
