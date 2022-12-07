package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestGenerateStatefullset(t *testing.T) {

	var (
		o *elasticsearchapi.Elasticsearch
		err error
		sts []*appv1.StatefulSet
	)

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "all",
					Replicas: 1,
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
				},
			},
		},
	}

	sts, err = BuildStatefullsets(o)
	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "../../fixture/api/os-statefullset-all.yml", sts[0])

	// With complex config
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			Version: "2.3.0",
			PluginsList: []string{
			  "repository-s3",
			},
			SetVMMaxMapCount: pointer.Bool(true),
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				AdditionalVolumes: []elasticsearchapi.VolumeSpec{
					{
						Name: "snapshot",
						VolumeMount: corev1.VolumeMount{
							MountPath: "/mnt/snapshot",
						},
						VolumeSource: corev1.VolumeSource{
							NFS: &corev1.NFSVolumeSource{
								Server: "nfsserver",
								Path: "/snapshot",
							},
						},
					},
				},
				InitContainerResources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("500Mi"),
					},
				},
				KeystoreSecretRef: &corev1.LocalObjectReference{
					Name: "elasticsearch-security",
				},
				AntiAffinity: &elasticsearchapi.AntiAffinitySpec{
					TopologyKey: "rack",
					Type: "hard",
				},
				Config: map[string]string{
					"log4.yaml": "my log4j",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 3,
					Roles: []string{
						"cluster_manager",
					},
					Persistence: &elasticsearchapi.PersistenceSpec{
						VolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: pointer.String("local-path"),
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
					Jvm: "-Xms1g -Xmx1g",
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
				{
					Name: "data",
					Replicas: 3,
					Roles: []string{
						"data",
					},
					Persistence: &elasticsearchapi.PersistenceSpec{
						Volume: &corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/data/elasticsearch",
							},
						},
					},
					Jvm: "-Xms30g -Xmx30g",
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
							corev1.ResourceMemory: resource.MustParse("30Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("8"),
							corev1.ResourceMemory: resource.MustParse("64Gi"),
						},
					},
					NodeSelector: map[string]string{
						"project": "elasticsearch",
					},
					Tolerations: []corev1.Toleration{
						{
							Key: "project",
							Operator: corev1.TolerationOpEqual,
							Value: "elasticsearch",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
				{
					Name: "client",
					Replicas: 2,
					Roles: []string{
						"ingest",
					},
					Persistence: &elasticsearchapi.PersistenceSpec{
						VolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: pointer.String("local-path"),
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
					Jvm: "-Xms2g -Xmx2g",
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
		},
	}

	sts, err = BuildStatefullsets(o)

	assert.NoError(t, err)
	test.EqualFromYamlFile(t, "../../fixture/api/os-statefullset-master.yml", sts[0])
	test.EqualFromYamlFile(t, "../../fixture/api/os-statefullset-data.yml", sts[1])
	test.EqualFromYamlFile(t, "../../fixture/api/os-statefullset-client.yml", sts[2])
}


func TestComputeJavaOpts(t *testing.T) {

	var o *elasticsearchapi.Elasticsearch

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
				},
			},
		},
	}

	assert.Empty(t, computeJavaOpts(o, &o.Spec.NodeGroups[0]))

	// With global values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				Jvm: "-param1=1",
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
				},
			},
		},
	}

	assert.Equal(t, "-param1=1", computeJavaOpts(o, &o.Spec.NodeGroups[0]))

	// With global and node group values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				Jvm: "-param1=1",
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
					Jvm: "-xmx1G -xms1G",
				},
			},
		},
	}

	assert.Equal(t, "-param1=1 -xmx1G -xms1G", computeJavaOpts(o, &o.Spec.NodeGroups[0]))
}


func TestComputeInitialMasterNodes(t *testing.T) {
	var (
		o *elasticsearchapi.Elasticsearch
	)

	// With only one master
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 3,
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-es-0 test-master-es-1 test-master-es-2", computeInitialMasterNodes(o))

	// With multiple node groups
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "all",
					Replicas: 3,
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
				},
				{
					Name: "master",
					Replicas: 3,
					Roles: []string{
						"master",
					},
				},
			},
		},
	}

	assert.Equal(t, "test-all-es-0 test-all-es-1 test-all-es-2 test-master-es-0 test-master-es-1 test-master-es-2", computeInitialMasterNodes(o))
}

func TestComputeDiscoverySeedHosts(t *testing.T) {
	var (
		o *elasticsearchapi.Elasticsearch
	)

	// With only one master
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 3,
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-headless-es", computeDiscoverySeedHosts(o))

	// With multiple node groups
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "all",
					Replicas: 3,
					Roles: []string{
						"master",
						"data",
						"ingest",
					},
				},
				{
					Name: "master",
					Replicas: 3,
					Roles: []string{
						"master",
					},
				},
			},
		},
	}

	assert.Equal(t, "test-all-headless-es test-master-headless-es", computeDiscoverySeedHosts(o))
}

func TestComputeRoles(t *testing.T) {
	roles := []string {
		"master",
	}

	expectedEnvs := []corev1.EnvVar {
		{
			Name: "node.master",
			Value: "true",
		},
		{
			Name: "node.data",
			Value: "false",
		},
		{
			Name: "node.ingest",
			Value: "false",
		},
		{
			Name: "node.ml",
			Value: "false",
		},
		{
			Name: "node.remote_cluster_client",
			Value: "false",
		},
		{
			Name: "node.transform",
			Value: "false",
		},
	}

	assert.Equal(t, expectedEnvs, computeRoles(roles))
}

func TestComputeAntiAffinity(t *testing.T) {

	var (
		o *elasticsearchapi.Elasticsearch
		expectedAntiAffinity *corev1.PodAntiAffinity
		err error
		antiAffinity *corev1.PodAntiAffinity
	)

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
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
							"cluster": "test",
							"nodeGroup": "master",
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}

	antiAffinity, err = computeAntiAffinity(o, &o.Spec.NodeGroups[0])
	assert.NoError(t, err )
	assert.Equal(t, expectedAntiAffinity, antiAffinity)


	// With global anti affinity
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				AntiAffinity: &elasticsearchapi.AntiAffinitySpec{
					Type: "hard",
					TopologyKey: "topology.kubernetes.io/zone",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
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
						"cluster": "test",
						"nodeGroup": "master",
					},
				},
			},
		},
	}

	antiAffinity, err = computeAntiAffinity(o, &o.Spec.NodeGroups[0])
	assert.NoError(t, err )
	assert.Equal(t, expectedAntiAffinity, antiAffinity)

	// With global and node group anti affinity
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
				AntiAffinity: &elasticsearchapi.AntiAffinitySpec{
					Type: "soft",
					TopologyKey: "topology.kubernetes.io/zone",
				},
			},
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
					AntiAffinity: &elasticsearchapi.AntiAffinitySpec{
						Type: "hard",
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
						"cluster": "test",
						"nodeGroup": "master",
					},
				},
			},
		},
	}

	antiAffinity, err = computeAntiAffinity(o, &o.Spec.NodeGroups[0])
	assert.NoError(t, err )
	assert.Equal(t, expectedAntiAffinity, antiAffinity)
}

func TestComputeEnvFroms(t *testing.T) {
	var (
		o *elasticsearchapi.Elasticsearch
		expectedEnvFroms []corev1.EnvFromSource
	)

	// With default values
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
				},
			},
		},
	}

	assert.Empty(t, computeEnvFroms(o, &o.Spec.NodeGroups[0]))

	// When global envFrom
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
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
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
					Replicas: 1,
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
	o = &elasticsearchapi.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name: "test",
		},
		Spec: elasticsearchapi.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchapi.GlobalNodeGroupSpec{
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
			NodeGroups: []elasticsearchapi.NodeGroupSpec{
				{
					Name: "master",
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