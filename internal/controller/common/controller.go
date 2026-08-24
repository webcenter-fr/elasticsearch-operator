package common

import (
	"os"
	"reflect"
	"time"

	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// FieldManager is the stable Server-Side Apply field manager name shared by
// every controller of this operator. Changing it orphans every field applied
// under the old name.
const FieldManager = "elasticsearch-operator"

// envtestEnvVar is the explicit, unambiguous env var the envtest suites set to
// signal "no kubelet / no StatefulSet controller" (the generic TEST var is too
// easy to set accidentally and is reserved for the Go test harness).
const envtestEnvVar = "ES_OPERATOR_ENVTEST"

// IsEnvtest reports whether the operator is running under the envtest harness
// (where no kubelet or StatefulSet controller exists and the controller's
// convergence/upgrade gating must fast-forward instead of waiting for pods).
func IsEnvtest() bool {
	return os.Getenv(envtestEnvVar) == "true"
}

// ESClientTimeout bounds every Elasticsearch API request issued through the
// disaster37/elasticsearch v9 (resty) client. The v9 client has no default
// timeout (resty default is 0 = no timeout), so a hung, black-holed or
// unreachable cluster would otherwise block a reconcile worker indefinitely.
// This preserves the 10s dial/response-header timeout the v2
// go-elasticsearch client enforced, with headroom for larger administrative
// operations (license upload, snapshot repository registration, etc.).
const ESClientTimeout = 30 * time.Second

// InjectTypeMeta sets the apiVersion/kind (TypeMeta) on the given objects using
// the provided scheme. Server-Side Apply requires the apiVersion and kind to be
// present in the apply patch; the deprecated client.Apply patch serializes the
// object as-is, so objects built without TypeMeta are rejected by the API server
// ("Incorrect version specified in apply patch"). Objects that already carry a
// non-empty kind are left untouched.
func InjectTypeMeta[T client.Object](scheme *runtime.Scheme, objs ...T) {
	for _, obj := range objs {
		v := reflect.ValueOf(obj)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		if obj.GetObjectKind().GroupVersionKind().Kind != "" {
			continue
		}
		if gvks, _, err := scheme.ObjectKinds(obj); err == nil && len(gvks) > 0 {
			obj.GetObjectKind().SetGroupVersionKind(gvks[0])
		}
	}
}

// DriftSecretForMetadata returns a copy of current with the expected labels and
// annotations applied when they differ from the current metadata, or nil when
// there is no drift. It preserves the current Secret data (certificates) so the
// SSA apply only reconciles metadata. This lets the single-certificate TLS steps
// detect label/annotation-only CR updates without regenerating certificates.
func DriftSecretForMetadata(current *corev1.Secret, labels, annotations map[string]string) *corev1.Secret {
	if current == nil {
		return nil
	}
	if reflect.DeepEqual(current.Labels, labels) && reflect.DeepEqual(current.Annotations, annotations) {
		return nil
	}
	expected := current.DeepCopy()
	expected.Labels = labels
	expected.Annotations = annotations
	if expected.TypeMeta.Kind == "" {
		expected.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}
	}
	return expected
}

func DefaultControllerRateLimiter() workqueue.TypedRateLimiter[reconcile.Request] {
	return workqueue.NewTypedMaxOfRateLimiter[reconcile.Request](
		workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](1*time.Second, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}
