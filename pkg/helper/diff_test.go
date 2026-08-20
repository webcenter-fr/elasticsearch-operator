package helper

import (
	"testing"

	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDiffDiffLabels(t *testing.T) {
	var (
		m         map[string]string
		expectedM map[string]string
	)

	// When the same without exclude key

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	assert.Empty(t, DiffLabels(expectedM, m))

	// When differ

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val4",
	}

	assert.NotEmpty(t, DiffLabels(expectedM, m))
}

func TestDiffOwnerReference(t *testing.T) {
	var (
		err           error
		o             *elasticsearchcrd.Elasticsearch
		s             *corev1.Secret
		diff          string
		controllerRef *metav1.OwnerReference
		trueVal       = true
	)

	// When missing owner reference
	o = &elasticsearchcrd.Elasticsearch{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "elasticsearch.k8s.webcenter.fr",
		Version: "v1",
		Kind:    "Elasticsearch",
	})
	s = &corev1.Secret{}
	diff, err = DiffOwnerReferences(o, s)
	assert.Nil(t, err)
	assert.Equal(t, "Need to set owner references", diff)

	// When owner reference group and kind differ
	o = &elasticsearchcrd.Elasticsearch{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "elasticsearch.k8s.webcenter.fr",
		Version: "v1",
		Kind:    "Elasticsearch",
	})
	s = &corev1.Secret{}
	controllerRef = &metav1.OwnerReference{
		APIVersion: "test/v2",
		Kind:       "Test",
		Controller: &trueVal,
	}
	s.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
	diff, err = DiffOwnerReferences(o, s)
	assert.Nil(t, err)
	assert.Equal(t, "Owner references differ: Test.test/v2 vs Elasticsearch.elasticsearch.k8s.webcenter.fr/v1", diff)

	// When owner reference version differ
	o = &elasticsearchcrd.Elasticsearch{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "elasticsearch.k8s.webcenter.fr",
		Version: "v1",
		Kind:    "Elasticsearch",
	})
	s = &corev1.Secret{}
	controllerRef = &metav1.OwnerReference{
		APIVersion: "elasticsearch.k8s.webcenter.fr/v2",
		Kind:       "Elasticsearch",
		Controller: &trueVal,
	}
	s.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
	diff, err = DiffOwnerReferences(o, s)
	assert.Nil(t, err)
	assert.Equal(t, "Owner references differ: elasticsearch.k8s.webcenter.fr/v2 vs elasticsearch.k8s.webcenter.fr/v1", diff)

	// When owner reference is up to date
	controllerRef = &metav1.OwnerReference{
		APIVersion: "elasticsearch.k8s.webcenter.fr/v1",
		Kind:       "Elasticsearch",
		Controller: &trueVal,
	}
	s.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
	diff, err = DiffOwnerReferences(o, s)
	assert.Nil(t, err)
	assert.Empty(t, diff)
}

func TestDiffAnnotations(t *testing.T) {
	var (
		m         map[string]string
		expectedM map[string]string
	)

	// When the same without exclude key

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	assert.Empty(t, DiffAnnotations(expectedM, m))

	// When differ

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val4",
	}

	assert.NotEmpty(t, DiffAnnotations(expectedM, m))

	// When differ but exclude key

	m = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"kubectl.kubernetes.io/last-applied-configuration": "dfdf",
	}

	expectedM = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"kubectl.kubernetes.io/last-applied-configuration": "aaa",
	}

	assert.Empty(t, DiffAnnotations(expectedM, m))
}
