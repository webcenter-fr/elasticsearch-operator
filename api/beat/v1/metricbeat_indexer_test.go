package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupMetricbeatIndexer() {
	// Add metricbeat to force indexer execution

	metricbeat := &Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: MetricbeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "test",
				},
			},

			Deployment: MetricbeatDeploymentSpec{
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

	err := t.k8sClient.Create(context.Background(), metricbeat)
	assert.NoError(t.T(), err)
}
