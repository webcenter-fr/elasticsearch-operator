package cerebro

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestBuildDeployment(t *testing.T) {
	var (
		o               *cerebrocrd.Cerebro
		err             error
		dpls            []*appv1.Deployment
		checksumSecrets []*corev1.Secret
		checksumCms     []*corev1.ConfigMap
	)

	// With default values and elasticsearch managed by operator
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 1,
				},
			},
		},
	}

	dpls, err = buildDeployments(o, nil, nil, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_default.yml", dpls[0], scheme.Scheme)

	// With default values on Openshift
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 1,
				},
			},
		},
	}

	dpls, err = buildDeployments(o, nil, nil, true)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_default_openshift.yml", dpls[0], scheme.Scheme)

	// With complexe sample
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 1,
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
					NodeSelector: map[string]string{
						"project": "kibana",
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "project",
							Operator: corev1.TolerationOpEqual,
							Value:    "kibana",
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "env1",
							Value: "value1",
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
					},
				},
			},
			Version: "8.5.1",
			ExtraConfigs: map[string]string{
				"log4j.yaml": "my log4j",
			},
		},
	}
	checksumSecrets = []*corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "keystore",
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
			},
		},
	}

	cms, err := buildConfigMaps(o, nil, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	checksumCms = append(checksumCms, cms...)

	dpls, err = buildDeployments(o, checksumSecrets, checksumCms, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_complet.yml", dpls[0], scheme.Scheme)
}
