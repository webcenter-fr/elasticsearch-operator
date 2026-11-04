package v1

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *TestSuite) TestSetupFilebeatIndexer() {
	// Add filebeat to force indexer execution

	var (
		filebeat *Filebeat
		err      error
	)

	// When Logstash target
	filebeat = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: FilebeatSpec{
			LogstashRef: &FilebeatLogstashRef{
				ManagedLogstashRef: &FilebeatLogstashManagedRef{
					Name: "test",
				},
				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},
			Deployment: FilebeatDeploymentSpec{
				AdditionalVolumes: []shared.DeploymentVolumeSpec{
					{
						Name: "config",
						VolumeMount: corev1.VolumeMount{
							MountPath: "/tmp/config",
						},
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test",
								},
							},
						},
					},
					{
						Name: "secret",
						VolumeMount: corev1.VolumeMount{
							MountPath: "/tmp/secret",
						},
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "test",
							},
						},
					},
				},
				Deployment: shared.Deployment{
					Env: []corev1.EnvVar{
						{
							Name: "config",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test",
									},
								},
							},
						},
						{
							Name: "secret",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test",
									},
								},
							},
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test",
								},
							},
						},
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test",
								},
							},
						},
					},
				},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), filebeat)
	assert.NoError(t.T(), err)

	// When Elasticsearch target
	filebeat = &Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2",
			Namespace: "default",
		},
		Spec: FilebeatSpec{
			ElasticsearchRef: &shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},
			Deployment: FilebeatDeploymentSpec{
				AdditionalVolumes: []shared.DeploymentVolumeSpec{
					{
						Name: "config",
						VolumeMount: corev1.VolumeMount{
							MountPath: "/tmp/config",
						},
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test",
								},
							},
						},
					},
					{
						Name: "secret",
						VolumeMount: corev1.VolumeMount{
							MountPath: "/tmp/secret",
						},
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "test",
							},
						},
					},
				},
				Deployment: shared.Deployment{
					Env: []corev1.EnvVar{
						{
							Name: "config",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test",
									},
								},
							},
						},
						{
							Name: "secret",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test",
									},
								},
							},
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test",
								},
							},
						},
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test",
								},
							},
						},
					},
				},
			},
		},
	}

	err = t.k8sClient.Create(context.Background(), filebeat)
	assert.NoError(t.T(), err)

	// The managed fullname indexers must index by resource name (namespace/name),
	// so a field selector on the referenced resource fullname must match.
	assert.Eventually(t.T(), func() bool {
		list := &FilebeatList{}
		if err := t.k8sClient.List(context.Background(), list, &client.ListOptions{
			Namespace: "default",
			FieldSelector: fields.SelectorFromSet(fields.Set{
				"spec.logstashRef.managed.fullname": "default/test",
			}),
		}); err != nil {
			return false
		}
		return len(list.Items) == 1 && list.Items[0].Name == "test"
	}, 10*time.Second, 100*time.Millisecond)

	assert.Eventually(t.T(), func() bool {
		list := &FilebeatList{}
		if err := t.k8sClient.List(context.Background(), list, &client.ListOptions{
			Namespace: "default",
			FieldSelector: fields.SelectorFromSet(fields.Set{
				"spec.elasticsearchRef.managed.fullname": "default/test",
			}),
		}); err != nil {
			return false
		}
		return len(list.Items) == 1 && list.Items[0].Name == "test2"
	}, 10*time.Second, 100*time.Millisecond)
}
