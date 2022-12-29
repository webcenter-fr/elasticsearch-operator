package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestBuildConfigMap(t *testing.T) {

	var o *kibanaapi.Kibana

	// Normal
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
			Labels: map[string]string{
				"label1": "value1",
			},
			Annotations: map[string]string{
				"anno1": "value1",
			},
		},
		Spec: kibanaapi.KibanaSpec{

			Config: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
		},
	}

	configMap, err := BuildConfigMap(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_default.yml", configMap, test.CleanApi)

	// When TLS is disabled
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
			Labels: map[string]string{
				"label1": "value1",
			},
			Annotations: map[string]string{
				"anno1": "value1",
			},
		},
		Spec: kibanaapi.KibanaSpec{
			Tls: kibanaapi.TlsSpec{
				Enabled: pointer.Bool(false),
			},
			Config: map[string]string{
				"kibana.yml": `node.value: test
node.value2: test`,
				"log4j.yml": "log.test: test\n",
			},
		},
	}

	configMap, err = BuildConfigMap(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/configmap_tls_disabled.yml", configMap, test.CleanApi)
}
