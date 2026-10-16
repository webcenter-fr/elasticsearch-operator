package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/multiphase"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestLogstashGetStatus(t *testing.T) {
	status := LogstashStatus{
		DefaultMultiPhaseObjectStatus: multiphase.DefaultMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestLogstashGetDiscoverStatus(t *testing.T) {
	status := LogstashStatus{
		DiscoverStatus: DiscoverStatus{
			SecretEnvRef:  ptr.To("test-env"),
			SecretFileRef: ptr.To("test-file"),
		},
	}
	o := &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetDiscoverStatus())
}

func TestLogstashGetInternalName(t *testing.T) {
	// When custom name
	o := &Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LogstashSpec{
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

func TestLogstashRefIsManaged(t *testing.T) {
	var logstashRef LogstashRef

	// When managed Logstash is set
	logstashRef = LogstashRef{
		ManagedLogstashRef: &LogstashManagedRef{
			Name: "test-logstash",
		},
	}
	assert.True(t, logstashRef.IsManaged())

	// When managed Logstash name is empty
	logstashRef = LogstashRef{
		ManagedLogstashRef: &LogstashManagedRef{
			Name: "",
		},
	}
	assert.False(t, logstashRef.IsManaged())

	// When managed Logstash is nil
	logstashRef = LogstashRef{
		ManagedLogstashRef: nil,
	}
	assert.False(t, logstashRef.IsManaged())

	// When empty
	logstashRef = LogstashRef{}
	assert.False(t, logstashRef.IsManaged())
}

func TestLogstashRefIsExternal(t *testing.T) {
	var logstashRef LogstashRef

	// When external Logstash is set
	logstashRef = LogstashRef{
		ExternalLogstashRef: &LogstashExternalRef{
			Addresses: []string{
				"logstash1.example.com:5000",
				"logstash2.example.com:5000",
			},
		},
	}
	assert.True(t, logstashRef.IsExternal())

	// When external Logstash addresses are empty
	logstashRef = LogstashRef{
		ExternalLogstashRef: &LogstashExternalRef{
			Addresses: []string{},
		},
	}
	assert.False(t, logstashRef.IsExternal())

	// When external Logstash is nil
	logstashRef = LogstashRef{
		ExternalLogstashRef: nil,
	}
	assert.False(t, logstashRef.IsExternal())

	// When empty
	logstashRef = LogstashRef{}
	assert.False(t, logstashRef.IsExternal())
}

func TestLogstashRefValidateField(t *testing.T) {
	var logstashRef LogstashRef
	var err *field.Error

	// Valid managed Logstash
	logstashRef = LogstashRef{
		ManagedLogstashRef: &LogstashManagedRef{
			Name: "test-logstash",
		},
	}
	err = logstashRef.ValidateField()
	assert.Nil(t, err)

	// Valid external Logstash
	logstashRef = LogstashRef{
		ExternalLogstashRef: &LogstashExternalRef{
			Addresses: []string{"logstash.example.com:5000"},
		},
	}
	err = logstashRef.ValidateField()
	assert.Nil(t, err)

	// Invalid when neither managed nor external
	logstashRef = LogstashRef{}
	err = logstashRef.ValidateField()
	assert.Error(t, err)

	// Invalid when both managed and external
	logstashRef = LogstashRef{
		ManagedLogstashRef: &LogstashManagedRef{
			Name: "test-logstash",
		},
		ExternalLogstashRef: &LogstashExternalRef{
			Addresses: []string{"logstash.example.com:5000"},
		},
	}
	err = logstashRef.ValidateField()
	assert.Error(t, err)
}
