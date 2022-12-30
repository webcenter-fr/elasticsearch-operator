package kibana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kibanaapi "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestGetSecretNameForTls(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	// With default value
	assert.Equal(t, "test-tls-kb", GetSecretNameForTls(o))

	// When specify TLS secret
	o.Spec.Tls = kibanaapi.TlsSpec{
		CertificateSecretRef: &v1.LocalObjectReference{
			Name: "my-secret",
		},
		Enabled: pointer.Bool(true),
	}
	assert.Equal(t, "my-secret", GetSecretNameForTls(o))
}

func TestGetSecretNameForPki(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-pki-kb", GetSecretNameForPki(o))
}

func TestGetSecretNameForKeystore(t *testing.T) {
	var o *kibanaapi.Kibana

	// When default value
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "", GetSecretNameForKeystore(o))

	// When keystore is provided
	o = &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{
			KeystoreSecretRef: &v1.LocalObjectReference{
				Name: "my-secret",
			},
		},
	}

	assert.Equal(t, "my-secret", GetSecretNameForKeystore(o))
}

func TestGetConfigMapName(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-config-kb", GetConfigMapName(o))
}

func TestGetServiceName(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetServiceName(o))
}

func TestGetLoadBalancerName(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-lb-kb", GetLoadBalancerName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetIngressName(o))
}

func TestGetPDBName(t *testing.T) {
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-kb", GetPDBName(o))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
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
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
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
	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
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

	o := &kibanaapi.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: kibanaapi.KibanaSpec{},
	}

	assert.Equal(t, "test-credential-kb", GetSecretNameForCredentials(o))

}
