package elasticsearch

import (
	"context"
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SystemUserCondition = "SystemUserReady"
	SystemUserPhase     = "systemUser"
)

type SystemUserReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewSystemUserReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &SystemUserReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "systemUser",
	}
}

// Name return the current phase
func (r *SystemUserReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *SystemUserReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, SystemUserCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   SystemUserCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = SystemUserPhase
	}

	return res, nil
}

// Read existing ingress
func (r *SystemUserReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	userList := &elasticsearchapicrd.UserList{}
	s := &corev1.Secret{}

	// Read current system users
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, userList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read system users")
	}
	data["currentUsers"] = userList.Items

	// Read secret that store credentials
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, s); err != nil {
		return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
	}

	// Generate expected users
	expectedUsers, err := BuildUserSystem(o, s)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate system users")
	}
	data["expectedUsers"] = expectedUsers

	return res, nil
}

// Create will create ingress
func (r *SystemUserReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newUsers")
	if err != nil {
		return res, err
	}
	expectedUsers := d.([]elasticsearchapicrd.User)

	for _, u := range expectedUsers {
		if err = r.Client.Create(ctx, &u); err != nil {
			return res, errors.Wrapf(err, "Error when create user %s", u.Name)
		}
	}

	return res, nil
}

// Update will update ingress
func (r *SystemUserReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "users")
	if err != nil {
		return res, err
	}
	expectedUsers := d.([]elasticsearchapicrd.User)

	for _, u := range expectedUsers {
		if err = r.Client.Update(ctx, &u); err != nil {
			return res, errors.Wrapf(err, "Error when update user %s", u.Name)
		}
	}

	return res, nil
}

// Delete permit to delete ingress
func (r *SystemUserReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldUsers")
	if err != nil {
		return res, err
	}
	oldUsers := d.([]elasticsearchapicrd.User)

	for _, u := range oldUsers {
		if err = r.Client.Delete(ctx, &u); err != nil {
			return res, errors.Wrapf(err, "Error when delete user %s", u.Name)
		}
	}

	return res, nil
}

// Diff permit to check if users are up to date
func (r *SystemUserReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentUsers")
	if err != nil {
		return diff, res, err
	}
	currentUsers := d.([]elasticsearchapicrd.User)

	d, err = helper.Get(data, "expectedUsers")
	if err != nil {
		return diff, res, err
	}
	expectedUsers := d.([]elasticsearchapicrd.User)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	userToUpdate := make([]elasticsearchapicrd.User, 0)
	userToCreate := make([]elasticsearchapicrd.User, 0)

	for _, expectedUser := range expectedUsers {
		isFound := false
		for i, currentUser := range currentUsers {
			// Need compare pdb
			if currentUser.Name == expectedUser.Name {
				isFound = true

				patchResult, err := patch.DefaultPatchMaker.Calculate(&currentUser, &expectedUser, patch.CleanMetadata(), patch.IgnoreStatusFields())
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing user %s", currentUser.Name)
				}
				if !patchResult.IsEmpty() {
					updatedUser := patchResult.Patched.(*elasticsearchapicrd.User)
					diff.NeedUpdate = true
					diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedUser.Name, string(patchResult.Patch)))
					userToUpdate = append(userToUpdate, *updatedUser)
					r.Log.Debugf("Need update user %s", updatedUser.Name)
				}

				// Remove items found
				currentUsers = helperdiff.DeleteItemFromSlice(currentUsers, i).([]elasticsearchapicrd.User)

				break
			}
		}

		if !isFound {
			// Need create pdbs
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("User %s not yet exist\n", expectedUser.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, &expectedUser, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(&expectedUser); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on user %s", expectedUser.Name)
			}

			userToCreate = append(userToCreate, expectedUser)

			r.Log.Debugf("Need create user %s", expectedUser.Name)
		}
	}

	if len(currentUsers) > 0 {
		diff.NeedDelete = true
		for _, u := range currentUsers {
			diff.Diff.WriteString(fmt.Sprintf("Need delete user %s\n", u.Name))
		}
	}

	data["newUsers"] = userToCreate
	data["users"] = userToUpdate
	data["oldUsers"] = currentUsers

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *SystemUserReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    SystemUserCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *SystemUserReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "System users successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, SystemUserCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    SystemUserCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
