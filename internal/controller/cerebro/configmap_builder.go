package cerebro

import (
	"dario.cat/mergo"
	"emperror.dev/errors"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cerebroConfig is the generated application.yaml config
type cerebroConfig struct {
	Secret   string             `yaml:"secret"`
	BasePath string             `yaml:"basePath"`
	Data     cerebroDataConfig  `yaml:"data"`
	Rest     cerebroRestConfig  `yaml:"rest"`
	Es       cerebroEsConfig    `yaml:"es"`
	Auth     cerebroAuthConfig  `yaml:"auth"`
	Hosts    []cerebroHostEntry `yaml:"hosts"`
}

type cerebroDataConfig struct {
	Path string `yaml:"path"`
}

type cerebroRestConfig struct {
	History cerebroRestHistoryConfig `yaml:"history"`
}

type cerebroRestHistoryConfig struct {
	Size int `yaml:"size"`
}

type cerebroEsConfig struct {
	Gzip bool `yaml:"gzip"`
}

type cerebroAuthConfig struct {
	Type     string                    `yaml:"type"`
	Settings cerebroAuthSettingsConfig `yaml:"settings"`
}

type cerebroAuthSettingsConfig struct {
	Username     string                   `yaml:"username"`
	Password     string                   `yaml:"password"`
	URL          string                   `yaml:"url"`
	BaseDN       string                   `yaml:"base-dn"`
	Method       string                   `yaml:"method"`
	UserTemplate string                   `yaml:"user-template"`
	BindDN       string                   `yaml:"bind-dn"`
	BindPW       string                   `yaml:"bind-pw"`
	GroupSearch  cerebroGroupSearchConfig `yaml:"group-search"`
}

type cerebroGroupSearchConfig struct {
	BaseDN           string `yaml:"base-dn"`
	UserAttr         string `yaml:"user-attr"`
	UserAttrTemplate string `yaml:"user-attr-template"`
	Group            string `yaml:"group"`
}

// cerebroHostEntry is one entry of the `hosts` list.
// `type` is intentionally omitted: upstream auto-detects elasticsearch vs opensearch
// via `GET /` when the field is empty.
type cerebroHostEntry struct {
	Host string `yaml:"host"`
	Name string `yaml:"name"`
}

// buildCerebroConfig computes the base config from the Cerebro spec and its enrolled hosts.
func buildCerebroConfig(esList []elasticsearchcrd.Elasticsearch, externalList []cerebrocrd.ElasticsearchExternalRef) *cerebroConfig {
	config := &cerebroConfig{
		Secret:   "",
		BasePath: "/",
		Data: cerebroDataConfig{
			Path: "/data/cerebro.db",
		},
		Rest: cerebroRestConfig{
			History: cerebroRestHistoryConfig{
				Size: 50,
			},
		},
		Es: cerebroEsConfig{
			Gzip: true,
		},
		Auth: cerebroAuthConfig{
			Type: "",
			Settings: cerebroAuthSettingsConfig{
				Method:       "simple",
				UserTemplate: "uid=%s,%s",
			},
		},
		Hosts: make([]cerebroHostEntry, 0, len(esList)+len(externalList)),
	}

	// Compute managed hosts
	for _, es := range esList {
		name := es.Name
		if es.Spec.ClusterName != "" {
			name = es.Spec.ClusterName
		}
		config.Hosts = append(config.Hosts, cerebroHostEntry{
			Name: name,
			Host: elasticsearchcontrollers.GetPublicUrl(&es, "", false),
		})
	}

	// Compute not managed hosts
	config.Hosts = appendExternalRefs(config.Hosts, externalList)

	return config
}

// appendExternalRefs appends cerebroHostEntry items for each external reference.
func appendExternalRefs(hosts []cerebroHostEntry, refs []cerebrocrd.ElasticsearchExternalRef) []cerebroHostEntry {
	for _, ref := range refs {
		hosts = append(hosts, cerebroHostEntry{
			Name: ref.Name,
			Host: ref.Address,
		})
	}
	return hosts
}

// renderCerebroConfig marshals the base config, then deep merges the user supplied
// YAML overlays on top of it. Overlays win on conflict. Later overlays win over earlier ones.
func renderCerebroConfig(base *cerebroConfig, overlays ...string) (string, error) {
	raw, err := yaml.Marshal(base)
	if err != nil {
		return "", errors.Wrap(err, "Error when marshal cerebro config")
	}

	merged := map[string]any{}
	if err = yaml.Unmarshal(raw, &merged); err != nil {
		return "", errors.Wrap(err, "Error when normalize cerebro config")
	}

	for _, overlay := range overlays {
		if overlay == "" {
			continue
		}
		src := map[string]any{}
		if err = yaml.Unmarshal([]byte(overlay), &src); err != nil {
			return "", errors.Wrap(err, "Error when parse provided cerebro config, it must be a valid YAML document")
		}
		if err = mergo.Merge(&merged, src, mergo.WithOverride); err != nil {
			return "", errors.Wrap(err, "Error when merge provided config with default config")
		}
	}

	out, err := yaml.Marshal(merged)
	if err != nil {
		return "", errors.Wrap(err, "Error when marshal merged cerebro config")
	}

	return string(out), nil
}

// BuildConfigMap permit to generate config map
func buildConfigMaps(cb *cerebrocrd.Cerebro, esList []elasticsearchcrd.Elasticsearch, externalList []cerebrocrd.ElasticsearchExternalRef) (configMaps []*corev1.ConfigMap, err error) {
	configMaps = make([]*corev1.ConfigMap, 0, 1)

	overlays := make([]string, 0, 2)
	if cb.Spec.Config != nil && *cb.Spec.Config != "" {
		overlays = append(overlays, *cb.Spec.Config)
	}
	if cb.Spec.ExtraConfigs["application.yaml"] != "" {
		overlays = append(overlays, cb.Spec.ExtraConfigs["application.yaml"])
	}

	config, err := renderCerebroConfig(buildCerebroConfig(esList, externalList), overlays...)
	if err != nil {
		return nil, err
	}

	expectedConfig := map[string]string{
		"application.yaml": config,
	}

	// Extra config files are added as standalone keys. mergo does not override the
	// already computed `application.yaml`, which was consumed as an overlay above.
	if err = mergo.Merge(&expectedConfig, cb.Spec.ExtraConfigs); err != nil {
		return nil, errors.Wrap(err, "Error when merge provided config with default config")
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cb.Namespace,
			Name:        GetConfigMapName(cb),
			Labels:      getLabels(cb),
			Annotations: getAnnotations(cb),
		},
		Data: expectedConfig,
	}

	configMaps = append(configMaps, configMap)

	return configMaps, nil
}
