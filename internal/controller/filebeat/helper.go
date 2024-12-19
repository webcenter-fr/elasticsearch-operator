package filebeat

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/thoas/go-funk"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultImage = "docker.elastic.co/beats/filebeat"
)

// GetConfigMapConfigName permit to get the configMap name that store the config
func GetConfigMapConfigName(fb *beatcrd.Filebeat) (configMapName string) {
	return fmt.Sprintf("%s-config-fb", fb.Name)
}

// GetConfigMapModuleName permit to get the configMap name that store the moodules settings
func GetConfigMapModuleName(fb *beatcrd.Filebeat) (configMapName string) {
	return fmt.Sprintf("%s-module-fb", fb.Name)
}

// GetServiceName permit to get the service name
func GetServiceName(fb *beatcrd.Filebeat, serviceName string) string {
	return fmt.Sprintf("%s-%s-fb", fb.Name, serviceName)
}

// GetGlobalServiceName permit to get the global service name
func GetGlobalServiceName(fb *beatcrd.Filebeat) string {
	return fmt.Sprintf("%s-headless-fb", fb.Name)
}

// GetSecretNameForCAElasticsearch permit to get the secret name that store all Elasticsearch CA
// It return the secret name as string
func GetSecretNameForCAElasticsearch(fb *beatcrd.Filebeat) (secretName string) {
	return fmt.Sprintf("%s-ca-es-fb", fb.Name)
}

// GetPDBName permit to get the pdb name
func GetPDBName(fb *beatcrd.Filebeat) (serviceName string) {
	return fmt.Sprintf("%s-fb", fb.Name)
}

// GetStatefulsetName permit to get the statefulset name
func GetStatefulsetName(fb *beatcrd.Filebeat) (name string) {
	return fmt.Sprintf("%s-fb", fb.Name)
}

// GetContainerImage permit to get the image name
func GetContainerImage(fb *beatcrd.Filebeat) string {
	version := "latest"
	if fb.Spec.Version != "" {
		version = fb.Spec.Version
	}

	image := defaultImage
	if fb.Spec.Image != "" {
		image = fb.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, version)
}

// getLabels permit to return global label must be set on all resources
func getLabels(fb *beatcrd.Filebeat, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":                     fb.Name,
		beatcrd.FilebeatAnnotationKey: "true",
	}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}

	labels = funk.UnionStringMap(labels, fb.Labels)

	return labels
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(fb *beatcrd.Filebeat, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		beatcrd.FilebeatAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}

// GetSecretNameForCredentials permit to get the secret name that store the credentials
func GetSecretNameForCredentials(fb *beatcrd.Filebeat) (secretName string) {
	return fmt.Sprintf("%s-credential-fb", fb.Name)
}

// GetPodMonitorName return the name for podMonitor
func GetPodMonitorName(fb *beatcrd.Filebeat) string {
	return fmt.Sprintf("%s-fb", fb.Name)
}

// GetIngressName permit to get the ingress name
func GetIngressName(fb *beatcrd.Filebeat, ingressName string) string {
	return fmt.Sprintf("%s-%s-fb", fb.Name, ingressName)
}

// GetMetricbeatName return the metricbeat namme
func GetMetricbeatName(fb *beatcrd.Filebeat) (name string) {
	return fmt.Sprintf("%s-metricbeat-fb", fb.Name)
}

// GetServiceAccountName return the service account name
func GetServiceAccountName(fb *beatcrd.Filebeat) string {
	return fmt.Sprintf("%s-fb", fb.Name)
}

// GetSecretNameForPki permit to get the secret name that store PKI
// It return the secret name as string
func GetSecretNameForPki(fb *beatcrd.Filebeat) (secretName string) {
	return fmt.Sprintf("%s-pki-fb", fb.Name)
}

// GetSecretNameForTls permit to get the secret name that store all certificates for Filebeat
// It return the secret name as string
func GetSecretNameForTls(fb *beatcrd.Filebeat) (secretName string) {
	return fmt.Sprintf("%s-tls-fb", fb.Name)
}

// GetSecretNameForCALogstash permit to get the secret name that store Logstash CA
// It return the secret name as string
func GetSecretNameForCALogstash(fb *beatcrd.Filebeat) (secretName string) {
	return fmt.Sprintf("%s-ca-ls-fb", fb.Name)
}

// GetLogstashFromRef permit to get Logstash
func GetLogstashFromRef(ctx context.Context, c client.Client, o client.Object, lsRef beatcrd.FilebeatLogstashRef) (ls *logstashcrd.Logstash, err error) {
	if !lsRef.IsManaged() {
		return nil, nil
	}

	ls = &logstashcrd.Logstash{}
	target := types.NamespacedName{Name: lsRef.ManagedLogstashRef.Name}
	if lsRef.ManagedLogstashRef.Namespace != "" {
		target.Namespace = lsRef.ManagedLogstashRef.Namespace
	} else {
		target.Namespace = o.GetNamespace()
	}
	if err = c.Get(ctx, target, ls); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Error when read Logstash %s/%s", target.Namespace, target.Name)
	}

	return ls, nil
}
