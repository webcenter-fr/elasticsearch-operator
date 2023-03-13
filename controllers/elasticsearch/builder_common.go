package elasticsearch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thoas/go-funk"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
)

const (
	defaultImage         = "docker.elastic.co/elasticsearch/elasticsearch"
	defaultExporterImage = "quay.io/prometheuscommunity/elasticsearch-exporter"
)

// GetNodeGroupName permit to get the node group name
func GetNodeGroupName(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) (name string) {
	return fmt.Sprintf("%s-%s-es", elasticsearch.Name, nodeGroupName)
}

// GetNodeGroupNodeNames permit to get node names that composed the node group
func GetNodeGroupNodeNames(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) (nodeNames []string) {

	for _, nodeGroup := range elasticsearch.Spec.NodeGroups {
		if nodeGroup.Name == nodeGroupName {
			nodeNames = make([]string, 0, nodeGroup.Replicas)

			for i := 0; i < int(nodeGroup.Replicas); i++ {
				nodeNames = append(nodeNames, fmt.Sprintf("%s-%d", GetNodeGroupName(elasticsearch, nodeGroup.Name), i))
			}

			return nodeNames
		}
	}

	return nil
}

// GetConfigMapNameForConfig permit to get the configMap name that store the config of Elasticsearch
func GetNodeGroupConfigMapName(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) (configMapName string) {
	return fmt.Sprintf("%s-%s-config-es", elasticsearch.Name, nodeGroupName)
}

// GetNodeGroupServiceName permit to get the service name for specified node group name
func GetNodeGroupServiceName(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) (serviceName string) {
	return GetNodeGroupName(elasticsearch, nodeGroupName)
}

// GetNodeGroupServiceNameHeadless permit to get the service name headless for specified node group name
func GetNodeGroupServiceNameHeadless(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) (serviceName string) {
	return fmt.Sprintf("%s-%s-headless-es", elasticsearch.Name, nodeGroupName)
}

// GetNodeNames permit to get all nodes names
// It return the list with all node names (DNS / pod name)
func GetNodeNames(elasticsearch *elasticsearchcrd.Elasticsearch) (nodeNames []string) {
	nodeNames = make([]string, 0)

	for _, nodeGroup := range elasticsearch.Spec.NodeGroups {
		nodeNames = append(nodeNames, GetNodeGroupNodeNames(elasticsearch, nodeGroup.Name)...)
	}

	return nodeNames
}

// GetSecretNameForTlsTransport permit to get the secret name that store all certificates for transport layout
// It return the secret name as string
func GetSecretNameForTlsTransport(elasticsearch *elasticsearchcrd.Elasticsearch) (secretName string) {
	return fmt.Sprintf("%s-tls-transport-es", elasticsearch.Name)
}

// GetSecretNameForPkiTransport permit to get the secret name that store PKI for transport layer
// It return the secret name as string
func GetSecretNameForPkiTransport(elasticsearch *elasticsearchcrd.Elasticsearch) (secretName string) {
	return fmt.Sprintf("%s-pki-transport-es", elasticsearch.Name)
}

// GetSecretNameForTlsApi permit to get the secret name that store all certificates for Api layout (Http endpoint)
// It return the secret name as string
func GetSecretNameForTlsApi(elasticsearch *elasticsearchcrd.Elasticsearch) (secretName string) {

	if !elasticsearch.IsSelfManagedSecretForTlsApi() {
		return elasticsearch.Spec.Tls.CertificateSecretRef.Name
	}

	return fmt.Sprintf("%s-tls-api-es", elasticsearch.Name)
}

// GetSecretNameForPkiApi permit to get the secret name that store PKI for API layer
// It return the secret name as string
func GetSecretNameForPkiApi(elasticsearch *elasticsearchcrd.Elasticsearch) (secretName string) {
	return fmt.Sprintf("%s-pki-api-es", elasticsearch.Name)
}

// GetSecretNameForCredentials permit to get the secret name that store the credentials
func GetSecretNameForCredentials(elasticsearch *elasticsearchcrd.Elasticsearch) (secretName string) {
	return fmt.Sprintf("%s-credential-es", elasticsearch.Name)
}

// GetSecretNameForKeystore permit to get the secret name that store the secret
// It will inject each key on keystore
// It return empty string if not secret provided
func GetSecretNameForKeystore(elasticsearch *elasticsearchcrd.Elasticsearch) (secretName string) {

	if elasticsearch.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
		return elasticsearch.Spec.GlobalNodeGroup.KeystoreSecretRef.Name
	}

	return ""
}

// GetGlobalServiceName permit to get the global service name
func GetGlobalServiceName(elasticsearch *elasticsearchcrd.Elasticsearch) (serviceName string) {
	return fmt.Sprintf("%s-es", elasticsearch.Name)
}

// GetLoadBalancerName permit to get the load balancer name
func GetLoadBalancerName(elasticsearch *elasticsearchcrd.Elasticsearch) (serviceName string) {
	return fmt.Sprintf("%s-lb-es", elasticsearch.Name)
}

// GetIngressName permit to get the ingress name
func GetIngressName(elasticsearch *elasticsearchcrd.Elasticsearch) (ingressName string) {
	return fmt.Sprintf("%s-es", elasticsearch.Name)
}

// GetNodeGroupPDBName permit to get the pdb name
func GetNodeGroupPDBName(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) (serviceName string) {
	return GetNodeGroupName(elasticsearch, nodeGroupName)
}

// GetContainerImage permit to get the image name
func GetContainerImage(elasticsearch *elasticsearchcrd.Elasticsearch) string {
	version := "latest"
	if elasticsearch.Spec.Version != "" {
		version = elasticsearch.Spec.Version
	}

	image := defaultImage
	if elasticsearch.Spec.Image != "" {
		image = elasticsearch.Spec.Image
	}

	return fmt.Sprintf("%s:%s", image, version)
}

func GetNodeGroupNameFromNodeName(nodeName string) (nodeGroupName string) {
	r := regexp.MustCompile(`^(.+)-\d+$`)
	res := r.FindStringSubmatch(nodeName)

	if len(res) > 1 {
		return res[1]
	}

	return ""
}

// isMasterRole return true if nodegroup have `cluster_manager` role
func IsMasterRole(elasticsearch *elasticsearchcrd.Elasticsearch, nodeGroupName string) bool {

	for _, nodeGroup := range elasticsearch.Spec.NodeGroups {
		if nodeGroup.Name == nodeGroupName {
			return funk.Contains(nodeGroup.Roles, "master")
		}
	}

	return false
}

// GetUserSystemName return the name for system users
func GetUserSystemName(es *elasticsearchcrd.Elasticsearch, username string) string {
	return fmt.Sprintf("%s-%s-es", es.Name, strings.ReplaceAll(username, "_", "-"))
}

// GetLicenseName return the name for the license
func GetLicenseName(es *elasticsearchcrd.Elasticsearch) string {
	return fmt.Sprintf("%s-es", es.Name)
}

// GetElasticsearchNameFromSecretApiTlsName return the Elasticsearch name from secret name that store TLS API
func GetElasticsearchNameFromSecretApiTlsName(secretApiTlsName string) (elasticsearchName string) {
	r := regexp.MustCompile(`^(.+)-tls-api-es`)
	res := r.FindStringSubmatch(secretApiTlsName)

	if len(res) > 1 {
		return res[1]
	}

	return ""
}

// GetNetworkPolicyName return the name for network policy
func GetNetworkPolicyName(es *elasticsearchcrd.Elasticsearch) string {
	return fmt.Sprintf("%s-allow-api-es", es.Name)
}

// GetExporterImage return the image to use for exporter pod
func GetExporterImage(es *elasticsearchcrd.Elasticsearch) string {
	version := "latest"
	if es.Spec.Monitoring.Prometheus != nil && es.Spec.Monitoring.Prometheus.Version != "" {
		version = es.Spec.Monitoring.Prometheus.Version

	}
	if es.Spec.Monitoring.Prometheus != nil && es.Spec.Monitoring.Prometheus.Image != "" {
		return fmt.Sprintf("%s:%s", es.Spec.Monitoring.Prometheus.Image, version)
	}

	return fmt.Sprintf("%s:%s", defaultExporterImage, version)
}

// GetExporterDeployementName return the exporter deployement name
func GetExporterDeployementName(es *elasticsearchcrd.Elasticsearch) string {
	return fmt.Sprintf("%s-exporter-es", es.Name)
}

// GetPodMonitorName return the name for podMonitor
func GetPodMonitorName(es *elasticsearchcrd.Elasticsearch) string {
	return fmt.Sprintf("%s-es", es.Name)
}

// GetPublicUrl permit to get the public URL to connect on Elasticsearch
func GetPublicUrl(es *elasticsearchcrd.Elasticsearch, targetNodeGroup string, external bool) string {

	scheme := "https"

	if !external {
		if !es.IsTlsApiEnabled() {
			scheme = "http"
		}

		if targetNodeGroup == "" {
			return fmt.Sprintf("%s://%s.%s.svc:9200", scheme, GetGlobalServiceName(es), es.Namespace)
		} else {
			return fmt.Sprintf("%s://%s.%s.svc:9200", scheme, GetNodeGroupServiceName(es, targetNodeGroup), es.Namespace)
		}
	}

	return es.Status.Url

}

// GetMetricbeatName return the metricbeat namme
func GetMetricbeatName(es *elasticsearchcrd.Elasticsearch) (name string) {
	return fmt.Sprintf("%s-metricbeat-es", es.Name)
}

// getLabels permit to return global label must be set on all resources
func getLabels(elasticsearch *elasticsearchcrd.Elasticsearch, customLabels ...map[string]string) (labels map[string]string) {
	labels = map[string]string{
		"cluster":                  elasticsearch.Name,
		ElasticsearchAnnotationKey: "true",
	}
	for _, label := range customLabels {
		for key, val := range label {
			labels[key] = val
		}
	}

	labels = funk.UnionStringMap(labels, elasticsearch.Labels)

	return labels
}

// getAnnotations permit to return global annotations must be set on all resources
func getAnnotations(elasticsearch *elasticsearchcrd.Elasticsearch, customAnnotation ...map[string]string) (annotations map[string]string) {
	annotations = map[string]string{
		ElasticsearchAnnotationKey: "true",
	}
	for _, annotation := range customAnnotation {
		for key, val := range annotation {
			annotations[key] = val
		}
	}

	return annotations
}
