package kibana

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RouteCondition shared.ConditionName = "RouteReady"
	RoutePhase     shared.PhaseName     = "Route"
)

type routeReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *routev1.Route]
}

func newRouteReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *routev1.Route]) {
	return &routeReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *routev1.Route](
			client,
			RoutePhase,
			RouteCondition,
			recorder,
		),
	}
}

// Read existing route
func (r *routeReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*routev1.Route], res reconcile.Result, err error) {
	route := &routev1.Route{}
	read = multiphase.NewMultiPhaseRead[*routev1.Route]()
	var secretTlsAPI *corev1.Secret

	// Read current route
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetIngressName(o)}, route); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read route")
		}
		route = nil
	}
	if route != nil {
		read.AddCurrentObject(route)
	}

	// Read APi Crt if needed
	if o.Spec.Tls.IsTlsEnabled() {
		secretTlsAPI = &corev1.Secret{}
		if o.Spec.Tls.IsSelfManagedSecretForTls() {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForTls(o)}, secretTlsAPI); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForTls(o))
				}
				logger.Warnf("Secret %s not yet exist, try again later", GetSecretNameForTls(o))
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
		} else {
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: o.Spec.Tls.CertificateSecretRef.Name}, secretTlsAPI); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", o.Spec.Tls.CertificateSecretRef.Name)
				}
				logger.Warnf("Secret %s not yet exist, try again later", o.Spec.Tls.CertificateSecretRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Generate expected route
	expectedRoutes, err := buildRoutes(o, secretTlsAPI)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate route")
	}
	read.SetExpectedObjects(expectedRoutes)

	return read, res, nil
}
