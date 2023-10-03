package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildStatefulset(t *testing.T) {

	var (
		o               *logstashcrd.Logstash
		es              *elasticsearchcrd.Elasticsearch
		err             error
		sts             []appv1.StatefulSet
		extraSecrets    []corev1.Secret
		extraConfigMaps []corev1.ConfigMap
	)

	// With default values and elasticsearch managed by operator
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Deployment: logstashcrd.LogstashDeploymentSpec{
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

	sts, err = buildStatefulsets(o, es, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/statefulset_default.yml", &sts[0], test.CleanApi)

	// With default values and external elasticsearch
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Deployment: logstashcrd.LogstashDeploymentSpec{
				Replicas: 1,
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{
						"https://es1:9200",
					},
					SecretRef: &corev1.LocalObjectReference{
						Name: "es-credential",
					},
				},
			},
		},
	}

	sts, err = buildStatefulsets(o, nil, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/statefulset_default_with_external_es.yml", &sts[0], test.CleanApi)

	// With default values and external elasticsearch and custom CA Elasticsearch
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			Deployment: logstashcrd.LogstashDeploymentSpec{
				Replicas: 1,
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{
						"https://es1:9200",
					},
					SecretRef: &corev1.LocalObjectReference{
						Name: "es-credential",
					},
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-es",
				},
			},
		},
	}
	extraSecrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "custom-ca-es",
			},
			Data: map[string][]byte{
				"ca.crt": []byte("secret3"),
			},
		},
	}

	sts, err = buildStatefulsets(o, nil, extraSecrets, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/statefulset_custom_ca_es_with_external_es.yml", &sts[0], test.CleanApi)

	// With complexe sample
	o = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Deployment: logstashcrd.LogstashDeploymentSpec{
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
				Jvm: "-Xms1G -Xmx1G",
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
					"project": "logstash",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "project",
						Operator: corev1.TolerationOpEqual,
						Value:    "logstash",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
				AntiAffinity: &logstashcrd.LogstashAntiAffinitySpec{
					TopologyKey: "rack",
					Type:        "hard",
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
				Persistence: &logstashcrd.LogstashPersistenceSpec{
					VolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{
						StorageClassName: ptr.To[string]("local-path"),
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
					},
				},
			},
			Version: "8.5.1",
			KeystoreSecretRef: &corev1.LocalObjectReference{
				Name: "keystore",
			},
			Config: map[string]string{
				"log4j.yaml": "my log4j",
			},
			Pipeline: map[string]string{
				"pipeline.yaml": "my pipeline",
			},
			Pattern: map[string]string{
				"pattern.conf": "my pattern",
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

	extraSecrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "keystore",
			},
			Data: map[string][]byte{
				"fu": []byte("bar"),
			},
		},
	}

	extraConfigMaps = []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test",
			},
			Data: map[string]string{
				"fu": "bar",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-config-ls",
				Annotations: map[string]string{
					"logstash.k8s.webcenter.fr/config-type": "config",
				},
			},
			Data: map[string]string{
				"fu": "bar",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-pipeline-ls",
				Annotations: map[string]string{
					"logstash.k8s.webcenter.fr/config-type": "pipeline",
				},
			},
			Data: map[string]string{
				"fu": "bar",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-pattern-ls",
				Annotations: map[string]string{
					"logstash.k8s.webcenter.fr/config-type": "pattern",
				},
			},
			Data: map[string]string{
				"fu": "bar",
			},
		},
	}

	sts, err = buildStatefulsets(o, es, extraSecrets, extraConfigMaps)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "testdata/statefulset_complet.yml", &sts[0], test.CleanApi)
}
