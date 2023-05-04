package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestIsBasicLicense(t *testing.T) {
	var o *License

	// With default parameters
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LicenseSpec{},
	}

	assert.True(t, o.IsBasicLicense())

	// When basic license is set to true without specify secret
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LicenseSpec{
			Basic: pointer.Bool(true),
		},
	}

	assert.True(t, o.IsBasicLicense())

	// When basic license is set to true with specify secret
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LicenseSpec{
			Basic: pointer.Bool(true),
			SecretRef: &v1.LocalObjectReference{
				Name: "test",
			},
		},
	}

	assert.True(t, o.IsBasicLicense())

	// When basic license is set to false without specify secret
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LicenseSpec{
			Basic: pointer.Bool(false),
		},
	}
	assert.False(t, o.IsBasicLicense())

	// When only set secret
	o = &License{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: LicenseSpec{
			SecretRef: &v1.LocalObjectReference{
				Name: "test",
			},
		},
	}
	assert.False(t, o.IsBasicLicense())

}
