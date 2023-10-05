package filebeat

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestBuildStatefulset(t *testing.T) {

	var (
		o               *beatcrd.Filebeat
		es              *elasticsearchcrd.Elasticsearch
		ls              *logstashcrd.Logstash
		err             error
		sts             []appv1.StatefulSet
		extraSecrets    []corev1.Secret
		extraConfigMaps []corev1.ConfigMap
	)

	// With default values and elasticsearch managed by operator
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Deployment: beatcrd.FilebeatDeploymentSpec{
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

	sts, err = buildStatefulsets(o, es, nil, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_default_elasticsearch.yml", &sts[0], scheme.Scheme)

	// With default values and external elasticsearch
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Deployment: beatcrd.FilebeatDeploymentSpec{
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

	sts, err = buildStatefulsets(o, nil, nil, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_default_with_external_es.yml", &sts[0], scheme.Scheme)

	// With default values and external elasticsearch and custom CA Elasticsearch
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Deployment: beatcrd.FilebeatDeploymentSpec{
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

	sts, err = buildStatefulsets(o, nil, nil, extraSecrets, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_custom_ca_es_with_external_es.yml", &sts[0], scheme.Scheme)

	// With default values and logstash managed by operator
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			LogstashRef: beatcrd.FilebeatLogstashRef{
				ManagedLogstashRef: &beatcrd.FilebeatLogstashManagedRef{
					Name:          "test",
					TargetService: "beat",
					Port:          5002,
				},
				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-ls",
				},
			},
			Deployment: beatcrd.FilebeatDeploymentSpec{
				Replicas: 1,
			},
		},
	}
	ls = &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: logstashcrd.LogstashSpec{},
	}

	sts, err = buildStatefulsets(o, nil, ls, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_default_logstash.yml", &sts[0], scheme.Scheme)

	// With default values and external logstash
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Deployment: beatcrd.FilebeatDeploymentSpec{
				Replicas: 1,
			},
			LogstashRef: beatcrd.FilebeatLogstashRef{
				ExternalLogstashRef: &beatcrd.FilebeatLogstashExternalRef{
					Addresses: []string{
						"beat.logstash.svc:5002",
					},
				},
			},
		},
	}

	sts, err = buildStatefulsets(o, nil, nil, nil, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_default_with_external_ls.yml", &sts[0], scheme.Scheme)

	// With default values and external logstash and custom CA Logstash
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			Deployment: beatcrd.FilebeatDeploymentSpec{
				Replicas: 1,
			},
			LogstashRef: beatcrd.FilebeatLogstashRef{
				ExternalLogstashRef: &beatcrd.FilebeatLogstashExternalRef{
					Addresses: []string{
						"beat.logstash.svc:5002",
					},
				},
				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-ls",
				},
			},
		},
	}
	extraSecrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "custom-ca-ls",
			},
			Data: map[string][]byte{
				"ca.crt": []byte("secret3"),
			},
		},
	}

	sts, err = buildStatefulsets(o, nil, nil, extraSecrets, nil)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_custom_ca_ls_with_external_ls.yml", &sts[0], scheme.Scheme)

	// With complexe sample
	o = &beatcrd.Filebeat{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: beatcrd.FilebeatSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "test",
				},
			},
			Deployment: beatcrd.FilebeatDeploymentSpec{
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
					"project": "filebeat",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "project",
						Operator: corev1.TolerationOpEqual,
						Value:    "filebeat",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
				AntiAffinity: &beatcrd.FilebeatAntiAffinitySpec{
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
				Ports: []corev1.ContainerPort{
					{
						Name:          "beat",
						ContainerPort: 1234,
						Protocol:      corev1.ProtocolTCP,
						HostPort:      1234,
					},
				},
				Persistence: &beatcrd.FilebeatPersistenceSpec{
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
			Config: map[string]string{
				"log4j.yaml": "my log4j",
			},
			Module: map[string]string{
				"module.yaml": "my module",
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
				Name:      "test-config-fb",
			},
			Data: map[string]string{
				"fu": "bar",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-module-fb",
			},
			Data: map[string]string{
				"fu": "bar",
			},
		},
	}

	sts, err = buildStatefulsets(o, es, nil, extraSecrets, extraConfigMaps)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefulset_complet.yml", &sts[0], scheme.Scheme)
}
