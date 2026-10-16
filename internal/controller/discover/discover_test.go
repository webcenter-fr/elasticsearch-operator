package discover

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/disaster37/k8sbuilder"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIsDiscoverSecret(t *testing.T) {
	// When discover annotation is true
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey: "true",
			},
		},
	}
	assert.True(t, IsDiscoverSecret(secret))

	// When discover annotation is false
	secret.Annotations[discovercrd.DiscoverAnnotationKey] = "false"
	assert.False(t, IsDiscoverSecret(secret))

	// When discover annotation is empty
	secret.Annotations[discovercrd.DiscoverAnnotationKey] = ""
	assert.False(t, IsDiscoverSecret(secret))

	// When annotations are nil
	secret.Annotations = nil
	assert.False(t, IsDiscoverSecret(secret))

	// When annotations does not contain discover
	secret.Annotations = map[string]string{
		"other": "value",
	}
	assert.False(t, IsDiscoverSecret(secret))
}

func TestIsDiscoverSecretEnv(t *testing.T) {
	// When discover env secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey:                         "true",
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "env",
			},
		},
	}
	assert.True(t, IsDiscoverSecretEnv(secret))

	// When discover file secret
	secret.Annotations[fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey)] = "file"
	assert.False(t, IsDiscoverSecretEnv(secret))

	// When not a discover secret
	delete(secret.Annotations, discovercrd.DiscoverAnnotationKey)
	assert.False(t, IsDiscoverSecretEnv(secret))

	// When annotations are nil
	secret.Annotations = nil
	assert.False(t, IsDiscoverSecretEnv(secret))
}

func TestIsDiscoverSecretFile(t *testing.T) {
	// When discover file secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey:                         "true",
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "file",
			},
		},
	}
	assert.True(t, IsDiscoverSecretFile(secret))

	// When discover env secret
	secret.Annotations[fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey)] = "env"
	assert.False(t, IsDiscoverSecretFile(secret))

	// When not a discover secret
	delete(secret.Annotations, discovercrd.DiscoverAnnotationKey)
	assert.False(t, IsDiscoverSecretFile(secret))

	// When annotations are nil
	secret.Annotations = nil
	assert.False(t, IsDiscoverSecretFile(secret))
}

func TestGetDiscoverMountPathFromAnnotations(t *testing.T) {
	// When mount path is set
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "my-kafka",
			},
		},
	}
	assert.Equal(t, "my-kafka", GetDiscoverMountPathFromAnnotations(secret))

	// When mount path is empty
	secret.Annotations[fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey)] = ""
	assert.Equal(t, "", GetDiscoverMountPathFromAnnotations(secret))

	// When annotations are nil
	secret.Annotations = nil
	assert.Equal(t, "", GetDiscoverMountPathFromAnnotations(secret))

	// When annotation does not exist
	secret.Annotations = map[string]string{
		discovercrd.DiscoverAnnotationKey: "true",
	}
	assert.Equal(t, "", GetDiscoverMountPathFromAnnotations(secret))
}

func TestComputeDiscoverPod(t *testing.T) {
	envSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-env",
			Namespace: "default",
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey:                         "true",
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "env",
			},
		},
	}
	fileSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-file",
			Namespace: "default",
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey:                              "true",
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "file",
				fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test",
			},
		},
	}
	otherSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other",
			Namespace: "default",
		},
	}

	ptb := k8sbuilder.NewPodTemplateBuilder()
	cb := k8sbuilder.NewContainerBuilder()
	ComputeDiscoverPod(ptb, cb, []*corev1.Secret{envSecret, fileSecret, otherSecret}, "/usr/share/filebeat")

	// Check volume
	assert.Len(t, ptb.PodTemplate().Spec.Volumes, 1)
	assert.Equal(t, "test-file", ptb.PodTemplate().Spec.Volumes[0].Name)
	assert.NotNil(t, ptb.PodTemplate().Spec.Volumes[0].Secret)
	assert.Equal(t, "test-file", ptb.PodTemplate().Spec.Volumes[0].Secret.SecretName)

	// Check volume mount
	assert.Len(t, cb.Container().VolumeMounts, 1)
	assert.Equal(t, "test-file", cb.Container().VolumeMounts[0].Name)
	assert.Equal(t, "/usr/share/filebeat/discover/test", cb.Container().VolumeMounts[0].MountPath)

	// Check env from
	assert.Len(t, cb.Container().EnvFrom, 1)
	assert.NotNil(t, cb.Container().EnvFrom[0].SecretRef)
	assert.Equal(t, "test-env", cb.Container().EnvFrom[0].SecretRef.Name)
}

func TestReadDiscoversSecrets(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())
	o := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "dummy",
		},
	}

	buildClient := func(objects ...runtime.Object) *fake.ClientBuilder {
		sch := runtime.NewScheme()
		if err := scheme.AddToScheme(sch); err != nil {
			panic(err)
		}
		if err := discovercrd.AddToScheme(sch); err != nil {
			panic(err)
		}
		builder := fake.NewClientBuilder().WithScheme(sch)
		for _, object := range objects {
			builder = builder.WithRuntimeObjects(object)
		}

		return builder
	}

	// When discover is ready and secrets exist
	discover := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: discovercrd.KafkaStatus{
			DiscoverStatus: discovercrd.DiscoverStatus{
				SecretEnvRef:  ptr.To("test-env"),
				SecretFileRef: ptr.To("test-file"),
			},
		},
	}
	envSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-env",
		},
	}
	fileSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-file",
		},
	}
	c := buildClient(discover, envSecret, fileSecret).Build()
	secrets, res, err := ReadDiscoversSecrets(ctx, c, logger, o, []*discovercrd.DiscoverRef{
		{Kafka: &corev1.LocalObjectReference{Name: "test"}},
	})
	assert.NoError(t, err)
	assert.Nil(t, res)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "test-env", secrets[0].Name)
	assert.Equal(t, "test-file", secrets[1].Name)

	// When discover does not exist
	c = buildClient().Build()
	secrets, res, err = ReadDiscoversSecrets(ctx, c, logger, o, []*discovercrd.DiscoverRef{
		{Kafka: &corev1.LocalObjectReference{Name: "test"}},
	})
	assert.NoError(t, err)
	assert.Nil(t, secrets)
	assert.NotNil(t, res)
	assert.NotZero(t, res.RequeueAfter)

	// When discover status is not ready
	discover = &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
	}
	c = buildClient(discover).Build()
	secrets, res, err = ReadDiscoversSecrets(ctx, c, logger, o, []*discovercrd.DiscoverRef{
		{Kafka: &corev1.LocalObjectReference{Name: "test"}},
	})
	assert.NoError(t, err)
	assert.Nil(t, secrets)
	assert.NotNil(t, res)
	assert.Equal(t, 30*time.Second, res.RequeueAfter)

	// When env secret does not exist
	discover = &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: discovercrd.KafkaStatus{
			DiscoverStatus: discovercrd.DiscoverStatus{
				SecretEnvRef:  ptr.To("test-env"),
				SecretFileRef: ptr.To("test-file"),
			},
		},
	}
	c = buildClient(discover, fileSecret).Build()
	secrets, res, err = ReadDiscoversSecrets(ctx, c, logger, o, []*discovercrd.DiscoverRef{
		{Kafka: &corev1.LocalObjectReference{Name: "test"}},
	})
	assert.NoError(t, err)
	assert.Nil(t, secrets)
	assert.NotNil(t, res)
	assert.NotZero(t, res.RequeueAfter)

	// When file secret does not exist
	c = buildClient(discover, envSecret).Build()
	secrets, res, err = ReadDiscoversSecrets(ctx, c, logger, o, []*discovercrd.DiscoverRef{
		{Kafka: &corev1.LocalObjectReference{Name: "test"}},
	})
	assert.NoError(t, err)
	assert.Nil(t, secrets)
	assert.NotNil(t, res)
	assert.NotZero(t, res.RequeueAfter)

	// When a discover ref is nil
	c = buildClient().Build()
	secrets, res, err = ReadDiscoversSecrets(ctx, c, logger, o, []*discovercrd.DiscoverRef{nil})
	assert.NoError(t, err)
	assert.Empty(t, secrets)
	assert.Nil(t, res)
}

func TestReadDiscoverOutputSecrets(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())
	o := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "dummy",
		},
	}

	sch := runtime.NewScheme()
	if err := scheme.AddToScheme(sch); err != nil {
		panic(err)
	}
	if err := discovercrd.AddToScheme(sch); err != nil {
		panic(err)
	}

	newClient := func(objects ...runtime.Object) client.Client {
		builder := fake.NewClientBuilder().WithScheme(sch)
		for _, object := range objects {
			builder = builder.WithRuntimeObjects(object)
		}
		return builder.Build()
	}

	envSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-kafka-env",
			Labels: map[string]string{
				"discoverName":                 "test-kafka",
				discovercrd.KafkaAnnotationKey: "true",
			},
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey:                         "true",
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "env",
			},
		},
	}
	fileSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-kafka-file",
			Labels: map[string]string{
				"discoverName":                 "test-kafka",
				discovercrd.KafkaAnnotationKey: "true",
			},
			Annotations: map[string]string{
				discovercrd.DiscoverAnnotationKey:                         "true",
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "file",
			},
		},
	}

	// When discover output secrets exist
	c := newClient(envSecret, fileSecret)
	discoverType, secretEnv, secretFile, res, err := ReadDiscoverOutputSecrets(ctx, c, logger, o, &discovercrd.DiscoverRef{
		Kafka: &corev1.LocalObjectReference{Name: "test-kafka"},
	})
	assert.NoError(t, err)
	assert.Nil(t, res)
	assert.Equal(t, discovercrd.DiscoverTypeKafka, discoverType)
	assert.NotNil(t, secretEnv)
	assert.Equal(t, "test-kafka-env", secretEnv.Name)
	assert.NotNil(t, secretFile)
	assert.Equal(t, "test-kafka-file", secretFile.Name)

	// When discover output secrets do not exist
	c = newClient()
	discoverType, secretEnv, secretFile, res, err = ReadDiscoverOutputSecrets(ctx, c, logger, o, &discovercrd.DiscoverRef{
		Kafka: &corev1.LocalObjectReference{Name: "test-kafka"},
	})
	assert.NoError(t, err)
	assert.Equal(t, discovercrd.DiscoverTypeKafka, discoverType)
	assert.Nil(t, secretEnv)
	assert.Nil(t, secretFile)
	assert.NotNil(t, res)
	assert.Equal(t, 30*time.Second, res.RequeueAfter)

	// When no discover is referenced
	c = newClient(envSecret, fileSecret)
	discoverType, secretEnv, secretFile, res, err = ReadDiscoverOutputSecrets(ctx, c, logger, o, &discovercrd.DiscoverRef{})
	assert.NoError(t, err)
	assert.Equal(t, discovercrd.DiscoverType(""), discoverType)
	assert.Nil(t, secretEnv)
	assert.Nil(t, secretFile)
	assert.Nil(t, res)
}
