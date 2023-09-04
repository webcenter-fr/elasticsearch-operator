package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetUserSpaceID(t *testing.T) {
	var o *UserSpace

	// When ID is set
	o = &UserSpace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpaceSpec{
			ID: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetUserSpaceID())

	// When ID isn't set
	o = &UserSpace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: UserSpaceSpec{},
	}

	assert.Equal(t, "test", o.GetUserSpaceID())
}

func TestIsIncludeReference(t *testing.T) {
	var o KibanaUserSpaceCopy

	// Default value
	o = KibanaUserSpaceCopy{}
	assert.True(t, o.IsIncludeReference())

	// When set to true
	o = KibanaUserSpaceCopy{
		IncludeReferences: ptr.To[bool](true),
	}
	assert.True(t, o.IsIncludeReference())

	// When set to false
	o = KibanaUserSpaceCopy{
		IncludeReferences: ptr.To[bool](false),
	}
	assert.False(t, o.IsIncludeReference())
}

func TestIsOverwrite(t *testing.T) {
	var o KibanaUserSpaceCopy

	// Default value
	o = KibanaUserSpaceCopy{}
	assert.True(t, o.IsOverwrite())

	// When set to true
	o = KibanaUserSpaceCopy{
		Overwrite: ptr.To[bool](true),
	}
	assert.True(t, o.IsOverwrite())

	// When set to false
	o = KibanaUserSpaceCopy{
		Overwrite: ptr.To[bool](false),
	}
	assert.False(t, o.IsOverwrite())
}

func TestIsCreateNewCopy(t *testing.T) {
	var o KibanaUserSpaceCopy

	// Default value
	o = KibanaUserSpaceCopy{}
	assert.False(t, o.IsCreateNewCopy())

	// When set to true
	o = KibanaUserSpaceCopy{
		CreateNewCopies: ptr.To[bool](true),
	}
	assert.True(t, o.IsCreateNewCopy())

	// When set to false
	o = KibanaUserSpaceCopy{
		CreateNewCopies: ptr.To[bool](false),
	}
	assert.False(t, o.IsCreateNewCopy())
}

func TestIsForceUpdate(t *testing.T) {
	var o KibanaUserSpaceCopy

	// Default value
	o = KibanaUserSpaceCopy{}
	assert.False(t, o.IsForceUpdate())

	// When set to true
	o = KibanaUserSpaceCopy{
		ForceUpdateWhenReconcile: ptr.To[bool](true),
	}
	assert.True(t, o.IsForceUpdate())

	// When set to false
	o = KibanaUserSpaceCopy{
		ForceUpdateWhenReconcile: ptr.To[bool](false),
	}
	assert.False(t, o.IsForceUpdate())
}
