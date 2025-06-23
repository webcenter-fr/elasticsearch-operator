package v1

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *TestSuite) TestKibanaWebhook() {
	var (
		o   *Kibana
		err error
	)

	// Need failed when not specify target Opensearch cluster
	o = &Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: KibanaSpec{
			ElasticsearchRef: shared.ElasticsearchRef{},
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
	err = t.k8sClient.Create(context.Background(), o)
	assert.Error(t.T(), err)
}
