package kibana

import (
	"context"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/sirupsen/logrus"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
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
	multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *rbacv1.RoleBinding]
	isOpenshift bool
}

func newRoleBindingReconciler(client client.Client, recorder record.EventRecorder, isOpenshift bool) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *rbacv1.RoleBinding]) {
	return &roleBindingReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*kibanacrd.Kibana, *rbacv1.RoleBinding](
			client,
			RoleBindingPhase,
			RoleBindingCondition,
			recorder,
		),
		isOpenshift: isOpenshift,
	}
}

// Read existing service account
func (r *roleBindingReconciler) Read(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*rbacv1.RoleBinding], res reconcile.Result, err error) {
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
	read.SetExpectedObjects(expectedRoleBindings)

	return read, res, nil
}

// Update permit to handle how to update role binding
// RoleRef is immutable. So if we update it, we need to recreate it
func (r *roleBindingReconciler) Update(ctx context.Context, o *kibanacrd.Kibana, data map[string]any, objects []*rbacv1.RoleBinding, logger *logrus.Entry) (res reconcile.Result, err error) {
	// First, we try to update it
	res, err = r.MultiPhaseStepReconcilerAction.Update(ctx, o, data, objects, logger)
	if err != nil {
		if k8serrors.IsForbidden(err) {
			// Delete
			res, err = r.Delete(ctx, o, data, objects, logger)
			if err != nil {
				return res, errors.Wrap(err, "Error when delete role bindins in gload to recreate it (update)")
			}

			// Create
			res, err = r.Create(ctx, o, data, objects, logger)
			if err != nil {
				return res, errors.Wrap(err, "Error when create role binding after delete it (update)")
			}

			return res, nil
		}

		return res, err
	}

	return res, nil
}
