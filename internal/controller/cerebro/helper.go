package cerebro

import (
	"fmt"

	"github.com/thoas/go-funk"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
)

const (
	defaultImage   = "ghcr.io/disaster37/cerebro"
	defaultVersion = "v0.1.0"

	// legacyVersion is the value the old CRD defaulted `spec.version` to. Existing
	// Cerebro resources have it persisted, and no `latest` tag exists upstream, so it
	// must be remapped to avoid ImagePullBackOff on upgrade.
	legacyVersion = "latest"
)

// GetConfigMapName permit to get the configMap name that store the config
func GetConfigMapName(cb *cerebrocrd.Cerebro) (configMapName string) {
	return fmt.Sprintf("%s-config-cb", cb.Name)
}

// GetSecretNameForApplication permit to get the secret name for application
func GetSecretNameForApplication(cb *cerebrocrd.Cerebro) (secretName string) {
	return fmt.Sprintf("%s-application-cb", cb.Name)
}

// GetServiceName permit to get the service name
func GetServiceName(cb *cerebrocrd.Cerebro) (serviceName string) {
	return fmt.Sprintf("%s-cb", cb.Name)
}

// GetLoadBalancerName permit to get the load balancer name
func GetLoadBalancerName(cb *cerebrocrd.Cerebro) (serviceName string) {
	return fmt.Sprintf("%s-lb-cb", cb.Name)
}

// GetIngressName permit to get the ingress name
func GetIngressName(cb *cerebrocrd.Cerebro) (ingressName string) {
	return fmt.Sprintf("%s-cb", cb.Name)
}

// GetDeploymentName permit to get the deployment name
func GetDeploymentName(cb *cerebrocrd.Cerebro) (name string) {
	return fmt.Sprintf("%s-cb", cb.Name)
}

// GetContainerImage permit to get the image name
func GetContainerImage(cb *cerebrocrd.Cerebro) string {
	image := defaultImage
	if cb.Spec.Image != "" {
		image = cb.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, effectiveVersion(cb))
}

// effectiveVersion returns the resolved image version tag used for the container
// (remapping empty or legacy "latest" to defaultVersion, matching GetContainerImage).
func effectiveVersion(cb *cerebrocrd.Cerebro) string {
	if cb == nil || cb.Spec.Version == "" || cb.Spec.Version == legacyVersion {
		return defaultVersion
	}
	return cb.Spec.Version
}

// isKnownPreHotReloadVersion reports whether version is one of the known Cerebro
// tags released prior to config hot-reload support.
// NOTE: Review this list at release time when defaultVersion is bumped to a
// hot-reload-capable Cerebro release.
func isKnownPreHotReloadVersion(version string) bool {
	switch version {
	case "", legacyVersion, "v0.1.0", "0.1.0", "0.0.6", "v0.0.6", "0.0.5", "v0.0.5", "0.0.4", "v0.0.4":
		return true
	default:
		return false
	}
}

// shouldRolloutOnGeneratedConfigChange reports whether generated ConfigMap
// changes should roll the Deployment. Only the explicit Restart strategy
// does so; the zero value behaves as the HotReload default.
func shouldRolloutOnGeneratedConfigChange(cb *cerebrocrd.Cerebro) bool {
	if cb == nil {
		return false
	}
	return cb.Spec.Deployment.ConfigUpdateStrategy == cerebrocrd.ConfigUpdateStrategyRestart
}

// IsLegacyVersion returns true when spec.version still carries the obsolete
// `latest` value inherited from the previous CRD default.
func IsLegacyVersion(cb *cerebrocrd.Cerebro) bool {
	return cb.Spec.Version == legacyVersion
}

// getLabels permit to return global label must be set on all resources
func getLabels(cb *cerebrocrd.Cerebro, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":                       cb.Name,
		cerebrocrd.CerebroAnnotationKey: "true",
	}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}

	labels = funk.UnionStringMap(labels, cb.Labels)

	return labels
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(cb *cerebrocrd.Cerebro, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		cerebrocrd.CerebroAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}
