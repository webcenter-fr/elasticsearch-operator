package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetNodeGroupName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-es", GetNodeGroupName(o, o.Spec.NodeGroups[0].Name))
}

func TestGetNodeNames(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	expectedList := []string{
		"test-master-es-0",
		"test-master-es-1",
		"test-master-es-2",
		"test-data-es-0",
	}

	assert.Equal(t, expectedList, GetNodeNames(o))
}

func TestGetNodeGroupNodeNames(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	expectedList := []string{
		"test-master-es-0",
		"test-master-es-1",
		"test-master-es-2",
	}

	assert.Equal(t, expectedList, GetNodeGroupNodeNames(o, o.Spec.NodeGroups[0].Name))
}

func TestGetSecretNameForTlsTransport(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-tls-transport-es", GetSecretNameForTlsTransport(o))
}

func TestGetSecretNameForPkiTransport(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-pki-transport-es", GetSecretNameForPkiTransport(o))
}

func TestGetSecretNameForTlsApi(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	// With default value
	assert.Equal(t, "test-tls-api-es", GetSecretNameForTlsApi(o))

	// When specify TLS secret
	o.Spec.Tls = shared.TlsSpec{
		CertificateSecretRef: &v1.LocalObjectReference{
			Name: "my-secret",
		},
		Enabled: ptr.To[bool](true),
	}
	assert.Equal(t, "my-secret", GetSecretNameForTlsApi(o))
}

func TestGetSecretNameForPkiApi(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-pki-api-es", GetSecretNameForPkiApi(o))
}

func TestGetSecretNameForCredentials(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-credential-es", GetSecretNameForCredentials(o))
}

func TestGetNodeGroupConfigMapName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-config-es", GetNodeGroupConfigMapName(o, o.Spec.NodeGroups[0].Name))
}

func TestGetGlobalServiceName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-es", GetGlobalServiceName(o))
}

func TestGetLoadBalancerName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-lb-es", GetLoadBalancerName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-es", GetIngressName(o))
}

func TestGetNodeGroupServiceName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-es", GetNodeGroupServiceName(o, o.Spec.NodeGroups[0].Name))
}

func TestGetNodeGroupServiceNameHeadless(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-headless-es", GetNodeGroupServiceNameHeadless(o, o.Spec.NodeGroups[0].Name))
}

func TestGetNodeGroupPDBName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-master-es", GetNodeGroupPDBName(o, o.Spec.NodeGroups[0].Name))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}
	assert.Equal(t, "docker.elastic.co/elasticsearch/elasticsearch:latest", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v1"
	assert.Equal(t, "docker.elastic.co/elasticsearch/elasticsearch:v1", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v1", GetContainerImage(o))
}

func TestGetNodeGroupNameFromNodeName(t *testing.T) {
	assert.Equal(t, "my-test", GetNodeGroupNameFromNodeName("my-test-0"))
	assert.Equal(t, "", GetNodeGroupNameFromNodeName("my-test"))
}

func TestIsMasterRole(t *testing.T) {
	var o *elasticsearchcrd.Elasticsearch

	// With only master role
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"master",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.True(t, IsMasterRole(o, o.Spec.NodeGroups[0].Name))

	// With multiple role
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Roles: []string{
						"data",
						"master",
						"ingest",
					},
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.True(t, IsMasterRole(o, o.Spec.NodeGroups[0].Name))

	// Without master role
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",

					Roles: []string{
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

	assert.False(t, IsMasterRole(o, o.Spec.NodeGroups[0].Name))
}

func TestGetLabels(t *testing.T) {
	var expectedLabels map[string]string

	// With default values
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	expectedLabels = map[string]string{
		"cluster":                        "test",
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedLabels, getLabels(o))

	// With additional labels
	expectedLabels["foo"] = "bar"

	assert.Equal(t, expectedLabels, getLabels(o, map[string]string{"foo": "bar"}))
}

func TestGetAnnotations(t *testing.T) {
	var expectedAnnotations map[string]string

	// With default values
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	expectedAnnotations = map[string]string{
		"elasticsearch.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedAnnotations, getAnnotations(o))

	// With additional annottaions
	expectedAnnotations["foo"] = "bar"

	assert.Equal(t, expectedAnnotations, getAnnotations(o, map[string]string{"foo": "bar"}))
}

func TestGetUserSystemName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-kibana-system-es", GetUserSystemName(o, "kibana_system"))
}

func TestGetLicenseName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-es", GetLicenseName(o))
}

func TestGetElasticsearchNameFromSecretApiTlsName(t *testing.T) {
	assert.Equal(t, "test", GetElasticsearchNameFromSecretApiTlsName("test-tls-api-es"))
}

func TestGetNetworkPolicyName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-allow-api-es", GetNetworkPolicyName(o))
}

func TestGetExporterImage(t *testing.T) {
	var o *elasticsearchcrd.Elasticsearch

	// Default value
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "quay.io/prometheuscommunity/elasticsearch-exporter:latest", GetExporterImage(o))

	// When image is defined
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					ImageSpec: shared.ImageSpec{
						Image: "toto",
					},
				},
			},
		},
	}

	assert.Equal(t, "toto:latest", GetExporterImage(o))
}

func TestGetExporterDeployementName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-exporter-es", GetExporterDeployementName(o))
}

func TestGetPodMonitorName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-es", GetPodMonitorName(o))
}

func TestGetPublicUrl(t *testing.T) {
	var o *elasticsearchcrd.Elasticsearch

	// When ask public URL for internal application without target node group and API tls is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "https://test-es.default.svc:9200", GetPublicUrl(o, "", false))

	// When ask public URL for internal application without target node group and API tls is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	assert.Equal(t, "http://test-es.default.svc:9200", GetPublicUrl(o, "", false))

	// When ask public URL for internal application with target node group and API tls is enabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "https://test-client-es.default.svc:9200", GetPublicUrl(o, "client", false))

	// When ask public URL for internal application with target node group and API tls is disabled
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Tls: shared.TlsSpec{
				Enabled: ptr.To[bool](false),
			},
		},
	}

	assert.Equal(t, "http://test-client-es.default.svc:9200", GetPublicUrl(o, "client", false))

	// When external url
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
		Status: elasticsearchcrd.ElasticsearchStatus{
			Url: "https://fake",
		},
	}

	assert.Equal(t, "https://fake", GetPublicUrl(o, "", true))
}

func TestGetMetricbeatName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "test-metricbeat-es", GetMetricbeatName(o))
}

func TestGetBootstrappingConfigMapName(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{
					Name: "master",
					Deployment: shared.Deployment{
						Replicas: 3,
					},
				},
				{
					Name: "data",
					Deployment: shared.Deployment{
						Replicas: 1,
					},
				},
			},
		},
	}

	assert.Equal(t, "test-bootstrapping-es", GetBootstrappingConfigMapName(o))
}

func TestGetSecretNameForKeystore(t *testing.T) {
	// When no keystore
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "", GetSecretNameForKeystore(o))

	// When keystore
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				KeystoreSecretRef: &v1.LocalObjectReference{
					Name: "test",
				},
			},
		},
	}

	assert.Equal(t, "test", GetSecretNameForKeystore(o))
}

func TestGetSecretNameForCacerts(t *testing.T) {
	// When no cacerts
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{},
	}

	assert.Equal(t, "", GetSecretNameForCacerts(o))

	// When keystore
	o = &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			GlobalNodeGroup: elasticsearchcrd.ElasticsearchGlobalNodeGroupSpec{
				CacertsSecretRef: &v1.LocalObjectReference{
					Name: "test",
				},
			},
		},
	}

	assert.Equal(t, "test", GetSecretNameForCacerts(o))
}
