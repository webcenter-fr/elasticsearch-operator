package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
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

func TestFindDiscoverRef(t *testing.T) {
	refs := []*DiscoverRef{
		{
			Kafka: &corev1.LocalObjectReference{Name: "kafka-cluster"},
		},
		nil,
		{
			Logstash: &corev1.LocalObjectReference{Name: "logstash-cluster"},
		},
	}

	// When discovered by name
	assert.Equal(t, "kafka-cluster", FindDiscoverRef(refs, ptr.To("kafka-cluster")).GetName())
	assert.Equal(t, "logstash-cluster", FindDiscoverRef(refs, ptr.To("logstash-cluster")).GetName())

	// When not found, nil or empty
	assert.Nil(t, FindDiscoverRef(refs, ptr.To("unknown")))
	assert.Nil(t, FindDiscoverRef(refs, nil))
	assert.Nil(t, FindDiscoverRef(refs, ptr.To("")))
}

func TestDiscoverValidateName(t *testing.T) {
	var discover Discover

	// When name is not set
	assert.Nil(t, discover.ValidateName())

	// When name is valid
	for _, name := range []string{"kafka-sample", "KAFKA_SAMPLE", "MYAPP_PROD", "app.logstash", "a1"} {
		discover = Discover{Name: ptr.To(name)}
		assert.Nil(t, discover.ValidateName())
	}

	// When name is invalid
	for _, name := range []string{"../../etc", "a/b", "..", "a..b", "a b", "a$b", "-start", ".start", "a\nb"} {
		discover = Discover{Name: ptr.To(name)}
		assert.NotNil(t, discover.ValidateName(), "name %q must be rejected", name)
	}
}

func TestDiscoverRefValidateField(t *testing.T) {
	fldPath := field.NewPath("spec").Child("discoverRef").Index(0)

	// When valid
	ref := DiscoverRef{
		Kafka: &corev1.LocalObjectReference{Name: "kafka-cluster"},
	}
	assert.Nil(t, ref.ValidateField(fldPath))

	// When no discover type is set
	assert.NotNil(t, DiscoverRef{}.ValidateField(fldPath))
	assert.NotNil(t, DiscoverRef{Kafka: &corev1.LocalObjectReference{Name: ""}}.ValidateField(fldPath))

	// When many discover types are set
	ref = DiscoverRef{
		Kafka:    &corev1.LocalObjectReference{Name: "kafka-cluster"},
		Logstash: &corev1.LocalObjectReference{Name: "logstash-cluster"},
	}
	assert.NotNil(t, ref.ValidateField(fldPath))
}

func TestValidateDiscoverRefs(t *testing.T) {
	fldPath := field.NewPath("spec").Child("discoverRef")

	// When all refs are valid
	errs := ValidateDiscoverRefs([]*DiscoverRef{
		{Kafka: &corev1.LocalObjectReference{Name: "kafka-cluster"}},
		{Logstash: &corev1.LocalObjectReference{Name: "logstash-cluster"}},
	}, fldPath)
	assert.Len(t, errs, 0)

	// When refs contain a nil entry and an invalid entry
	errs = ValidateDiscoverRefs([]*DiscoverRef{
		nil,
		{Kafka: &corev1.LocalObjectReference{Name: "kafka-cluster"}, Elasticsearch: &corev1.LocalObjectReference{Name: "es-cluster"}},
	}, fldPath)
	assert.Len(t, errs, 2)
	assert.Equal(t, fldPath.Index(0).String(), errs[0].Field)
	assert.Equal(t, fldPath.Index(1).String(), errs[1].Field)
}
