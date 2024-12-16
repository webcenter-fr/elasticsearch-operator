package elasticsearch

import (
	"fmt"
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
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
		o               *elasticsearchcrd.Elasticsearch
		err             error
		sts             []appv1.StatefulSet
		extraSecrets    []corev1.Secret
		extraConfigMaps []corev1.ConfigMap
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "all",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	sts, err = buildStatefulsets(o, nil, nil, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefullset-all.yml", &sts[0], scheme.Scheme)

	// With default values on Openshift
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "all",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	sts, err = buildStatefulsets(o, nil, nil, true)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefullset-all_openshift.yml", &sts[0], scheme.Scheme)

	// With complex config
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Version: "2.3.0",
			PluginsList: []string{
				"repository-s3",
			},
			SetVMMaxMapCount: ptr.To[bool](true),
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				AdditionalVolumes: []shared.DeploymentVolumeSpec{
					{
						Name: "snapshot",
						VolumeMount: corev1.VolumeMount{
							MountPath: "/mnt/snapshot",
						},
						VolumeSource: corev1.VolumeSource{
							NFS: &corev1.NFSVolumeSource{
								Server: "nfsserver",
								Path:   "/snapshot",
							},
						},
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
				KeystoreSecretRef: &corev1.LocalObjectReference{
					Name: "elasticsearch-security",
				},
				CacertsSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca",
				},
				AntiAffinity: &shared.DeploymentAntiAffinitySpec{
					TopologyKey: "rack",
					Type:        "hard",
				},
				Config: map[string]string{
					"log4j.yaml": "my log4j",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",

					Roles: []string{
						"master",
					},
					Persistence: &shared.DeploymentPersistenceSpec{
						VolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: ptr.To[string]("local-path"),
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
					Jvm: "-Xms1g -Xmx1g",
					Deployment: shared.Deployment{
						Replicas: 3,
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
					},
				},
				{
					Name: "data",

					Roles: []string{
						"data",
					},
					Persistence: &shared.DeploymentPersistenceSpec{
						Volume: &corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/data/elasticsearch",
							},
						},
					},
					Jvm: "-Xms30g -Xmx30g",
					Deployment: shared.Deployment{
						Replicas: 3,
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("5"),
								corev1.ResourceMemory: resource.MustParse("30Gi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("8"),
								corev1.ResourceMemory: resource.MustParse("64Gi"),
							},
						},
						NodeSelector: map[string]string{
							"project": "elasticsearch",
						},
						Tolerations: []corev1.Toleration{
							{
								Key:      "project",
								Operator: corev1.TolerationOpEqual,
								Value:    "elasticsearch",
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
					},
				},
				{
					Name: "client",

					Roles: []string{
						"ingest",
					},
					Persistence: &shared.DeploymentPersistenceSpec{
						VolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: ptr.To[string]("local-path"),
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
					Jvm: "-Xms2g -Xmx2g",
					Deployment: shared.Deployment{
						Replicas: 2,
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
					},
				},
			},
		},
	}

	extraSecrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "elasticsearch-security",
			},
			Data: map[string][]byte{
				"key1": []byte("secret1"),
			},
		},
	}

	// Keep only configmap of type config
	extraConfigMapsTmp, err := buildConfigMaps(o)
	if err != nil {
		t.Fatal(err.Error())
	}
	extraConfigMaps = make([]corev1.ConfigMap, 0, len(extraConfigMapsTmp))
	for _, cm := range extraConfigMapsTmp {
		if cm.Annotations[fmt.Sprintf("%s/type", elasticsearchcrd.ElasticsearchAnnotationKey)] == "config" {
			extraConfigMaps = append(extraConfigMaps, cm)
		}
	}

	sts, err = buildStatefulsets(o, extraSecrets, extraConfigMaps, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefullset-master.yml", &sts[0], scheme.Scheme)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefullset-data.yml", &sts[1], scheme.Scheme)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefullset-client.yml", &sts[2], scheme.Scheme)

	// When secret for certificates is set
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](true),
				CertificateSecretRef: &corev1.LocalObjectReference{
					Name: "api-certificates",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "all",
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	extraSecrets = []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-certificates",
			},
			Data: map[string][]byte{
				"tls.crt": []byte("secret1"),
				"tls.key": []byte("secret2"),
				"ca.crt":  []byte("secret3"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "transport-certificates",
				Annotations: map[string]string{
					"elasticsearch.k8s.webcenter.fr/sequence": "testsequence",
				},
			},
			Data: map[string][]byte{
				"tls.crt": []byte("secret1"),
				"tls.key": []byte("secret2"),
				"ca.crt":  []byte("secret3"),
			},
		},
	}

	// Keep only configmap of type config
	extraConfigMapsTmp, err = buildConfigMaps(o)
	if err != nil {
		t.Fatal(err.Error())
	}
	extraConfigMaps = make([]corev1.ConfigMap, 0, len(extraConfigMapsTmp))
	for _, cm := range extraConfigMapsTmp {
		if cm.Annotations[fmt.Sprintf("%s/type", elasticsearchcrd.ElasticsearchAnnotationKey)] == "config" {
			extraConfigMaps = append(extraConfigMaps, cm)
		}
	}

	sts, err = buildStatefulsets(o, extraSecrets, extraConfigMaps, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.StatefulSet](t, "testdata/statefullset-all-external-tls.yml", &sts[0], scheme.Scheme)
}

func TestComputeJavaOpts(t *testing.T) {
	var o *elasticsearchcrd.Elasticsearch

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Empty(t, computeJavaOpts(o, &o.Spec.NodeGroups[0]))

	// With global values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				Jvm: "-param1=1",
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "-param1=1", computeJavaOpts(o, &o.Spec.NodeGroups[0]))

	// With global and node group values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				Jvm: "-param1=1",
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Jvm:  "-xmx1G -xms1G",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "-param1=1 -xmx1G -xms1G", computeJavaOpts(o, &o.Spec.NodeGroups[0]))
}

func TestComputeRoles(t *testing.T) {
	var roles []string

	roles = []string{
		"master",
	}

	assert.Equal(t, "master", computeRoles(roles))

	roles = []string{
		"master",
		"data",
		"ingest",
	}

	assert.Equal(t, "master, data, ingest", computeRoles(roles))
}

func TestComputeAntiAffinity(t *testing.T) {
	var (
		o                    *elasticsearchcrd.Elasticsearch
		expectedAntiAffinity *corev1.PodAntiAffinity
		err                  error
		antiAffinity         *corev1.PodAntiAffinity
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	expectedAntiAffinity = &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 10,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"cluster":                        "test",
							"nodeGroup":                      "master",
							"elasticsearch.k8s.webcenter.fr": "true",
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}

	antiAffinity, err = computeAntiAffinity(o, &o.Spec.NodeGroups[0])
	assert.NoError(t, err)
	assert.Equal(t, expectedAntiAffinity, antiAffinity)

	// With global anti affinity
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				AntiAffinity: &shared.DeploymentAntiAffinitySpec{
					Type:        "hard",
					TopologyKey: "topology.kubernetes.io/zone",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	expectedAntiAffinity = &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{
				TopologyKey: "topology.kubernetes.io/zone",
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                        "test",
						"nodeGroup":                      "master",
						"elasticsearch.k8s.webcenter.fr": "true",
					},
				},
			},
		},
	}

	antiAffinity, err = computeAntiAffinity(o, &o.Spec.NodeGroups[0])
	assert.NoError(t, err)
	assert.Equal(t, expectedAntiAffinity, antiAffinity)

	// With global and node group anti affinity
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				AntiAffinity: &shared.DeploymentAntiAffinitySpec{
					Type:        "soft",
					TopologyKey: "topology.kubernetes.io/zone",
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",

					AntiAffinity: &shared.DeploymentAntiAffinitySpec{
						Type: "hard",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	expectedAntiAffinity = &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{
				TopologyKey: "topology.kubernetes.io/zone",
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                        "test",
						"nodeGroup":                      "master",
						"elasticsearch.k8s.webcenter.fr": "true",
					},
				},
			},
		},
	}

	antiAffinity, err = computeAntiAffinity(o, &o.Spec.NodeGroups[0])
	assert.NoError(t, err)
	assert.Equal(t, expectedAntiAffinity, antiAffinity)
}

func TestComputeEnvFroms(t *testing.T) {
	var (
		o                *elasticsearchcrd.Elasticsearch
		expectedEnvFroms []corev1.EnvFromSource
	)

	// With default values
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Empty(t, computeEnvFroms(o, &o.Spec.NodeGroups[0]))

	// When global envFrom
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
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
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	expectedEnvFroms = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test",
				},
			},
		},
	}

	assert.Equal(t, expectedEnvFroms, computeEnvFroms(o, &o.Spec.NodeGroups[0]))

	// When global envFrom and node group envFrom
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				EnvFrom: []corev1.EnvFromSource{
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test",
							},
						},
					},
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test2",
							},
						},
					},
				},
			},
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 1,
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test",
									},
								},
							},
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test3",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	expectedEnvFroms = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test",
				},
			},
		},
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test3",
				},
			},
		},
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "test2",
				},
			},
		},
	}

	assert.Equal(t, expectedEnvFroms, computeEnvFroms(o, &o.Spec.NodeGroups[0]))
}

func TestGetElasticsearchContainer(t *testing.T) {
	var o *appv1.StatefulSet

	// When no container
	o = &appv1.StatefulSet{
		Spec: appv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{},
		},
	}
	assert.Nil(t, getElasticsearchContainer(&o.Spec.Template))

	// When Elasticsearch container
	o = &appv1.StatefulSet{
		Spec: appv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "elasticsearch",
						},
					},
				},
			},
		},
	}
	assert.Equal(t, &o.Spec.Template.Spec.Containers[0], getElasticsearchContainer(&o.Spec.Template))
}
