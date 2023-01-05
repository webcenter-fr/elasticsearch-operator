package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildDeployment(t *testing.T) {

	var (
		o   *kibanacrd.Kibana
		es  *elasticsearchcrd.Elasticsearch
		err error
		dpl *appv1.Deployment
		s   *corev1.Secret
	)

	// With default values and elasticsearch managed by operator
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			ElasticsearchRef: &kibanacrd.ElasticsearchRef{
				Name: "test",
			},
			Deployment: kibanacrd.DeploymentSpec{
				Replicas: 1,
			},
		},
	}
	es = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	dpl, err = BuildDeployment(o, es, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_default.yml", dpl, test.CleanApi)

	// With default values and external elasticsearch
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Deployment: kibanacrd.DeploymentSpec{
				Replicas: 1,
			},
		},
	}
	es = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	dpl, err = BuildDeployment(o, es, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_default_with_external_es.yml", dpl, test.CleanApi)

	// When use external API cert
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			ElasticsearchRef: &kibanacrd.ElasticsearchRef{
				Name: "test",
			},
			Deployment: kibanacrd.DeploymentSpec{
				Replicas: 1,
			},
			Tls: kibanacrd.TlsSpec{
				CertificateSecretRef: &v1.LocalObjectReference{
					Name: "api-certificates",
				},
			},
		},
	}
	es = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "api-certificates",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("secret1"),
			"tls.key": []byte("secret2"),
			"ca.crt":  []byte("secret3"),
		},
	}

	dpl, err = BuildDeployment(o, es, nil, s)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_with_external_certs.yml", dpl, test.CleanApi)

	// With complexe sample
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			ElasticsearchRef: &kibanacrd.ElasticsearchRef{
				Name: "test",
			},
			Deployment: kibanacrd.DeploymentSpec{
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
				Node: "--param1",
				InitContainerResources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("500Mi"),
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
				AntiAffinity: &kibanacrd.AntiAffinitySpec{
					TopologyKey: "rack",
					Type:        "hard",
				},
				Env: []v1.EnvVar{
					{
						Name:  "env1",
						Value: "value1",
					},
				},
				EnvFrom: []v1.EnvFromSource{
					{
						ConfigMapRef: &v1.ConfigMapEnvSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test",
							},
						},
					},
				},
			},
			Version: "8.5.1",
			KeystoreSecretRef: &v1.LocalObjectReference{
				Name: "keystore",
			},
			Config: map[string]string{
				"log4j.yaml": "my log4j",
			},
		},
	}
	es = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	s = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "keystore",
		},
		Data: map[string][]byte{
			"key1": []byte("value1"),
		},
	}

	dpl, err = BuildDeployment(o, es, s, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/deployment_complet.yml", dpl, test.CleanApi)
}
