package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretNameForCAElasticsearch(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-ca-es-ls", GetSecretNameForCAElasticsearch(o))
}

func TestGetSecretNameForKeystore(t *testing.T) {
	var o *logstashcrd.Logstash

	// When default value
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "", GetSecretNameForKeystore(o))

	// When keystore is provided
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			KeystoreSecretRef: &v1.LocalObjectReference{
				Name: "my-secret",
			},
		},
	}

	assert.Equal(t, "my-secret", GetSecretNameForKeystore(o))
}

func TestGetConfigMapConfigName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-config-ls", GetConfigMapConfigName(o))
}

func TestGetConfigMapPipelineName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-pipeline-ls", GetConfigMapPipelineName(o))
}

func TestGetConfigMapPatternName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-pattern-ls", GetConfigMapPatternName(o))
}

func TestGetServiceName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-test-ls", GetServiceName(o, "test"))
}

func TestGetGlobalServiceName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-headless-ls", GetGlobalServiceName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-test-ls", GetIngressName(o, "test"))
}

func TestGetPDBName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-ls", GetPDBName(o))
}

func TestGetStatefulsetName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-ls", GetStatefulsetName(o))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}
	assert.Equal(t, "docker.elastic.co/logstash/logstash:latest", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v1"
	assert.Equal(t, "docker.elastic.co/logstash/logstash:v1", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v1", GetContainerImage(o))
}

func TestGetLabels(t *testing.T) {
	var expectedLabels map[string]string

	// With default values
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	expectedLabels = map[string]string{
		"cluster":                   "test",
		"logstash.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedLabels, getLabels(o))

	// With additional labels
	expectedLabels["foo"] = "bar"

	assert.Equal(t, expectedLabels, getLabels(o, map[string]string{"foo": "bar"}))
}

func TestGetAnnotations(t *testing.T) {
	var expectedAnnotations map[string]string

	// With default values
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	expectedAnnotations = map[string]string{
		"logstash.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedAnnotations, getAnnotations(o))

	// With additional annottaions
	expectedAnnotations["foo"] = "bar"

	assert.Equal(t, expectedAnnotations, getAnnotations(o, map[string]string{"foo": "bar"}))
}

func TestGetSecretNameForCredentials(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-credential-ls", GetSecretNameForCredentials(o))
}

func TestGetNetworkPolicyName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-allow-ls", GetNetworkPolicyName(o))
}

func TestGetPodMonitorName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-ls", GetPodMonitorName(o))
}

func TestGetMetricbeatName(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-metricbeat-ls", GetMetricbeatName(o))
}

func TestGetSecretNameForPki(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	assert.Equal(t, "test-pki-ls", GetSecretNameForPki(o))
}

func TestGetSecretNameForTls(t *testing.T) {
	o := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	// With default value
	assert.Equal(t, "test-tls-ls", GetSecretNameForTls(o))
}
