package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/remote"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestUserGetStatus(t *testing.T) {
	status := UserStatus{
		DefaultRemoteObjectStatus: remote.DefaultRemoteObjectStatus{
			LastAppliedConfiguration: "test",
		},
	}
	o := &User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

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

func TestGetExternalName(t *testing.T) {
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

	assert.Equal(t, "test2", o.GetExternalName())

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

	assert.Equal(t, "test", o.GetExternalName())
}

func TestIsAutoGeneratePassword(t *testing.T) {
	// When not specified
	o := User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{},
	}
	assert.False(t, o.IsAutoGeneratePassword())

	// When disabled
	o = User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			AutoGeneratePassword: ptr.To[bool](false),
		},
	}
	assert.False(t, o.IsAutoGeneratePassword())

	// When enabled
	o = User{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpec{
			AutoGeneratePassword: ptr.To[bool](true),
		},
	}
	assert.True(t, o.IsAutoGeneratePassword())
}
