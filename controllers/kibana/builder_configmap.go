package kibana

import (
	"fmt"

	"github.com/pkg/errors"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config map
func BuildConfigMap(kb *kibanaapi.Kibana) (configMap *corev1.ConfigMap, err error) {
	var (
		expectedConfig map[string]string
	)

	injectedConfigMap := map[string]string{}

	if kb.IsTlsEnabled() {
		injectedConfigMap["kibana.yml"] = `
server.ssl.enabled: true
server.ssl.certificate: /usr/share/kibana/config/api-cert/tls.crt
server.ssl.key: /usr/share/kibana/config/api-cert/tls.key
`
	} else {
		injectedConfigMap["kibana.yml"] = "server.ssl.enabled: false\n"
	}

	if kb.IsIngressEnabled() {
		scheme := "https"
		if !kb.IsTlsEnabled() && kb.Spec.Endpoint.Ingress.SecretRef == nil {
			scheme = "http"
		}
		injectedConfigMap["kibana.yml"] += fmt.Sprintf("server.publicBaseUrl: %s://%s\n", scheme, kb.Spec.Endpoint.Ingress.Host)
	}

	// Inject computed config
	expectedConfig, err = helper.MergeSettings(injectedConfigMap, kb.Spec.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Error when merge expected config with computed config")
	}

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetConfigMapName(kb),
			Labels:      getLabels(kb),
			Annotations: getAnnotations(kb),
		},
		Data: expectedConfig,
	}

	return configMap, nil
}
