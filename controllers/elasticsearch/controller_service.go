package elasticsearch

import (
	"context"
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/pkg/errors"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceCondition = "ServiceReady"
	ServicePhase     = "Service"
)

type ServiceReconciler struct {
	common.Reconciler
	client.Client
	Scheme *runtime.Scheme
	name   string
}

func NewServiceReconciler(client client.Client, scheme *runtime.Scheme, reconciler common.Reconciler) controller.K8sPhaseReconciler {
	return &ServiceReconciler{
		Reconciler: reconciler,
		Client:     client,
		Scheme:     scheme,
		name:       "service",
	}
}

// Name return the current phase
func (r *ServiceReconciler) Name() string {
	return r.name
}

// Configure permit to init condition
func (r *ServiceReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	// Init condition status if not exist
	if condition.FindStatusCondition(o.Status.Conditions, ServiceCondition) == nil {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:   ServiceCondition,
			Status: metav1.ConditionFalse,
			Reason: "Initialize",
		})

		o.Status.Phase = ServicePhase
	}

	return res, nil
}

// Read existing services
func (r *ServiceReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	serviceList := &corev1.ServiceList{}

	// Read current node group services
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true,%s/service=true", o.Name, ElasticsearchAnnotationKey, ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, serviceList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read service")
	}
	data["currentServices"] = serviceList.Items

	// Generate expected node group services
	expectedServices, err := BuildServices(o)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate services")
	}
	data["expectedServices"] = expectedServices

	return res, nil
}

// Create will create services
func (r *ServiceReconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "newServices")
	if err != nil {
		return res, err
	}
	expectedServices := d.([]corev1.Service)

	for _, service := range expectedServices {
		if err = r.Client.Create(ctx, &service); err != nil {
			return res, errors.Wrapf(err, "Error when create service %s", service.Name)
		}
	}

	return res, nil
}

// Update will update services
func (r *ServiceReconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "services")
	if err != nil {
		return res, err
	}
	expectedServices := d.([]corev1.Service)

	for _, service := range expectedServices {
		if err = r.Client.Update(ctx, &service); err != nil {
			return res, errors.Wrapf(err, "Error when update service %s", service.Name)
		}
	}

	return res, nil
}

// Delete permit to delete service
func (r *ServiceReconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "oldServices")
	if err != nil {
		return res, err
	}
	oldServices := d.([]corev1.Service)

	for _, service := range oldServices {
		if err = r.Client.Delete(ctx, &service); err != nil {
			return res, errors.Wrapf(err, "Error when delete service %s", service.Name)
		}
	}

	return res, nil
}

// Diff permit to check if services are up to date
func (r *ServiceReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)
	var d any

	d, err = helper.Get(data, "currentServices")
	if err != nil {
		return diff, res, err
	}
	currentServices := d.([]corev1.Service)

	d, err = helper.Get(data, "expectedServices")
	if err != nil {
		return diff, res, err
	}
	expectedServices := d.([]corev1.Service)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	serviceToUpdate := make([]corev1.Service, 0)
	serviceToCreate := make([]corev1.Service, 0)

	for _, expectedService := range expectedServices {
		isFound := false
		for i, currentService := range currentServices {
			// Need compare service
			if currentService.Name == expectedService.Name {
				isFound = true

				patchResult, err := patch.DefaultPatchMaker.Calculate(&currentService, &expectedService, patch.CleanMetadata(), patch.IgnoreStatusFields())
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing service %s", currentService.Name)
				}
				if !patchResult.IsEmpty() {
					updatedService := patchResult.Patched.(*corev1.Service)
					diff.NeedUpdate = true
					diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedService.Name, string(patchResult.Patch)))
					serviceToUpdate = append(serviceToUpdate, *updatedService)
					r.Log.Debugf("Need update service %s", updatedService.Name)
				}

				// Remove items found
				currentServices = helperdiff.DeleteItemFromSlice(currentServices, i).([]corev1.Service)

				break
			}
		}

		if !isFound {
			// Need create services
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Service %s not yet exist\n", expectedService.Name))

			// Set owner
			err = ctrl.SetControllerReference(o, &expectedService, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(&expectedService); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on service %s", expectedService.Name)
			}

			serviceToCreate = append(serviceToCreate, expectedService)

			r.Log.Debugf("Need create service %s", expectedService.Name)
		}
	}

	if len(currentServices) > 0 {
		diff.NeedDelete = true
		for _, service := range currentServices {
			diff.Diff.WriteString(fmt.Sprintf("Need delete service %s\n", service.Name))
		}
	}

	data["newServices"] = serviceToCreate
	data["services"] = serviceToUpdate
	data["oldServices"] = currentServices

	return diff, res, nil
}

// OnError permit to set status condition on the right state and record error
func (r *ServiceReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	r.Log.Error(currentErr)
	r.Recorder.Event(resource, corev1.EventTypeWarning, "Failed", currentErr.Error())

	condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    ServiceCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "Failed",
		Message: currentErr.Error(),
	})

	return res, currentErr

}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *ServiceReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchapi.Elasticsearch)

	if diff.NeedCreate || diff.NeedUpdate || diff.NeedDelete {
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Completed", "Services successfully updated")
	}

	// Update condition status if needed
	if !condition.IsStatusConditionPresentAndEqual(o.Status.Conditions, ServiceCondition, metav1.ConditionTrue) {
		condition.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
			Type:    ServiceCondition,
			Reason:  "Success",
			Status:  metav1.ConditionTrue,
			Message: "Ready",
		})
	}

	return res, nil
}
