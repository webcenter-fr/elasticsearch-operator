package kibana

import (
	"fmt"

	"github.com/thoas/go-funk"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
)

const (
	defaultImage = "docker.elastic.co/kibana/kibana"
)

// GetConfigMapName permit to get the configMap name that store the config of Kibana
func GetConfigMapName(kb *kibanaapi.Kibana) (configMapName string) {
	return fmt.Sprintf("%s-config-kb", kb.Name)
}

// GetServiceName permit to get the service name for Kibana
func GetServiceName(kb *kibanaapi.Kibana) (serviceName string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetSecretNameForTls permit to get the secret name that store all certificates for Kibana
// It return the secret name as string
func GetSecretNameForTls(kb *kibanaapi.Kibana) (secretName string) {

	if !kb.IsSelfManagedSecretForTls() {
		return kb.Spec.Tls.CertificateSecretRef.Name
	}

	return fmt.Sprintf("%s-tls-kb", kb.Name)
}

// GetSecretNameForPki permit to get the secret name that store PKI
// It return the secret name as string
func GetSecretNameForPki(kb *kibanaapi.Kibana) (secretName string) {
	return fmt.Sprintf("%s-pki-kb", kb.Name)
}

// GetSecretNameForKeystore permit to get the secret name that store the secret
// It will inject each key on keystore
// It return empty string if not secret provided
func GetSecretNameForKeystore(kb *kibanaapi.Kibana) (secretName string) {

	if kb.Spec.KeystoreSecretRef != nil {
		return kb.Spec.KeystoreSecretRef.Name
	}

	return ""
}

// GetLoadBalancerName permit to get the load balancer name
func GetLoadBalancerName(kb *kibanaapi.Kibana) (serviceName string) {
	return fmt.Sprintf("%s-lb-kb", kb.Name)
}

// GetIngressName permit to get the ingress name
func GetIngressName(kb *kibanaapi.Kibana) (ingressName string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetPDBName permit to get the pdb name
func GetPDBName(kb *kibanaapi.Kibana) (serviceName string) {
	return fmt.Sprintf("%s-kb", kb.Name)
}

// GetContainerImage permit to get the image name
func GetContainerImage(kb *kibanaapi.Kibana) string {
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
func getLabels(kb *kibanaapi.Kibana, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":           kb.Name,
		KibanaAnnotationKey: "true",
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
func getAnnotations(kb *kibanaapi.Kibana, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		KibanaAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}
