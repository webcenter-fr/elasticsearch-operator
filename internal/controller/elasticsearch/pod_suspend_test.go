package elasticsearch

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetSuspendedPodNames(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    map[string]bool
	}{
		{
			name:        "nil annotations",
			annotations: nil,
			expected:    map[string]bool{},
		},
		{
			name: "simple list",
			annotations: map[string]string{
				elasticsearchcrd.ElasticsearchSuspendAnnotation: "pod-0,pod-1",
			},
			expected: map[string]bool{"pod-0": true, "pod-1": true},
		},
		{
			name: "with spaces and trailing comma",
			annotations: map[string]string{
				elasticsearchcrd.ElasticsearchSuspendAnnotation: " pod-0 , pod-1 , ",
			},
			expected: map[string]bool{"pod-0": true, "pod-1": true},
		},
		{
			name: "empty annotation",
			annotations: map[string]string{
				elasticsearchcrd.ElasticsearchSuspendAnnotation: "",
			},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &elasticsearchcrd.Elasticsearch{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Namespace:   "default",
					Annotations: tt.annotations,
				},
			}
			result := GetSuspendedPodNames(es)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPodSuspended(t *testing.T) {
	es := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Annotations: map[string]string{
				elasticsearchcrd.ElasticsearchSuspendAnnotation: "pod-0,pod-1",
			},
		},
	}

	assert.True(t, IsPodSuspended(es, "pod-0"))
	assert.True(t, IsPodSuspended(es, "pod-1"))
	assert.False(t, IsPodSuspended(es, "pod-2"))
	assert.False(t, IsPodSuspended(es, ""))
}

func TestBuildSuspendedPodsConfigMap(t *testing.T) {
	t.Run("with suspended pods", func(t *testing.T) {
		es := &elasticsearchcrd.Elasticsearch{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				Annotations: map[string]string{
					elasticsearchcrd.ElasticsearchSuspendAnnotation: "pod-1,pod-0",
				},
			},
		}

		cm := buildSuspendedPodsConfigMap(es)
		assert.Equal(t, "test-suspended-pods-es", cm.Name)
		assert.Equal(t, "default", cm.Namespace)
		assert.Equal(t, "pod-0\npod-1", cm.Data["suspended_pods.txt"])
		assert.NotEmpty(t, cm.Labels)
		assert.Equal(t, "test", cm.Labels["cluster"])
		assert.Equal(t, "true", cm.Labels[elasticsearchcrd.ElasticsearchAnnotationKey])
	})

	t.Run("without annotation", func(t *testing.T) {
		es := &elasticsearchcrd.Elasticsearch{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		}

		cm := buildSuspendedPodsConfigMap(es)
		assert.Equal(t, "test-suspended-pods-es", cm.Name)
		assert.Equal(t, "", cm.Data["suspended_pods.txt"])
		assert.NotEmpty(t, cm.Labels)
	})
}

func TestReconcileSuspendedPods(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	esLabels := map[string]string{
		"cluster": "test",
		elasticsearchcrd.ElasticsearchAnnotationKey: "true",
	}

	es := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Annotations: map[string]string{
				elasticsearchcrd.ElasticsearchSuspendAnnotation: "test-data-es-0,test-data-es-1",
			},
		},
	}

	runningPod := func(name string, labels map[string]string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "elasticsearch",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}
	}

	suspendedPod := func(name string, labels map[string]string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Name: SuspendInitContainerName,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}
	}

	// Pod listed in annotation and running -> should be deleted
	pod0 := runningPod("test-data-es-0", esLabels)
	// Pod listed in annotation and already suspended -> should NOT be deleted
	pod1 := suspendedPod("test-data-es-1", esLabels)
	// Pod NOT listed and running -> should NOT be deleted
	pod2 := runningPod("test-data-es-2", esLabels)
	// Pod NOT listed and suspended -> should NOT be deleted
	pod3 := suspendedPod("test-data-es-3", esLabels)
	// Pod with wrong labels -> should be ignored
	podWrongLabels := runningPod("test-data-es-4", map[string]string{"cluster": "other"})

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod0, pod1, pod2, pod3, podWrongLabels).
		Build()

	logger := logrus.NewEntry(logrus.New())
	err := ReconcileSuspendedPods(ctx, k8sClient, es, logger)
	require.NoError(t, err)

	// pod-0: listed and running -> deleted
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-data-es-0"}, &corev1.Pod{})
	assert.True(t, apierrors.IsNotFound(err), "pod test-data-es-0 should be deleted")

	// pod-1: listed and suspended -> NOT deleted
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-data-es-1"}, &corev1.Pod{})
	assert.NoError(t, err, "pod test-data-es-1 should NOT be deleted")

	// pod-2: not listed and running -> NOT deleted
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-data-es-2"}, &corev1.Pod{})
	assert.NoError(t, err, "pod test-data-es-2 should NOT be deleted")

	// pod-3: not listed and suspended -> NOT deleted
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-data-es-3"}, &corev1.Pod{})
	assert.NoError(t, err, "pod test-data-es-3 should NOT be deleted")

	// pod with wrong labels -> NOT deleted (not even listed)
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-data-es-4"}, &corev1.Pod{})
	assert.NoError(t, err, "pod test-data-es-4 should NOT be deleted")
}

func TestReconcileSuspendedPodsSkipsTerminatingPods(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	esLabels := map[string]string{
		"cluster": "test",
		elasticsearchcrd.ElasticsearchAnnotationKey: "true",
	}

	es := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Annotations: map[string]string{
				elasticsearchcrd.ElasticsearchSuspendAnnotation: "test-data-es-0",
			},
		},
	}

	// Pod listed in the annotation and running but already terminating: it
	// must NOT be deleted again, it will restart in suspended state on its own.
	// The fake client requires a finalizer on objects with a deletionTimestamp.
	now := metav1.Time{Time: time.Now()}
	terminatingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-data-es-0",
			Namespace:         "default",
			Labels:            esLabels,
			DeletionTimestamp: &now,
			Finalizers:        []string{"test-finalizer"},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "elasticsearch",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(terminatingPod).
		Build()

	logger := logrus.NewEntry(logrus.New())
	err := ReconcileSuspendedPods(ctx, k8sClient, es, logger)
	require.NoError(t, err)

	// Pod must still exist (not deleted again)
	err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-data-es-0"}, &corev1.Pod{})
	assert.NoError(t, err, "terminating pod should NOT be deleted again")
}

func TestIsPodInSuspendedState(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			name:     "nil status",
			pod:      &corev1.Pod{},
			expected: false,
		},
		{
			name: "suspend-check init container running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: SuspendInitContainerName,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "suspend-check init container terminated",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: SuspendInitContainerName,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "other init container running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "other-init",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPodInSuspendedState(tt.pod))
		})
	}
}

func TestIsPodRunning(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			name:     "nil status",
			pod:      &corev1.Pod{},
			expected: false,
		},
		{
			name: "elasticsearch container running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "elasticsearch",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "elasticsearch container waiting",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "elasticsearch",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "other container running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "other",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPodRunning(tt.pod))
		})
	}
}

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			name:     "nil status",
			pod:      &corev1.Pod{},
			expected: false,
		},
		{
			name: "ready condition true",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "ready condition false",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "other condition true",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodScheduled,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPodReady(tt.pod))
		})
	}
}
