package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/multiphase"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestKafkaGetStatus(t *testing.T) {
	status := KafkaStatus{
		DefaultMultiPhaseObjectStatus: multiphase.DefaultMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestKafkaGetDiscoverStatus(t *testing.T) {
	status := KafkaStatus{
		DiscoverStatus: DiscoverStatus{
			SecretEnvRef:  ptr.To("test-env"),
			SecretFileRef: ptr.To("test-file"),
		},
	}
	o := &Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetDiscoverStatus())
}

func TestKafkaGetInternalName(t *testing.T) {
	// When custom name
	o := &Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: KafkaSpec{
			Discover: Discover{
				Name: ptr.To("custom-name"),
			},
		},
	}
	assert.Equal(t, "custom-name", o.GetInternalName())

	// When name is nil
	o.Spec.Name = nil
	assert.Equal(t, "test", o.GetInternalName())

	// When name is empty
	o.Spec.Name = ptr.To("")
	assert.Equal(t, "test", o.GetInternalName())
}

func TestKafkaRefIsManaged(t *testing.T) {
	var kafkaRef KafkaRef

	// When managed Kafka is set
	kafkaRef = KafkaRef{
		ManagedKafkaRef: &KafkaManagedRef{
			Name: "test-kafka",
			UserRef: &corev1.LocalObjectReference{
				Name: "test-user",
			},
		},
	}
	assert.True(t, kafkaRef.IsManaged())

	// When managed Kafka name is empty
	kafkaRef = KafkaRef{
		ManagedKafkaRef: &KafkaManagedRef{
			Name: "",
		},
	}
	assert.False(t, kafkaRef.IsManaged())

	// When managed Kafka is nil
	kafkaRef = KafkaRef{
		ManagedKafkaRef: nil,
	}
	assert.False(t, kafkaRef.IsManaged())

	// When empty
	kafkaRef = KafkaRef{}
	assert.False(t, kafkaRef.IsManaged())
}

func TestKafkaRefIsExternal(t *testing.T) {
	var kafkaRef KafkaRef

	// When external Kafka is set
	kafkaRef = KafkaRef{
		ExternalKafkaRef: &KafkaExternalRef{
			Addresses: []string{
				"kafka1.example.com:9092",
				"kafka2.example.com:9092",
			},
		},
	}
	assert.True(t, kafkaRef.IsExternal())

	// When external Kafka addresses are empty
	kafkaRef = KafkaRef{
		ExternalKafkaRef: &KafkaExternalRef{
			Addresses: []string{},
		},
	}
	assert.False(t, kafkaRef.IsExternal())

	// When external Kafka is nil
	kafkaRef = KafkaRef{
		ExternalKafkaRef: nil,
	}
	assert.False(t, kafkaRef.IsExternal())

	// When empty
	kafkaRef = KafkaRef{}
	assert.False(t, kafkaRef.IsExternal())
}

func TestKafkaRefValidateField(t *testing.T) {
	var kafkaRef KafkaRef
	var err *field.Error

	// Valid managed Kafka
	kafkaRef = KafkaRef{
		ManagedKafkaRef: &KafkaManagedRef{
			Name: "test-kafka",
		},
	}
	err = kafkaRef.ValidateField()
	assert.Nil(t, err)

	// Valid external Kafka
	kafkaRef = KafkaRef{
		ExternalKafkaRef: &KafkaExternalRef{
			Addresses: []string{"kafka.example.com:9092"},
		},
	}
	err = kafkaRef.ValidateField()
	assert.Nil(t, err)

	// Invalid when neither managed nor external
	kafkaRef = KafkaRef{}
	err = kafkaRef.ValidateField()
	assert.Error(t, err)

	// Invalid when both managed and external
	kafkaRef = KafkaRef{
		ManagedKafkaRef: &KafkaManagedRef{
			Name: "test-kafka",
		},
		ExternalKafkaRef: &KafkaExternalRef{
			Addresses: []string{"kafka.example.com:9092"},
		},
	}
	err = kafkaRef.ValidateField()
	assert.Error(t, err)
}
