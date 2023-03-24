package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetConfigMapName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-config-cb", GetConfigMapName(o))
}

func TestGetServiceName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-cb", GetServiceName(o))
}

func TestGetLoadBalancerName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-lb-cb", GetLoadBalancerName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-cb", GetIngressName(o))
}

func TestGetDeploymentName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-cb", GetDeploymentName(o))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}
	assert.Equal(t, "lmenezes/cerebro:latest", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v1"
	assert.Equal(t, "lmenezes/cerebro:v1", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v1", GetContainerImage(o))
}

func TestGetLabels(t *testing.T) {
	var expectedLabels map[string]string

	// With default values
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	expectedLabels = map[string]string{
		"cluster":                  "test",
		"cerebro.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedLabels, getLabels(o))

	// With additional labels
	expectedLabels["foo"] = "bar"

	assert.Equal(t, expectedLabels, getLabels(o, map[string]string{"foo": "bar"}))
}

func TestGetAnnotations(t *testing.T) {
	var expectedAnnotations map[string]string

	// With default values
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	expectedAnnotations = map[string]string{
		"cerebro.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedAnnotations, getAnnotations(o))

	// With additional annottaions
	expectedAnnotations["foo"] = "bar"

	assert.Equal(t, expectedAnnotations, getAnnotations(o, map[string]string{"foo": "bar"}))
}
