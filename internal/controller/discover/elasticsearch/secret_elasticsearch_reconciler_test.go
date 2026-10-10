package elasticsearch

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newElasticsearchTestClient(t *testing.T, objects ...runtime.Object) *fake.ClientBuilder {
	sch := runtime.NewScheme()
	if err := scheme.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}
	if err := discovercrd.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}
	if err := elasticsearchcrd.AddToScheme(sch); err != nil {
		t.Fatal(err)
	}

	return fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(objects...)
}

func TestSecretElasticsearchReconcilerReadExternal(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	userSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "external-user-secret",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("admin-password"),
		},
	}
	customCaSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "custom-ca-secret",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("custom-ca"),
		},
	}
	o := &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ExternalElasticsearchRef: &shared.ElasticsearchExternalRef{
					Addresses: []string{
						"https://elasticsearch1.example.com:9200",
						"https://elasticsearch2.example.com:9200",
					},
				},
				SecretRef: &corev1.LocalObjectReference{
					Name: "external-user-secret",
				},
				ElasticsearchCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-secret",
				},
			},
		},
	}

	c := newElasticsearchTestClient(t, o, userSecret, customCaSecret).Build()
	r := newSecretElasticsearchReconciler(c, record.NewFakeRecorder(10))

	read, res, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.Zero(t, res.RequeueAfter)
	assert.Empty(t, read.GetCurrentObjects())
	assert.Len(t, read.GetExpectedObjects(), 2)
	assert.Equal(t, "test-env", read.GetExpectedObjects()[0].Name)
	assert.Equal(t, []byte("https://elasticsearch1.example.com:9200,https://elasticsearch2.example.com:9200"), read.GetExpectedObjects()[0].Data["ELASTICSEARCH_HOSTS_TEST"])
	assert.Equal(t, []byte("admin"), read.GetExpectedObjects()[0].Data["ELASTICSEARCH_USERNAME_TEST"])
	assert.Equal(t, []byte("admin-password"), read.GetExpectedObjects()[0].Data["ELASTICSEARCH_PASSWORD_TEST"])
	assert.Equal(t, "test-file", read.GetExpectedObjects()[1].Name)
	assert.Equal(t, []byte("custom-ca"), read.GetExpectedObjects()[1].Data["ca.crt"])

	// Check diff must create the two secrets
	diff, _, err := r.Diff(ctx, o, read, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.True(t, diff.NeedCreate())
	assert.Len(t, diff.GetObjectsToCreate(), 2)
	assert.False(t, diff.NeedDelete())
}

func TestSecretElasticsearchReconcilerReadWhenManagedClusterNotReady(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	o := &discovercrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: discovercrd.ElasticsearchSpec{
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
					Name: "managed-elasticsearch",
				},
			},
		},
	}

	c := newElasticsearchTestClient(t, o).Build()
	r := newSecretElasticsearchReconciler(c, record.NewFakeRecorder(10))

	_, res, err := r.Read(ctx, o, map[string]any{}, logger)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, res.RequeueAfter)
}
