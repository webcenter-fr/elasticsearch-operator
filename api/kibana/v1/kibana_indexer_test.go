package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestSetupKibanaIndexer() {
	// Add Kibana to force indexer execution

	kibana := &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: KibanaSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
				ElasticsearchCaSecretRef: &v1.LocalObjectReference{
					Name: "test",
				},
			},
			Tls: shared.TlsSpec{
				CertificateSecretRef: &v1.LocalObjectReference{
					Name: "test",
				},
			},
			KeystoreSecretRef: &v1.LocalObjectReference{
				Name: "test",
			},
			Deployment: KibanaDeploymentSpec{
				Deployment: shared.Deployment{
					Env: []v1.EnvVar{
						{
							Name: "config",
							ValueFrom: &v1.EnvVarSource{
								ConfigMapKeyRef: &v1.ConfigMapKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test",
									},
								},
							},
						},
						{
							Name: "secret",
							ValueFrom: &v1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "test",
									},
								},
							},
						},
					},
					EnvFrom: []v1.EnvFromSource{
						{
							ConfigMapRef: &v1.ConfigMapEnvSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "test",
								},
							},
							SecretRef: &v1.SecretEnvSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "test",
								},
							},
						},
					},
				},
			},
		},
	}

	err := t.k8sClient.Create(context.Background(), kibana)
	assert.NoError(t.T(), err)
}
