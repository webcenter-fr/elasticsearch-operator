package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildDeployment(t *testing.T) {

	var (
		o               *cerebrocrd.Cerebro
		err             error
		dpl             *appv1.Deployment
		checksumSecrets []corev1.Secret
		checksumCms     []corev1.ConfigMap
	)

	// With default values and elasticsearch managed by operator
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Replicas: 1,
			},
		},
	}

	dpl, err = BuildDeployment(o, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_default.yml", dpl, test.CleanApi)

	// With complexe sample
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
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
			Version: "8.5.1",
			Config: map[string]string{
				"log4j.yaml": "my log4j",
			},
		},
	}
	checksumSecrets = []corev1.Secret{
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

	cm, err := BuildConfigMap(o, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	checksumCms = append(checksumCms, *cm)

	dpl, err = BuildDeployment(o, checksumSecrets, checksumCms)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_complet.yml", dpl, test.CleanApi)
}
