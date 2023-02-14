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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/codingsince1985/checksum"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	olivere "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
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
	LicenseFinalizer = "license.elasticsearchapi.k8s.webcenter.fr/finalizer"
	LicenseCondition = "License"
)

// LicenseReconciler reconciles a License object
type LicenseReconciler struct {
	Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewLicenseReconciler(client client.Client, scheme *runtime.Scheme) *LicenseReconciler {

	r := &LicenseReconciler{
		Client: client,
		Scheme: scheme,
		name:   "license",
	}

	common.ControllerMetrics.WithLabelValues(r.name).Add(0)

	return r
}

//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=patch;get;create
//+kubebuilder:rbac:groups="elasticsearch.k8s.webcenter.fr",resources=elasticsearches,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the License object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *LicenseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconciler, err := controller.NewStdReconciler(r.Client, LicenseFinalizer, r.reconciler, r.log, r.recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	license := &elasticsearchapicrd.License{}
	data := map[string]any{}

	return reconciler.Reconcile(ctx, req, license, data)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LicenseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticsearchapicrd.License{}).
		Watches(&source.Kind{Type: &core.Secret{}}, handler.EnqueueRequestsFromMapFunc(watchLicenseSecret(r.Client))).
		Complete(r)
}

// watchLicenseSecret permit to update license if secret change
func watchLicenseSecret(c client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {

		reconcileRequests := make([]reconcile.Request, 0)
		listLicenses := &elasticsearchapicrd.LicenseList{}

		fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.secretRef.name=%s", a.GetName()))

		// Get all license linked with secret
		if err := c.List(context.Background(), listLicenses, &client.ListOptions{Namespace: a.GetNamespace(), FieldSelector: fs}); err != nil {
			panic(err)
		}

		for _, l := range listLicenses.Items {
			reconcileRequests = append(reconcileRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: l.Name, Namespace: l.Namespace}})
		}

		return reconcileRequests
	}
}

// Configure permit to init Elasticsearch handler
func (r *LicenseReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (meta any, err error) {
	license := resource.(*elasticsearchapicrd.License)

	// Init condition status if not exist
	if condition.FindStatusCondition(license.Status.Conditions, LicenseCondition) == nil {
		condition.SetStatusCondition(&license.Status.Conditions, metav1.Condition{
			Type:   LicenseCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})
	}

	// Get elasticsearch handler / client
	meta, err = GetElasticsearchHandler(ctx, license, license.Spec.ElasticsearchRef, r.Client, r.log)
	if err != nil {
		r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Unable to init elasticsearch handler: %s", err.Error())
		return nil, err
	}

	return meta, err
}

// Read permit to get current License
func (r *LicenseReconciler) Read(ctx context.Context, resource client.Object, data map[string]any, meta any) (res ctrl.Result, err error) {
	license := resource.(*elasticsearchapicrd.License)

	// esHandler can be empty when Elasticsearch cluster not yet ready or cluster is deleted
	if meta == nil {
		// reschedule if ressource not being on delete phase
		if license.DeletionTimestamp.IsZero() {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		return res, nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)

	// Read license contend from secret if not basic
	if !license.IsBasicLicense() {
		if license.Spec.SecretRef == nil {
			return res, errors.New("You must set the secretRef to get license")
		}
		secret := &core.Secret{}
		secretNS := types.NamespacedName{
			Namespace: license.Namespace,
			Name:      license.Spec.SecretRef.Name,
		}
		if err = r.Get(ctx, secretNS, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				r.log.Warnf("Secret %s not yet exist, try later", license.Spec.SecretRef.Name)
				r.recorder.Eventf(resource, core.EventTypeWarning, "Failed", "Secret %s not yet exist", license.Spec.SecretRef.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return res, errors.Wrapf(err, "Error when get secret %s", license.Spec.SecretRef.Name)
		}
		licenseB, ok := secret.Data["license"]
		if !ok {
			return res, errors.Wrapf(err, "Secret %s must have a license key", license.Spec.SecretRef.Name)
		}
		expectedLicense := &olivere.XPackInfoServiceResponse{}
		if err = json.Unmarshal(licenseB, expectedLicense); err != nil {
			return res, errors.Wrap(err, "License contend is invalid")
		}
		data["expectedLicense"] = &expectedLicense.License
		data["rawLicense"] = string(licenseB)

	}

	// Read the current license from Elasticsearch

	licenseInfo, err := esHandler.LicenseGet()
	if err != nil {
		return res, errors.Wrap(err, "Unable to get current license from Elasticsearch")
	}
	data["currentLicense"] = licenseInfo

	return res, nil
}

// Create add new license or enable basic license
func (r *LicenseReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {

	esHandler := meta.(eshandler.ElasticsearchHandler)
	license := resource.(*elasticsearchapicrd.License)
	var d any

	// Basic license
	if license.IsBasicLicense() {
		if err = esHandler.LicenseEnableBasic(); err != nil {
			return res, errors.Wrap(err, "Error when activate basic license")
		}
		r.log.Info("Successfully enable basic license")
		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "Enable basic license")
		license.Status.LicenseType = "basic"
		license.Status.ExpireAt = ""
		license.Status.LicenseChecksum = ""

	} else {
		// Enterprise or platinium license
		d, err = helper.Get(data, "expectedLicense")
		if err != nil {
			return res, err
		}
		expectedLicense := d.(*olivere.XPackInfoLicense)
		d, err = helper.Get(data, "rawLicense")
		if err != nil {
			return res, err
		}
		rawLicense := d.(string)
		if err = esHandler.LicenseUpdate(rawLicense); err != nil {
			return res, errors.Wrap(err, "Error when add enterprise license on Elasticsearch")
		}
		r.log.Infof("Successfully enable %s license", expectedLicense.Type)
		r.recorder.Eventf(resource, core.EventTypeNormal, "Completed", "Enable %s license", expectedLicense.Type)

		licenseChecksum, err := checksum.SHA256sumReader(strings.NewReader(rawLicense))
		if err != nil {
			return res, errors.Wrap(err, "Error when checksum license")
		}

		license.Status.ExpireAt = time.UnixMilli(int64(expectedLicense.ExpiryMilis)).Format(time.RFC3339)
		license.Status.LicenseChecksum = licenseChecksum
		license.Status.LicenseType = expectedLicense.Type
	}

	return res, nil
}

// Update permit to update current license from Elasticsearch
func (r *LicenseReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (res ctrl.Result, err error) {
	return r.Create(ctx, resource, data, meta)
}

// Delete permit to delete current license from Elasticsearch
func (r *LicenseReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}, meta interface{}) (err error) {
	// Skip, ressource must be deleted and cluster not ready. Maybee cluster is already deleted
	if meta == nil {
		return nil
	}

	esHandler := meta.(eshandler.ElasticsearchHandler)
	license := resource.(*elasticsearchapicrd.License)

	// Not delete License
	// If enterprise license, it must enable basic license instead
	if !license.IsBasicLicense() {
		if err = esHandler.LicenseEnableBasic(); err != nil {
			return errors.Wrap(err, "Error when downgrade to basic license")
		}
		r.log.Info("Successfully downgrade to basic license")
	}

	return nil

}

// Diff permit to check if diff between actual and expected license exist
func (r *LicenseReconciler) Diff(resource client.Object, data map[string]interface{}, meta interface{}) (diff controller.Diff, err error) {
	esHandler := meta.(eshandler.ElasticsearchHandler)
	license := resource.(*elasticsearchapicrd.License)

	var expectedLicense *olivere.XPackInfoLicense
	var d any

	if license.IsBasicLicense() {
		expectedLicense = &olivere.XPackInfoLicense{
			Type: "basic",
		}
	} else {
		d, err = helper.Get(data, "expectedLicense")
		if err != nil {
			return diff, err
		}
		expectedLicense = d.(*olivere.XPackInfoLicense)

	}

	d, err = helper.Get(data, "currentLicense")
	if err != nil {
		return diff, err
	}
	currentLicense := d.(*olivere.XPackInfoLicense)

	diff = controller.Diff{
		NeedCreate: false,
		NeedUpdate: false,
	}
	if currentLicense == nil {
		diff.NeedCreate = true
		diff.Diff = "UID or license type mismatch"
		return diff, nil
	}

	if esHandler.LicenseDiff(currentLicense, expectedLicense) {
		diff.NeedUpdate = true
		return diff, nil
	}

	return
}

// OnError permit to set status condition on the right state and record error
func (r *LicenseReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, meta any, err error) {
	license := resource.(*elasticsearchapicrd.License)

	r.log.Error(err)
	r.recorder.Event(resource, core.EventTypeWarning, "Failed", err.Error())

	condition.SetStatusCondition(&license.Status.Conditions, metav1.Condition{
		Type:    LicenseCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: err.Error(),
	})

	license.Status.Sync = false
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *LicenseReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, meta any, diff controller.Diff) (err error) {
	license := resource.(*elasticsearchapicrd.License)

	license.Status.Sync = true

	if diff.NeedCreate {
		condition.SetStatusCondition(&license.Status.Conditions, metav1.Condition{
			Type:    LicenseCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: fmt.Sprintf("License of type %s successfully created", license.Status.LicenseType),
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "License successfully created")

		return nil
	}

	if diff.NeedUpdate {
		condition.SetStatusCondition(&license.Status.Conditions, metav1.Condition{
			Type:    LicenseCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: fmt.Sprintf("License of type %s successfully updated", license.Status.LicenseType),
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "License successfully updated")

		return nil
	}

	// Update condition status if needed
	if condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, LicenseCondition, metav1.ConditionFalse) {
		condition.SetStatusCondition(&license.Status.Conditions, metav1.Condition{
			Type:    LicenseCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "License already set",
		})

		r.recorder.Event(resource, core.EventTypeNormal, "Completed", "License already set")
	}

	return nil
}
