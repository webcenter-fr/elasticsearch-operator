package filebeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretNameForCAElasticsearch(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-ca-es-fb", GetSecretNameForCAElasticsearch(o))
}

func TestGetConfigMapConfigName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-config-fb", GetConfigMapConfigName(o))
}

func TestGetConfigMapModuleName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-module-fb", GetConfigMapModuleName(o))
}

func TestGetServiceName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-test-fb", GetServiceName(o, "test"))
}

func TestGetGlobalServiceName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-headless-fb", GetGlobalServiceName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-test-fb", GetIngressName(o, "test"))
}

func TestGetPDBName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-fb", GetPDBName(o))
}

func TestGetStatefulsetName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-fb", GetStatefulsetName(o))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}
	assert.Equal(t, "docker.elastic.co/beats/filebeat:latest", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v1"
	assert.Equal(t, "docker.elastic.co/beats/filebeat:v1", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v1", GetContainerImage(o))
}

func TestGetLabels(t *testing.T) {
	var expectedLabels map[string]string

	// With default values
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	expectedLabels = map[string]string{
		"cluster":                   "test",
		"filebeat.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedLabels, getLabels(o))

	// With additional labels
	expectedLabels["foo"] = "bar"

	assert.Equal(t, expectedLabels, getLabels(o, map[string]string{"foo": "bar"}))
}

func TestGetAnnotations(t *testing.T) {
	var expectedAnnotations map[string]string

	// With default values
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	expectedAnnotations = map[string]string{
		"filebeat.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedAnnotations, getAnnotations(o))

	// With additional annottaions
	expectedAnnotations["foo"] = "bar"

	assert.Equal(t, expectedAnnotations, getAnnotations(o, map[string]string{"foo": "bar"}))
}

func TestGetSecretNameForCredentials(t *testing.T) {

	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-credential-fb", GetSecretNameForCredentials(o))

}

func TestGetPodMonitorName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-fb", GetPodMonitorName(o))
}

func TestGetMetricbeatName(t *testing.T) {
	o := &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{},
	}

	assert.Equal(t, "test-metricbeat-fb", GetMetricbeatName(o))
}
