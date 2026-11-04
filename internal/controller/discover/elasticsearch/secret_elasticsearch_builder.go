package elasticsearch

import (
	"fmt"
	"strings"

	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildElasticsearchSecrets permit to build secret from Elasticsearch discovered
func buildElasticsearchSecrets(kk *discovercrd.Elasticsearch, elasticsearchCluster *elasticsearchcrd.Elasticsearch, secretCaElasticsearch *corev1.Secret, secretUserElasticsearch *corev1.Secret, secretCustomCaElasticsearch *corev1.Secret) (secrets []*corev1.Secret, err error) {
	secrets = make([]*corev1.Secret, 0, 2)
	envSuffix := getEnvSuffix(kk)

	secretForEnv := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSecretNameForEnv(kk),
			Namespace: kk.Namespace,
			Labels:    getLabels(kk),
			Annotations: getAnnotations(kk, map[string]string{
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "env",
			}),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	secretForFile := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSecretNameForFile(kk),
			Namespace: kk.Namespace,
			Labels:    getLabels(kk),
			Annotations: getAnnotations(kk, map[string]string{
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "file",
			}),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	// Compute CA certificates
	caCerts := make([]string, 0, 1)
	if secretCaElasticsearch != nil && secretCaElasticsearch.Data != nil && len(secretCaElasticsearch.Data["ca.crt"]) > 0 {
		caCerts = append(caCerts, string(secretCaElasticsearch.Data["ca.crt"]))
	}
	if secretCustomCaElasticsearch != nil && secretCustomCaElasticsearch.Data != nil {
		for certificateName, certificateData := range secretCustomCaElasticsearch.Data {
			if strings.HasSuffix(certificateName, ".crt") || strings.HasSuffix(certificateName, ".pem") {
				caCerts = append(caCerts, string(certificateData))
			}
		}
	}
	secretForFile.Data["ca.crt"] = []byte(strings.Join(caCerts, "\n"))

	// Compute Elasticsearch URL
	var elasticsearchHost string
	if elasticsearchCluster != nil {
		elasticsearchHost = elasticsearchcontrollers.GetPublicUrl(elasticsearchCluster, kk.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup, false)
	} else if kk.Spec.ElasticsearchRef.IsExternal() {
		elasticsearchHost = strings.Join(kk.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses, ",")
	}
	secretForEnv.Data[fmt.Sprintf("ELASTICSEARCH_HOSTS_%s", envSuffix)] = []byte(elasticsearchHost)

	// Compute Elasticsearch credentials
	if secretUserElasticsearch != nil && secretUserElasticsearch.Data != nil {
		if user, ok := secretUserElasticsearch.Data["username"]; ok {
			secretForEnv.Data[fmt.Sprintf("ELASTICSEARCH_USERNAME_%s", envSuffix)] = user
		}
		if password, ok := secretUserElasticsearch.Data["password"]; ok {
			secretForEnv.Data[fmt.Sprintf("ELASTICSEARCH_PASSWORD_%s", envSuffix)] = password
		}
	}

	secrets = append(secrets, secretForEnv, secretForFile)

	return secrets, nil
}
