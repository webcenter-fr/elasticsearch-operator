package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestIsProtected(t *testing.T) {
	var o *User

	// With default value
	o = &User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			Username: "test",
		},
	}

	assert.False(t, o.IsProtected())

	// When set to false
	o = &User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			Username:    "test",
			IsProtected: ptr.To[bool](false),
		},
	}

	assert.False(t, o.IsProtected())

	// When set to true
	o = &User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			Username:    "test",
			IsProtected: ptr.To[bool](true),
		},
	}

	assert.True(t, o.IsProtected())

}

func TestGetUsername(t *testing.T) {
	var o *User

	// When Username is set
	o = &User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			Username:    "test2",
			IsProtected: ptr.To[bool](false),
		},
	}

	assert.Equal(t, "test2", o.GetUsername())

	// When Username isn't set
	o = &User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			IsProtected: ptr.To[bool](false),
		},
	}

	assert.Equal(t, "test", o.GetUsername())
}
