package cerebro

import (
	"fmt"
	"strings"

	"dario.cat/mergo"
	"emperror.dev/errors"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildConfigMap permit to generate config map
func BuildConfigMap(cb *cerebrocrd.Cerebro, esList []elasticsearchcrd.Elasticsearch) (configMap *corev1.ConfigMap, err error) {
	var (
		expectedConfig map[string]string
		name           string
	)

	var config strings.Builder

	config.WriteString(`play.ws.ssl.loose.acceptAnyCertificate = true
basePath = "/"
pidfile.path = /dev/null
rest.history.size = 50
data.path = "/var/db/cerebro/cerebro.db"
es = {
  gzip = true
}
secret = ${?APPLICATION_SECRET}
auth = {
  # either basic or ldap
  type: ${?AUTH_TYPE}
  settings {
    # LDAP
    url = ${?LDAP_URL}
    base-dn = ${?LDAP_BASE_DN}
    method = ${?LDAP_METHOD}
    user-template = ${?LDAP_USER_TEMPLATE}
    bind-dn = ${?LDAP_BIND_DN}
    bind-pw = ${?LDAP_BIND_PWD}
    group-search {
      base-dn = ${?LDAP_GROUP_BASE_DN}
      user-attr = ${?LDAP_USER_ATTR}
      user-attr-template = ${?LDAP_USER_ATTR_TEMPLATE}
      group = ${?LDAP_GROUP}
    }
    # Basic auth
    username = ${?BASIC_AUTH_USER}
    password = ${?BASIC_AUTH_PWD}
  }
}
`)

	if len(esList) > 0 {
		config.WriteString("hosts = [\n")
		hosts := make([]string, 0, len(esList))

		for _, es := range esList {
			name = es.Name
			if es.Spec.ClusterName != "" {
				name = es.Spec.ClusterName
			}
			hosts = append(hosts, fmt.Sprintf("  {\n    name = \"%s\"\n    host = \"%s\"\n  }", name, elasticsearchcontrollers.GetPublicUrl(&es, "", false)))
		}

		config.WriteString(strings.Join(hosts, ",\n"))
		config.WriteString("\n]\n")
	}

	if cb.Spec.Config != nil && cb.Spec.Config["application.conf"] != "" {
		config.WriteString(cb.Spec.Config["application.conf"])
	}

	expectedConfig = map[string]string{
		"application.conf": config.String(),
	}

	if err = mergo.Merge(&expectedConfig, cb.Spec.Config); err != nil {
		return nil, errors.Wrap(err, "Error when merge provided config with default config")
	}

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cb.Namespace,
			Name:        GetConfigMapName(cb),
			Labels:      getLabels(cb),
			Annotations: getAnnotations(cb),
		},
		Data: expectedConfig,
	}

	return configMap, nil
}
