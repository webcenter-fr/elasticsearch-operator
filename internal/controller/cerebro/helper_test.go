package cerebro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetConfigMapName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-config-cb", GetConfigMapName(o))
}

func TestGetServiceName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-cb", GetServiceName(o))
}

func TestGetLoadBalancerName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-lb-cb", GetLoadBalancerName(o))
}

func TestGetIngressName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-cb", GetIngressName(o))
}

func TestGetDeploymentName(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	assert.Equal(t, "test-cb", GetDeploymentName(o))
}

func TestGetContainerImage(t *testing.T) {
	// With default values
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}
	assert.Equal(t, "ghcr.io/disaster37/cerebro:v0.1.0", GetContainerImage(o))

	// When version is specified
	o.Spec.Version = "v0.2.0"
	assert.Equal(t, "ghcr.io/disaster37/cerebro:v0.2.0", GetContainerImage(o))

	// When image is overwriten
	o.Spec.Image = "my-image"
	assert.Equal(t, "my-image:v0.2.0", GetContainerImage(o))

	// When the legacy `latest` default is still persisted, it is remapped
	o.Spec.Image = ""
	o.Spec.Version = "latest"
	assert.Equal(t, "ghcr.io/disaster37/cerebro:v0.1.0", GetContainerImage(o))
}

func TestIsLegacyVersion(t *testing.T) {
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	// Empty version is not legacy
	assert.False(t, IsLegacyVersion(o))

	// Explicit version is not legacy
	o.Spec.Version = "v0.1.0"
	assert.False(t, IsLegacyVersion(o))

	// Legacy `latest` default is detected
	o.Spec.Version = "latest"
	assert.True(t, IsLegacyVersion(o))
}

func TestGetLabels(t *testing.T) {
	var expectedLabels map[string]string

	// With default values
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	expectedLabels = map[string]string{
		"cluster":                  "test",
		"cerebro.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedLabels, getLabels(o))

	// With additional labels
	expectedLabels["foo"] = "bar"

	assert.Equal(t, expectedLabels, getLabels(o, map[string]string{"foo": "bar"}))
}

func TestGetAnnotations(t *testing.T) {
	var expectedAnnotations map[string]string

	// With default values
	o := &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{},
	}

	expectedAnnotations = map[string]string{
		"cerebro.k8s.webcenter.fr": "true",
	}

	assert.Equal(t, expectedAnnotations, getAnnotations(o))

	// With additional annottaions
	expectedAnnotations["foo"] = "bar"

	assert.Equal(t, expectedAnnotations, getAnnotations(o, map[string]string{"foo": "bar"}))
}

func TestShouldRolloutOnGeneratedConfigChange(t *testing.T) {
	assert.False(t, shouldRolloutOnGeneratedConfigChange(nil))
	assert.False(t, shouldRolloutOnGeneratedConfigChange(&cerebrocrd.Cerebro{}))
	assert.False(t, shouldRolloutOnGeneratedConfigChange(&cerebrocrd.Cerebro{
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				ConfigUpdateStrategy: cerebrocrd.ConfigUpdateStrategyHotReload,
			},
		},
	}))
	assert.True(t, shouldRolloutOnGeneratedConfigChange(&cerebrocrd.Cerebro{
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				ConfigUpdateStrategy: cerebrocrd.ConfigUpdateStrategyRestart,
			},
		},
	}))
}

func TestEffectiveVersionAndPreHotReloadWarning(t *testing.T) {
	cases := []struct {
		version     string
		wantVersion string
		wantPreHot  bool
	}{
		{"", defaultVersion, true},
		{"latest", defaultVersion, true},
		{"v0.1.0", "v0.1.0", true},
		{"0.0.6", "0.0.6", true},
		{"v0.0.6", "v0.0.6", true},
		{"0.0.7", "0.0.7", false},
		{"v0.2.0", "v0.2.0", false},
		{"dev", "dev", false},
	}
	for _, tc := range cases {
		o := &cerebrocrd.Cerebro{Spec: cerebrocrd.CerebroSpec{Version: tc.version}}
		assert.Equal(t, tc.wantVersion, effectiveVersion(o), "version %q", tc.version)
		assert.Equal(t, tc.wantPreHot, isKnownPreHotReloadVersion(effectiveVersion(o)), "version %q", tc.version)
	}
	assert.Equal(t, defaultVersion, effectiveVersion(nil))
}
