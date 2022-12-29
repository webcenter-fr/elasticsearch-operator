package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildService(t *testing.T) {

	var (
		err      error
		service *corev1.Service
		o        *kibanaapi.Kibana
	)
	// With default values
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	service, err = BuildService(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/service_default.yaml", service, test.CleanApi)

}
