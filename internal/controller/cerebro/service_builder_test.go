package cerebro

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildService(t *testing.T) {
	var (
		err      error
		services []corev1.Service
		o        *cerebrocrd.Cerebro
	)
	// With default values
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	services, err = buildServices(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*corev1.Service](t, "testdata/service_default.yaml", &services[0], scheme.Scheme)
}
