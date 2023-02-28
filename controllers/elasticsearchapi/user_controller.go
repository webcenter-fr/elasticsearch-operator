/*
Copyright 2022.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package elasticsearchapi

import (
	"context"
	"fmt"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	olivere "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	UserFinalizer = "user.elasticsearchapi.k8s.webcenter.fr/finalizer"
	UserCondition = "User"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewUserReconciler(client client.Client, scheme *runtime.Scheme) *UserReconciler {

	r := &UserReconciler{
		Client: client,
		Scheme: scheme,
		name:   "user",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=users/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconciler, err := controller.NewStdReconciler(r.Client, UserFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	user := &elasticsearchapicrd.User{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, user, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.User{}).
		Watches(&source.Kind{Type: &core.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchUserSecret(r.Client))).
		Complete(r)
}

// watchUserSecret permit to update user if secret change
func watchUserSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {

		reconcileRequests := make([]reconcile.Request, 0)
		listUsers := &elasticsearchapicrd.UserList{}

		fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.secretRef.name=%s", a.GetName()))

		// Get all users linked with secret
		if err := c.List(context.Background(), listUsers, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}

		for _, u := range listUsers.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: u.Name, Namespace: u.Namespace}})
		}

		return reconcileRequests
	}
}

// Configure permit to init Elasticsearch handler
// It also permit to init condition
func (r *UserReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	o := resource.(*elasticsearchapicrd.User)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, UserCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   UserCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	if condition.FindStatusCondition(o.Status.Conditions, common.ReadyCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, o, o.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, err
}

// Read permit to get current user
// It also read password from secret
func (r *UserReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapicrd.User)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if o.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read user from Elasticsearch
	currentUser, err := esHandler.UserGet(o.GetUsername())
	if err != nil {
		return res, errors.Wrap(err, "Unable to get user from Elasticsearch")
	}

	// Read password from secret if needed
	if o.Spec.SecretRef != nil {
		secret := &core.Secret{}
		secretNS := types.NamespacedName{
			Namespace: o.Namespace,
			Name:      o.Spec.SecretRef.Name,
		}
		if err = r.Get(ctx, secretNS, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				r.log.Warnf("Secret %s not yet exist, try later", o.Spec.SecretRef.Name)
				r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Secret %s not yet exist", o.Spec.SecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return res, errors.Wrapf(err, "Error when get secret %s", o.Spec.SecretRef.Name)
		}
		passwordB, ok := secret.Data[o.Spec.SecretRef.Key]
		if !ok {
			return res, errors.Wrapf(err, "Secret %s must have a %s key", o.Spec.SecretRef.Name, o.Spec.SecretRef.Key)
		}
		data["password"] = string(passwordB)
	}
	data["current"] = currentUser

	// Generate expected
	expectedUser, err := BuildUser(o)
	if err != nil {
		return res, errors.Wrap(err, "Unable to generate user")
	}
	data["expected"] = expectedUser

	return res, nil
}

// Create add new user
func (r *UserReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	user := resource.(*elasticsearchapicrd.User)
	var d any
	var passwordHash string

	// Create user on Elasticsearch
	expectedUser, err := BuildUser(user)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert user")
	}

	if user.Spec.SecretRef != nil {
		d, err = helper.Get(data, "password")
		if err != nil {
			return res, err
		}
		expectedUser.Password = d.(string)
		passwordHash, err = localhelper.HashPassword(expectedUser.Password)
		if err != nil {
			return res, errors.Wrap(err, "Error when hash password")
		}
	} else {
		passwordHash = user.Spec.PasswordHash
	}

	if err = esHandler.UserCreate(user.GetUsername(), expectedUser); err != nil {
		return res, errors.Wrap(err, "Error when create user")
	}

	user.Status.PasswordHash = passwordHash

	return res, nil
}

// Update permit to update user from Elasticsearch
func (r *UserReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	user := resource.(*elasticsearchapicrd.User)
	var d any
	var passwordHash string
	isUpdatePasssword := false

	// Create user on Elasticsearch
	expectedUser, err := BuildUser(user)
	if err != nil {
		return res, errors.Wrap(err, "Error when convert user")
	}

	if user.Spec.SecretRef != nil {
		d, err = helper.Get(data, "password")
		if err != nil {
			return res, err
		}
		password := d.(string)
		if !localhelper.CheckPasswordHash(password, user.Status.PasswordHash) {
			expectedUser.Password = password
			expectedUser.PasswordHash = ""
			passwordHash, err = localhelper.HashPassword(expectedUser.Password)
			if err != nil {
				return res, errors.Wrap(err, "Error when hash password")
			}
			isUpdatePasssword = true
		}
	} else {
		if user.Spec.PasswordHash == user.Status.PasswordHash {
			expectedUser.PasswordHash = ""
			passwordHash = user.Spec.PasswordHash
			isUpdatePasssword = true
		}
	}

	if err = esHandler.UserUpdate(user.GetUsername(), expectedUser, user.IsProtected()); err != nil {
		return res, errors.Wrap(err, "Error when update user")
	}

	if isUpdatePasssword {
		user.Status.PasswordHash = passwordHash
	}

	return res, nil
}

// Delete permit to delete user from Elasticsearch
func (r *UserReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	user := resource.(*elasticsearchapicrd.User)

	// Don't delete user if it protected
	if user.IsProtected() {
		return nil
	}

	if err = esHandler.UserDelete(user.GetUsername()); err != nil {
		return errors.Wrap(err, "Error when delete user")
	}

	return nil

}

// Diff permit to check if diff between actual and expected user exist
func (r *UserReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	user := resource.(*elasticsearchapicrd.User)
	var (
		d           any
		currentUser *olivere.XPackSecurityPutUserRequest
	)

	d, err = helper.Get(data, "current")
	if err != nil {
		return diff, err
	}
	currentUserTmp := d.(*olivere.XPackSecurityUser)

	d, err = helper.Get(data, "expected")
	if err != nil {
		return diff, err
	}
	expectedUser := d.(*olivere.XPackSecurityPutUserRequest)

	var originalUser *olivere.XPackSecurityPutUserRequest
	if user.Status.OriginalObject != "" {
		originalUser = &olivere.XPackSecurityPutUserRequest{}
		if err = localhelper.UnZipBase64Decode(user.Status.OriginalObject, originalUser); err != nil {
			return diff, err
		}
	}

	if currentUserTmp == nil {
		diff.NeedCreate = true
		diff.Diff = "user not exist"

		if err = localhelper.SetLastOriginal(user, expectedUser); err != nil {
			return diff, err
		}

		return diff, nil
	}

	// Is is protected user, only manage the password
	if user.IsProtected() {
		currentUser = &olivere.XPackSecurityPutUserRequest{
			Enabled:  currentUserTmp.Enabled,
			Password: user.Status.PasswordHash,
		}
	} else {
		currentUser = &olivere.XPackSecurityPutUserRequest{
			Enabled:  currentUserTmp.Enabled,
			Email:    currentUserTmp.Email,
			FullName: currentUserTmp.Fullname,
			Metadata: currentUserTmp.Metadata,
			Roles:    currentUserTmp.Roles,
			Password: user.Status.PasswordHash,
		}
	}

	if user.Spec.SecretRef != nil {
		d, err = helper.Get(data, "password")
		if err != nil {
			return diff, err
		}
		password := d.(string)

		// Check if password change, bcrypt generate hash different each time
		if !localhelper.CheckPasswordHash(password, user.Status.PasswordHash) {
			expectedUser.Password = "XXX"
		} else {
			expectedUser.Password = user.Status.PasswordHash
		}
	}

	differ, err := esHandler.UserDiff(currentUser, expectedUser, originalUser)
	if err != nil {
		return diff, err
	}

	if !differ.IsEmpty() {
		diff.NeedUpdate = true
		diff.Diff = string(differ.Patch)

		if err = localhelper.SetLastOriginal(user, expectedUser); err != nil {
			return diff, err
		}

		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *UserReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	user := resource.(*elasticsearchapicrd.User)
	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
		Type:    UserCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	condition.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
		Type:   common.ReadyCondition,
		Status: metav1.ConditionFalse,
		Reason: "Error",
	})

	user.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *UserReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	user := resource.(*elasticsearchapicrd.User)

	user.Status.Sync = true

	if condition.IsStatusConditionPresentAndEqual(user.Status.Conditions, common.ReadyCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:   common.ReadyCondition,
			Reason: "Available",
			Status: metav1.ConditionTrue,
		})
	}

	if diff.NeedCreate {
		condition.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    UserCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "User successfully created",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "User successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    UserCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "User successfully updated",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "User successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(user.Status.Conditions, UserCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    UserCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "User already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "User already set")
	}

	return nil
}
