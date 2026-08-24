package metricbeat

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RoleBindingCondition shared.ConditionName = "RoleBindingReady"
	RoleBindingPhase     shared.PhaseName     = "RoleBinding"
)

type roleBindingReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Metricbeat, *rbacv1.RoleBinding]
	isOpenshift bool
}

func newRoleBindingReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*beatcrd.Metricbeat, *rbacv1.RoleBinding]) {
	return &roleBindingReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*beatcrd.Metricbeat, *rbacv1.RoleBinding](
			client,
			RoleBindingPhase,
			RoleBindingCondition,
			recorder,
			common.FieldManager,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing service account
func (r *roleBindingReconciler) Read(ctx context.Context, o *beatcrd.Metricbeat, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*rbacv1.RoleBinding], res reconcile.Result, err error) {
	roleBinding := &rbacv1.RoleBinding{}
	read = multiphase.NewMultiPhaseRead[*rbacv1.RoleBinding]()

	// Read current service account
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetServiceAccountName(o)}, roleBinding); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read role binding")
		}
		roleBinding = nil
	}
	if roleBinding != nil {
		read.AddCurrentObject(roleBinding)
	}

	// Generate expected service account
	expectedRoleBindings, err := buildRoleBindings(o, r.isOpenshift)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate role bindings")
	}
	common.InjectTypeMeta(r.Client().Scheme(), expectedRoleBindings...)
	read.SetExpectedObjects(expectedRoleBindings)

	return read, res, nil
}

// Apply permit to handle how to apply role bindings
// RoleRef is immutable. So if it changed, we need to delete and recreate the object.
func (r *roleBindingReconciler) Apply(ctx context.Context, o *beatcrd.Metricbeat, data map[string]any, objects []*rbacv1.RoleBinding, logger *logrus.Entry) (res reconcile.Result, err error) {
	// First, we try to apply it
	res, err = r.MultiPhaseStepReconcilerAction.Apply(ctx, o, data, objects, logger)
	if err != nil {
		if k8serrors.IsForbidden(err) || k8serrors.IsInvalid(err) {
			// RoleRef is immutable: delete then recreate
			res, err = r.Delete(ctx, o, data, objects, logger)
			if err != nil {
				return res, errors.Wrap(err, "Error when delete role bindings in order to recreate it (apply)")
			}

			res, err = r.MultiPhaseStepReconcilerAction.Apply(ctx, o, data, objects, logger)
			if err != nil {
				return res, errors.Wrap(err, "Error when recreate role binding after delete it (apply)")
			}

			return res, nil
		}

		return res, err
	}

	return res, nil
}
