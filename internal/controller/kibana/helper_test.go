package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetSecretNameForTls(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	// With default value
	assert.Equal(t, "test-tls-kb", GetSecretNameForTls(o))

	// When specify TLS secret
	o.Spec.Tls = shared.TlsSpec{
		CertificateSecretRef: &v1.LocalObjectReference{
			Name: "my-secret",
		},
		Enabled: ptr.To[bool](true),
	}
	assert.Equal(t, "my-secret", GetSecretNameForTls(o))
}

func TestGetSecretNameForPki(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-pki-kb", GetSecretNameForPki(o))
}

func TestGetSecretNameForCAElasticsearch(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-ca-es-kb", GetSecretNameForCAElasticsearch(o))
}

func TestGetSecretNameForKeystore(t *testing.T) {
	var o *kibanacrd.Kibana

	// When default value
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "", GetSecretNameForKeystore(o))

	// When keystore is provided
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			KeystoreSecretRef: &v1.LocalObjectReference{
				Name: "my-secret",
			},
		},
	}

	assert.Equal(t, "my-secret", GetSecretNameForKeystore(o))
}

func TestGetConfigMapName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-config-kb", GetConfigMapName(o))
}

func TestGetServiceName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetServiceName(o))
}

func TestGetLoadBalancerName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-lb-kb", GetLoadBalancerName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetIngressName(o))
}

func TestGetPDBName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetPDBName(o))
}

func TestGetDeploymentName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetDeploymentName(o))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}
	assert.Equal(t, "docker.elastic.co/kibana/kibana:latest", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v1"
	assert.Equal(t, "docker.elastic.co/kibana/kibana:v1", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v1", GetContainerImage(o))
}

func TestGetLabels(t *testing.T) {
	var expectedLabels map[string]string

	// With default values
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	expectedLabels = map[string]string{
		"cluster":                 "test",
		"kibana.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedLabels, getLabels(o))

	// With additional labels
	expectedLabels["foo"] = "bar"

	assert.Equal(t, expectedLabels, getLabels(o, map[string]string{"foo": "bar"}))
}

func TestGetAnnotations(t *testing.T) {
	var expectedAnnotations map[string]string

	// With default values
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	expectedAnnotations = map[string]string{
		"kibana.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedAnnotations, getAnnotations(o))

	// With additional annottaions
	expectedAnnotations["foo"] = "bar"

	assert.Equal(t, expectedAnnotations, getAnnotations(o, map[string]string{"foo": "bar"}))
}

func TestGetSecretNameForCredentials(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-credential-kb", GetSecretNameForCredentials(o))
}

func TestGetNetworkPolicyName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-allow-api-kb", GetNetworkPolicyName(o))
}

func TestGetNetworkPolicyElasticsearchName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-allow-es-kb", GetNetworkPolicyElasticsearchName(o))
}

func TestGetExporterUrl(t *testing.T) {
	var o *kibanacrd.Kibana

	// Default value
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "https://github.com/pjhampton/kibana-prometheus-exporter/releases/download/8.6.0/kibanaPrometheusExporter-8.6.0.zip", GetExporterUrl(o))

	// When version is set
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Version: "8.0.0",
		},
	}

	assert.Equal(t, "https://github.com/pjhampton/kibana-prometheus-exporter/releases/download/8.0.0/kibanaPrometheusExporter-8.0.0.zip", GetExporterUrl(o))

	// When URL is set
	o = &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{
			Version: "8.0.0",
			Monitoring: shared.MonitoringSpec{
				Prometheus: &shared.MonitoringPrometheusSpec{
					Url: "fake",
				},
			},
		},
	}

	assert.Equal(t, "fake", GetExporterUrl(o))
}

func TestGetPodMonitorName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetPodMonitorName(o))
}

func TestGetMetricbeatName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-metricbeat-kb", GetMetricbeatName(o))
}

func TestGetServiceAccountName(t *testing.T) {
	o := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanacrd.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetServiceAccountName(o))
}
