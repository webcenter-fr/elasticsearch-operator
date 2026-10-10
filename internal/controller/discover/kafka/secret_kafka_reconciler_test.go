package kafka

import (
	"context"
	"testing"
	"time"

	strimzicrd "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testUserKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCqC7iHvcmK4O7d
hCvKkQ7qfKQSFnSMPhzREfPaRllUI0CpI7MgCUzC6BDjqjTzL+Ai3lfyW+kB0cAE
FJktBh36Le+ObFoP2wTYfSpEIMiD9lwYm6k1Lb2wc8vZlq4zlxkqXtJKs9KNCBCd
Rg/u1iwanxvvibmYCCLvHu+Syi9icUVjc5Owv5Wg+OCh3EI4Wvo9eUb7oE2WMxCh
UukhGf3a6viq1g4uQPh8NrbjB2USIl1rNSINOeg5da7+VZLwE7diRRmoOd6geT6o
sfCOlEKnC9JGXg+B/a0AukwABBBYDNsGpN7pNZpn0zjU9T07i9+14ePeyzGH+7yy
Bvj+WU4JAgMBAAECggEABxCzmxg8noOYCSYrt5/tUbraClJah1UxV1J6olAX+BH6
5swv7h3Uzahcw0jYKF6N3oUEfHJrLQMtCj5c2u0NI87yzCUeXNhJzEDvF6OREUKU
MwWMs0PyRLma97B2Dnixt/v2mtn73pY+MnqtwMuvS2/e9kXvqyxzXyIW7C9wZpOi
54y1SUwrJtce/UMtluStnZtgl8OGO8mDDH14edXojlb9cOy2MdkHEWqmcFI8UJ1I
kVh36cpp+k/TkIAYeE81IME/XYJ5uOvnIfoqVFU2Av9IfYduGQbpc1yJR6Wf34rN
AVbuciaJfA5THC4b5s7OmggqmiadltpcLAafSwRMvQKBgQDY45+PJaQjfNRAtDkW
bo0Qo+qtSLF5hl3CC0fc+hM3mhDM935raP2aoP7kvuZhjsZkQRM4YWQKpPnlBwpn
YHguSSAcAFjBTzNvMagfRH7TGxvjhzFqjxJhsxh6WMjoDF4YPufYGUWHnoIltpPZ
EkHvvIsS1HAdQaQkzeFPOncpfwKBgQDItaP99j2H24nkzfBe1MdgsE8fR47XIpDh
V3mcb2GOGomG/wJam/hZ9cGorwSn3RYakwx0CREFp95r08nPS5JgBIUn5BAxllBt
y7eGyJNhzQ+p2inz+L9ALxQsdBizC2hjD5ItKYjhX5Gli4KSDEQR7Zvzg4TwmXSM
Gmma8eT8dwKBgQCHvgap97v//et1x5YHJQ+iv4RsCZgR3/ekL509f0IxxXoIXkm/
/cjjUynUjmnv2wTt0BCwc7vCMbi5D6wCQU1WYqv2+nySF/iv+hsn3p2lkEznHUDf
WUX+4bFLOXAcU0k80XoXNNHgbGM2KUvlStj7qzw7f7vfg6qw4i3GgWf9DwKBgDjQ
8nNZ2WE0DHzwrzKpGSeNwVghcZYE+u5PTOWzwfHwIS5N0eTEcjFFGGUf5jl+pFsx
dT0DtAXd12C+u0orImx72xXHwa3H10y9dl55OP9ehSiX0Xh22ezZZuSOmM9WU153
cfHG6DCkVR5/t56QSlSo5pSwjlltl0hx0aNDBodRAoGAP2X1Uza3iq0tngKWFIeO
77AxiS6VUb9tjQtQpFF1hmVMMDBLySb15D23M0SzcSzKrtlKxoDSn1FzzETWdSNt
i+x83aUnRlsg3OGRByNVMbj8mJI7m3z/BhN0sUY4ABYE6e/8PLamBnnNe54dhyLs
jymHpXH28vD3JQuliElFEB0=
-----END PRIVATE KEY-----`

const testUserCrt = `-----BEGIN CERTIFICATE-----
MIIEKzCCAhOgAwIBAgIUVVzGc7Xc5MY1IikS1w08rtP/rywwDQYJKoZIhvcNAQEN
BQAwLTETMBEGA1UECgwKaW8uc3RyaW16aTEWMBQGA1UEAwwNY2xpZW50cy1jYSB2
MDAeFw0yNTA5MTcxNjExNTZaFw0zNTA5MTUxNjExNTZaMBAxDjAMBgNVBAMMBWFk
bWluMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqgu4h73JiuDu3YQr
ypEO6nykEhZ0jD4c0RHz2kZZVCNAqSOzIAlMwugQ46o08y/gIt5X8lvpAdHABBSZ
LQYd+i3vjmxaD9sE2H0qRCDIg/ZcGJupNS29sHPL2ZauM5cZKl7SSrPSjQgQnUYP
7tYsGp8b74m5mAgi7x7vksovYnFFY3OTsL+VoPjgodxCOFr6PXlG+6BNljMQoVLp
IRn92ur4qtYOLkD4fDa24wdlEiJdazUiDTnoOXWu/lWS8BO3YkUZqDneoHk+qLHw
jpRCpwvSRl4Pgf2tALpMAAQQWAzbBqTe6TWaZ9M41PU9O4vfteHj3ssxh/u8sgb4
/llOCQIDAQABo2AwXjAdBgNVHQ4EFgQUX/lsJNN0QdyvAP7xJYffv80tumQwDAYD
VR0TAQH/BAIwADAOBgNVHQ8BAf8EBAMCBaAwHwYDVR0jBBgwFoAU9xTZqBCn/CmW
aR8tVq7HIGxQpWQwDQYJKoZIhvcNAQENBQADggIBACiDVABQrm4KBlr6AXi1vvIm
9Yjo8sHLukfCCkIr0WIgH4jwKVPBOBYmAhVVml3D2SpnbhVVl0AzUfmk56UTTD5D
aXGeO7d+n9JWzsMC1lmbwce7d1jOhtFAbqFHKr1eF3sTXGkd2TfcJW28slAGk6Ae
2CRi7YWMKs9H6V4xlIe8cMkSmTvB3+RiyDa7LKk/w9vtMShyQr1WtW0AWX+JjuFn
ruWfSny4Nz+sixAqYjEFoddvjJWjRkdsgFWUdvnftIw9qmr8nrOISOnvMEfrtJ5c
xM9Ti53sFvO8bKKubDdJZZHIeSaLNmF1YCo3fb/oKzrpCIseTSCvVqeugn2B/4Mq
G/YtHB6XIQp9qWmBjiJTJ8bkyRz3t87XpKdfr/Y2Ryh1ABwfoSQgnkNsEDyufVIe
UmRSIftNQ1wAWVYN+U74/pqkq0ZBM6pelMEgaGqV7JhNtwY50XuFHoBFGKzIf1WY
AlDRJf5YBbM4SkWGIVXsSHWTkFJq6avkMLdeHEJZH7e5j16tNOI5ROwNALCwMmhz
uu9SSXvmLX73L0Kc8U5tCHd02bL4jBUrC2M0h1yVRfHH7t9a7s38lAsEOCMmEmhR
f4j8bexXdBM/6h6sPlV5nejz2fPb5lKoLS0tCg02n0WC6jNQF/yAdNFM0kzBxspj
S/PdFt4dZmYbx/dqWCV+
-----END CERTIFICATE-----`

func newKafkaTestClient(t *testing.T, objects ...runtime.Object) *fake.ClientBuilder {
	sch := runtime.NewScheme()
	if err := scheme.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}
	if err := discovercrd.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}
	if err := strimzicrd.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}

	return fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(objects...)
}

func TestSecretKafkaReconcilerReadExternal(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	userSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "external-user-secret",
		},
		Data: map[string][]byte{
			"user.key": []byte(testUserKey),
			"user.crt": []byte(testUserCrt),
		},
	}
	o := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.KafkaSpec{
			KafkaRef: discovercrd.KafkaRef{
				ExternalKafkaRef: &discovercrd.KafkaExternalRef{
					Addresses: []string{
						"kafka1.example.com:9092",
						"kafka2.example.com:9092",
					},
					UserSecretRef: &corev1.LocalObjectReference{
						Name: "external-user-secret",
					},
				},
			},
		},
	}

	c := newKafkaTestClient(t, o, userSecret).Build()
	r := newSecretKafkaReconciler(c, record.NewFakeRecorder(10), false)

	read, res, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.Zero(t, res.RequeueAfter)
	assert.Empty(t, read.GetCurrentObjects())
	assert.Len(t, read.GetExpectedObjects(), 2)
	assert.Equal(t, "test-env", read.GetExpectedObjects()[0].Name)
	assert.Equal(t, []byte("kafka1.example.com:9092,kafka2.example.com:9092"), read.GetExpectedObjects()[0].Data["KAFKA_BOOTSTRAP_SERVERS_TEST"])
	assert.NotEmpty(t, read.GetExpectedObjects()[0].Data["KAFKA_USER_PASSWORD_TEST"])
	assert.Equal(t, "test-file", read.GetExpectedObjects()[1].Name)
	assert.Equal(t, []byte(testUserKey), read.GetExpectedObjects()[1].Data["user.key"])
	assert.Equal(t, []byte(testUserCrt), read.GetExpectedObjects()[1].Data["user.crt"])
	assert.NotEmpty(t, read.GetExpectedObjects()[1].Data["user.p12"])

	// Check diff must create the two secrets
	diff, _, err := r.Diff(ctx, o, read, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.True(t, diff.NeedCreate())
	assert.Len(t, diff.GetObjectsToCreate(), 2)
	assert.False(t, diff.NeedDelete())
}

func TestSecretKafkaReconcilerReadWhenManagedWithoutStrimzi(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	o := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.KafkaSpec{
			KafkaRef: discovercrd.KafkaRef{
				ManagedKafkaRef: &discovercrd.KafkaManagedRef{
					Name: "managed-kafka",
				},
			},
		},
	}

	c := newKafkaTestClient(t, o).Build()
	r := newSecretKafkaReconciler(c, record.NewFakeRecorder(10), false)

	_, _, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.Error(t, err)
}

func TestSecretKafkaReconcilerReadWhenManagedClusterNotReady(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	o := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.KafkaSpec{
			KafkaRef: discovercrd.KafkaRef{
				ManagedKafkaRef: &discovercrd.KafkaManagedRef{
					Name: "managed-kafka",
				},
			},
		},
	}

	c := newKafkaTestClient(t, o).Build()
	r := newSecretKafkaReconciler(c, record.NewFakeRecorder(10), true)

	_, res, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, res.RequeueAfter)
}

func TestSecretKafkaReconcilerReadWhenManagedClusterStatusNil(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	o := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.KafkaSpec{
			KafkaRef: discovercrd.KafkaRef{
				ManagedKafkaRef: &discovercrd.KafkaManagedRef{
					Name: "managed-kafka",
				},
			},
		},
	}
	// Kafka cluster exists but has no status yet (just created)
	kafkaCluster := &strimzicrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "managed-kafka",
		},
		Spec: &strimzicrd.KafkaSpec{
			Kafka: strimzicrd.KafkaSpecKafka{
				Listeners: []strimzicrd.KafkaSpecKafkaListenersElem{
					{
						Name: "plain",
						Port: 9092,
						Type: strimzicrd.KafkaSpecKafkaListenersElemTypeInternal,
					},
				},
			},
			ClusterCa: &strimzicrd.KafkaSpecClusterCa{
				GenerateCertificateAuthority: ptr.To(false),
			},
		},
	}

	c := newKafkaTestClient(t, o, kafkaCluster).Build()
	r := newSecretKafkaReconciler(c, record.NewFakeRecorder(10), true)

	// Must requeue instead of panicking on nil Status
	_, res, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, res.RequeueAfter)
}

func TestSecretKafkaReconcilerReadWhenManagedUserSecretNotReady(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	o := &discovercrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.KafkaSpec{
			KafkaRef: discovercrd.KafkaRef{
				ManagedKafkaRef: &discovercrd.KafkaManagedRef{
					Name: "managed-kafka",
					UserRef: &corev1.LocalObjectReference{
						Name: "my-user",
					},
				},
			},
		},
	}
	kafkaCluster := &strimzicrd.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "managed-kafka",
		},
		Spec: &strimzicrd.KafkaSpec{
			Kafka: strimzicrd.KafkaSpecKafka{
				Listeners: []strimzicrd.KafkaSpecKafkaListenersElem{
					{
						Name: "plain",
						Port: 9092,
						Type: strimzicrd.KafkaSpecKafkaListenersElemTypeInternal,
					},
				},
			},
			ClusterCa: &strimzicrd.KafkaSpecClusterCa{
				GenerateCertificateAuthority: ptr.To(false),
			},
		},
		Status: &strimzicrd.KafkaStatus{
			Listeners: []strimzicrd.KafkaStatusListenersElem{
				{
					Name:             ptr.To("plain"),
					BootstrapServers: ptr.To("managed-kafka:9092"),
				},
			},
		},
	}
	// Kafka user exists but its status has no secret yet
	kafkaUser := &strimzicrd.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-user",
		},
		Status: &strimzicrd.KafkaUserStatus{},
	}

	c := newKafkaTestClient(t, o, kafkaCluster, kafkaUser).Build()
	r := newSecretKafkaReconciler(c, record.NewFakeRecorder(10), true)

	// Must requeue instead of panicking on nil Status.Secret
	_, res, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, res.RequeueAfter)
}
