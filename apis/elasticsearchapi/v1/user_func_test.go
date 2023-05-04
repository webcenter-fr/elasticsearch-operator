package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
			IsProtected: pointer.Bool(false),
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
			IsProtected: pointer.Bool(true),
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
			IsProtected: pointer.Bool(false),
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
			IsProtected: pointer.Bool(false),
		},
	}

	assert.Equal(t, "test", o.GetUsername())
}
