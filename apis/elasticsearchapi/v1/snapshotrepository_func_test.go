package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSnapshotRepositoryExternalName(t *testing.T) {
	var o *SnapshotRepository

	// When name is set
	o = &SnapshotRepository{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: SnapshotRepositorySpec{
			Name: "test2",
		},
	}

	assert.Equal(t, "test2", o.GetExternalName())

	// When name isn't set
	o = &SnapshotRepository{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: SnapshotRepositorySpec{},
	}

	assert.Equal(t, "test", o.GetExternalName())
}
