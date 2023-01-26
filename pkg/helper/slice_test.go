package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStringToSlice(t *testing.T) {

	assert.Equal(t, []string{"test"}, StringToSlice("test", ","))
	assert.Equal(t, []string{}, StringToSlice("", ","))
	assert.Equal(t, []string{"test", "test2"}, StringToSlice("test,test2", ","))
	assert.Equal(t, []string{"test", "test2"}, StringToSlice("test, test2", ","))

}

func TestDeleteItemFromSlice(t *testing.T) {
	var (
		s        []string
		expected []string
	)

	// Normal case
	s = []string{
		"one",
		"two",
		"three",
		"four",
	}

	expected = []string{
		"one",
		"two",
		"four",
	}

	assert.Equal(t, expected, DeleteItemFromSlice(s, 2))

	// When index is out of slice
	s = []string{
		"one",
		"two",
		"three",
		"four",
	}

	expected = []string{
		"one",
		"two",
		"three",
		"four",
	}

	assert.Equal(t, expected, DeleteItemFromSlice(s, 10))

	// When slcie is nil
	assert.Equal(t, nil, DeleteItemFromSlice(nil, 10))

}

func TestToSliceOfObject(t *testing.T) {
	pods := []corev1.Pod{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		},
	}

	expected := []client.Object{
		&pods[0],
	}

	assert.Equal(t, expected, ToSliceOfObject(pods))
}
