package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestDiscoverStatusGetSecretFileRef(t *testing.T) {
	var status *DiscoverStatus

	// When default
	status = &DiscoverStatus{}
	assert.Nil(t, status.GetSecretFileRef())

	// When secret file ref is set
	secretName := "test-secret-file"
	status = &DiscoverStatus{
		SecretFileRef: &secretName,
	}
	assert.NotNil(t, status.GetSecretFileRef())
	assert.Equal(t, &secretName, status.GetSecretFileRef())
}

func TestDiscoverStatusGetSecretEnvRef(t *testing.T) {
	var status *DiscoverStatus

	// When default
	status = &DiscoverStatus{}
	assert.Nil(t, status.GetSecretEnvRef())

	// When secret env ref is set
	secretName := "test-secret-env"
	status = &DiscoverStatus{
		SecretEnvRef: &secretName,
	}
	assert.NotNil(t, status.GetSecretEnvRef())
	assert.Equal(t, &secretName, status.GetSecretEnvRef())
}

func TestDiscoverRefGetType(t *testing.T) {
	var ref *DiscoverRef

	// When Kafka ref
	ref = &DiscoverRef{
		Kafka: &corev1.LocalObjectReference{
			Name: "kafka-cluster",
		},
	}
	assert.Equal(t, DiscoverTypeKafka, ref.GetType())

	// When Logstash ref
	ref = &DiscoverRef{
		Logstash: &corev1.LocalObjectReference{
			Name: "logstash-cluster",
		},
	}
	assert.Equal(t, DiscoverTypeLogstash, ref.GetType())

	// When Elasticsearch ref
	ref = &DiscoverRef{
		Elasticsearch: &corev1.LocalObjectReference{
			Name: "elasticsearch-cluster",
		},
	}
	assert.Equal(t, DiscoverTypeElasticsearch, ref.GetType())

	// When empty ref
	ref = &DiscoverRef{}
	assert.Equal(t, DiscoverType(""), ref.GetType())

	// When ref name is empty
	ref = &DiscoverRef{
		Kafka: &corev1.LocalObjectReference{
			Name: "",
		},
	}
	assert.Equal(t, DiscoverType(""), ref.GetType())
}

func TestDiscoverRefGetName(t *testing.T) {
	var ref *DiscoverRef

	// When Kafka ref
	ref = &DiscoverRef{
		Kafka: &corev1.LocalObjectReference{
			Name: "kafka-cluster",
		},
	}
	assert.Equal(t, "kafka-cluster", ref.GetName())

	// When Logstash ref
	ref = &DiscoverRef{
		Logstash: &corev1.LocalObjectReference{
			Name: "logstash-cluster",
		},
	}
	assert.Equal(t, "logstash-cluster", ref.GetName())

	// When Elasticsearch ref
	ref = &DiscoverRef{
		Elasticsearch: &corev1.LocalObjectReference{
			Name: "elasticsearch-cluster",
		},
	}
	assert.Equal(t, "elasticsearch-cluster", ref.GetName())

	// When empty ref
	ref = &DiscoverRef{}
	assert.Equal(t, "", ref.GetName())

	// When ref name is empty
	ref = &DiscoverRef{
		Logstash: &corev1.LocalObjectReference{
			Name: "",
		},
	}
	assert.Equal(t, "", ref.GetName())
}
