package discover

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestGetSecretNameForFile(t *testing.T) {
	// When no target secret is specified
	assert.Equal(t, "test-file", GetSecretNameForFile("test", nil))
	assert.Equal(t, "test-file", GetSecretNameForFile("test", &corev1.LocalObjectReference{}))

	// When target secret is specified
	assert.Equal(t, "custom-file", GetSecretNameForFile("test", &corev1.LocalObjectReference{Name: "custom-file"}))
}

func TestGetSecretNameForEnv(t *testing.T) {
	// When no target secret is specified
	assert.Equal(t, "test-env", GetSecretNameForEnv("test", nil))
	assert.Equal(t, "test-env", GetSecretNameForEnv("test", &corev1.LocalObjectReference{}))

	// When target secret is specified
	assert.Equal(t, "custom-env", GetSecretNameForEnv("test", &corev1.LocalObjectReference{Name: "custom-env"}))
}

func TestGetLabels(t *testing.T) {
	// When no user labels
	labels := GetLabels(discovercrd.KafkaAnnotationKey, "test-kafka", nil)
	assert.Equal(t, map[string]string{
		"discoverName":                    "test-kafka",
		discovercrd.KafkaAnnotationKey:    "true",
		discovercrd.DiscoverAnnotationKey: "true",
	}, labels)

	// When user labels are set
	labels = GetLabels(discovercrd.KafkaAnnotationKey, "test-kafka", map[string]string{
		"app": "filebeat",
	})
	assert.Equal(t, map[string]string{
		"discoverName":                    "test-kafka",
		discovercrd.KafkaAnnotationKey:    "true",
		discovercrd.DiscoverAnnotationKey: "true",
		"app":                             "filebeat",
	}, labels)

	// Operator labels must not be overridable by user labels
	labels = GetLabels(discovercrd.KafkaAnnotationKey, "test-kafka", map[string]string{
		"discoverName":                    "other-kafka",
		discovercrd.KafkaAnnotationKey:    "false",
		discovercrd.DiscoverAnnotationKey: "false",
	})
	assert.Equal(t, map[string]string{
		"discoverName":                    "test-kafka",
		discovercrd.KafkaAnnotationKey:    "true",
		discovercrd.DiscoverAnnotationKey: "true",
	}, labels)
}

func TestGetAnnotations(t *testing.T) {
	// When no custom annotations
	annotations := GetAnnotations(discovercrd.KafkaAnnotationKey, "test-kafka")
	assert.Equal(t, map[string]string{
		discovercrd.KafkaAnnotationKey:                                 "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-kafka",
	}, annotations)

	// When custom annotations are set
	annotations = GetAnnotations(discovercrd.KafkaAnnotationKey, "test-kafka", map[string]string{
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "env",
	})
	assert.Equal(t, map[string]string{
		discovercrd.KafkaAnnotationKey:                                 "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-kafka",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "env",
	}, annotations)
}

func TestGetEnvSuffix(t *testing.T) {
	assert.Equal(t, "TEST_KAFKA", GetEnvSuffix("test-kafka"))
	assert.Equal(t, "MYAPP_PROD", GetEnvSuffix("MYAPP_PROD"))
	assert.Equal(t, "KAFKA_SAMPLE", GetEnvSuffix("kafka-sample"))
}
