package cerebro

import (
	"context"
	"os"
	"testing"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	sigsyaml "sigs.k8s.io/yaml"
)

func TestBuildDeployment(t *testing.T) {
	var (
		o               *cerebrocrd.Cerebro
		err             error
		dpls            []*appv1.Deployment
		checksumSecrets []*corev1.Secret
		checksumCms     []*corev1.ConfigMap
	)

	// With default values and elasticsearch managed by operator
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 1,
				},
			},
		},
	}

	dpls, err = buildDeployments(o, nil, nil, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_default.yml", dpls[0], scheme.Scheme)

	// With default values on Openshift
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment: shared.Deployment{
					Replicas: 1,
				},
			},
		},
	}

	dpls, err = buildDeployments(o, nil, nil, true)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_default_openshift.yml", dpls[0], scheme.Scheme)

	// With complexe sample
	o = &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment: shared.Deployment{
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
			},
			Version: "v0.1.0",
			ExtraConfigs: map[string]string{
				"log4j.yaml": "my log4j",
			},
		},
	}
	checksumSecrets = []*corev1.Secret{
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

	cms, err := buildConfigMaps(o, nil, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	checksumCms = append(checksumCms, cms...)

	dpls, err = buildDeployments(o, checksumSecrets, checksumCms, false)
	assert.NoError(t, err)
	test.EqualFromYamlFile[*appv1.Deployment](t, "testdata/deployment_complet.yml", dpls[0], scheme.Scheme)
}

func newCerebroForStrategy(strategy cerebrocrd.ConfigUpdateStrategy) *cerebrocrd.Cerebro {
	return &cerebrocrd.Cerebro{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"},
		Spec: cerebrocrd.CerebroSpec{
			Deployment: cerebrocrd.CerebroDeploymentSpec{
				Deployment:           shared.Deployment{Replicas: 1},
				ConfigUpdateStrategy: strategy,
			},
		},
	}
}

func generatedConfigMaps(t *testing.T, o *cerebrocrd.Cerebro) []*corev1.ConfigMap {
	t.Helper()
	cms, err := buildConfigMaps(o, nil, nil)
	require.NoError(t, err)
	return cms
}

func TestBuildDeployment_ConfigMapChangeDoesNotChangePodTemplate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		strategy cerebrocrd.ConfigUpdateStrategy
	}{
		{"unset", ""},
		{"hot reload", cerebrocrd.ConfigUpdateStrategyHotReload},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := newCerebroForStrategy(tc.strategy)
			cms1 := generatedConfigMaps(t, o)
			cms2 := generatedConfigMaps(t, o)
			cms2[0] = cms2[0].DeepCopy()
			cms2[0].Data["application.yaml"] += "\n# changed\n"

			dpl1, err := buildDeployments(o, nil, cms1, false)
			require.NoError(t, err)
			dpl2, err := buildDeployments(o, nil, cms2, false)
			require.NoError(t, err)

			assert.NotContains(t, dpl1[0].Spec.Template.Annotations, "cerebro.k8s.webcenter.fr/configmap-test-config-cb")
			assert.Equal(t, dpl1[0].Spec.Template, dpl2[0].Spec.Template)

			external := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ext-env-cm"},
				Data:       map[string]string{"key": "value"},
			}
			dpl3, err := buildDeployments(o, nil, append(generatedConfigMaps(t, o), external), false)
			require.NoError(t, err)
			assert.Contains(t, dpl3[0].Spec.Template.Annotations, "cerebro.k8s.webcenter.fr/configmap-ext-env-cm")
		})
	}
}

func TestBuildDeployment_RestartStrategyIncludesConfigMapChecksum(t *testing.T) {
	o := newCerebroForStrategy(cerebrocrd.ConfigUpdateStrategyRestart)
	cms1 := generatedConfigMaps(t, o)
	cms2 := generatedConfigMaps(t, o)
	cms2[0] = cms2[0].DeepCopy()
	cms2[0].Data["application.yaml"] += "\n# changed\n"

	dpl1, err := buildDeployments(o, nil, cms1, false)
	require.NoError(t, err)
	dpl2, err := buildDeployments(o, nil, cms2, false)
	require.NoError(t, err)

	assert.Contains(t, dpl1[0].Spec.Template.Annotations, "cerebro.k8s.webcenter.fr/configmap-test-config-cb")
	assert.Contains(t, dpl2[0].Spec.Template.Annotations, "cerebro.k8s.webcenter.fr/configmap-test-config-cb")
	assert.NotEqual(t, dpl1[0].Spec.Template, dpl2[0].Spec.Template)
}

func TestBuildDeployment_SecretChangeUpdatesPodTemplate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		strategy cerebrocrd.ConfigUpdateStrategy
	}{
		{"unset", ""},
		{"hot reload", cerebrocrd.ConfigUpdateStrategyHotReload},
		{"restart", cerebrocrd.ConfigUpdateStrategyRestart},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := newCerebroForStrategy(tc.strategy)
			secret1 := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-application-cb"},
				Data:       map[string][]byte{"session-key": []byte("v1")},
			}
			secret2 := secret1.DeepCopy()
			secret2.Data["session-key"] = []byte("v2")

			dpl1, err := buildDeployments(o, []*corev1.Secret{secret1}, nil, false)
			require.NoError(t, err)
			dpl2, err := buildDeployments(o, []*corev1.Secret{secret2}, nil, false)
			require.NoError(t, err)

			assert.Contains(t, dpl1[0].Spec.Template.Annotations, "cerebro.k8s.webcenter.fr/secret-test-application-cb")
			assert.NotEqual(t, dpl1[0].Spec.Template, dpl2[0].Spec.Template)
		})
	}
}

func legacyDeploymentWithConfigMapVolume(staleAnnotation bool) *appv1.Deployment {
	annotations := map[string]string{"cerebro.k8s.webcenter.fr": "true"}
	if staleAnnotation {
		annotations["cerebro.k8s.webcenter.fr/configmap-test-config-cb"] = "stale"
	}
	annotations["cerebro.k8s.webcenter.fr/secret-test-application-cb"] = "secret-sum"

	return &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-cb"},
		Spec: appv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: annotations},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: "test-config-cb"},
								},
							},
						},
						{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					},
				},
			},
		},
	}
}

func TestMigrateLegacyDeploymentOnUpgrade(t *testing.T) {
	cases := []struct {
		name           string
		strategy       cerebrocrd.ConfigUpdateStrategy
		wantStaleAnno  bool
		wantSecretAnno bool
	}{
		{"unset hot reload", "", false, true},
		{"explicit hot reload", cerebrocrd.ConfigUpdateStrategyHotReload, false, true},
		{"explicit restart", cerebrocrd.ConfigUpdateStrategyRestart, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := newCerebroForStrategy(tc.strategy)
			dpl := legacyDeploymentWithConfigMapVolume(true)
			c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(dpl).Build()

			current := &appv1.Deployment{}
			require.NoError(t, c.Get(context.Background(), types.NamespacedName{Namespace: dpl.Namespace, Name: dpl.Name}, current))
			require.NoError(t, migrateLegacyDeployment(context.Background(), c, o, current, logrus.NewEntry(logrus.New())))

			got := &appv1.Deployment{}
			require.NoError(t, c.Get(context.Background(), types.NamespacedName{Namespace: dpl.Namespace, Name: dpl.Name}, got))
			_, hasStale := got.Spec.Template.Annotations["cerebro.k8s.webcenter.fr/configmap-test-config-cb"]
			_, hasSecret := got.Spec.Template.Annotations["cerebro.k8s.webcenter.fr/secret-test-application-cb"]
			assert.Equal(t, tc.wantStaleAnno, hasStale)
			assert.Equal(t, tc.wantSecretAnno, hasSecret)

			var configVolume *corev1.Volume
			for i := range got.Spec.Template.Spec.Volumes {
				if got.Spec.Template.Spec.Volumes[i].Name == "config" {
					configVolume = &got.Spec.Template.Spec.Volumes[i]
				}
			}
			require.NotNil(t, configVolume)
			assert.Nil(t, configVolume.ConfigMap)
			require.NotNil(t, configVolume.Projected)
			require.Len(t, configVolume.Projected.Sources, 1)
			require.NotNil(t, configVolume.Projected.Sources[0].ConfigMap)
			assert.Equal(t, "test-config-cb", configVolume.Projected.Sources[0].ConfigMap.Name)

			names := map[string]bool{}
			for _, v := range got.Spec.Template.Spec.Volumes {
				names[v.Name] = true
			}
			assert.True(t, names["data"], "data volume must survive migration")
			assert.True(t, names["tmp"], "tmp volume must survive migration")
		})
	}
}

func TestCerebroCRDSchemaConfigUpdateStrategy(t *testing.T) {
	raw, err := os.ReadFile("../../../config/crd/bases/cerebro.k8s.webcenter.fr_cerebroes.yaml")
	require.NoError(t, err)

	var crd map[string]any
	require.NoError(t, sigsyaml.Unmarshal(raw, &crd))

	version := crd["spec"].(map[string]any)["versions"].([]any)[0].(map[string]any)
	prop := version["schema"].(map[string]any)["openAPIV3Schema"].(map[string]any)["properties"].(map[string]any)["spec"].(map[string]any)["properties"].(map[string]any)["deployment"].(map[string]any)["properties"].(map[string]any)["configUpdateStrategy"].(map[string]any)

	assert.Equal(t, "HotReload", prop["default"])
	assert.ElementsMatch(t, []any{"Restart", "HotReload"}, prop["enum"])
}
