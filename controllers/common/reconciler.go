package common

import (
	"context"
	"fmt"
	"reflect"

	"emperror.dev/errors"
	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/sirupsen/logrus"
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	Recorder record.EventRecorder
	Log      *logrus.Entry
	client.Client
	Scheme *runtime.Scheme
	Name   string
}

// GetName return the current name of the reconciler
func (r *Reconciler) GetName() string {
	return r.Name
}

// Create will create resources on Kubernetes
// data map must have key `listToCreate` of type []client.Object
func (r *Reconciler) Create(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "listToCreate")
	if err != nil {
		return res, err
	}
	listToCreate := d.([]client.Object)

	for _, o := range listToCreate {
		if err = r.Client.Create(ctx, o); err != nil {
			return res, errors.Wrapf(err, "Error when create object of type %s with name %s/%s", o.GetObjectKind().GroupVersionKind().Kind, o.GetNamespace(), o.GetName())
		}
	}

	return res, nil
}

// Update will update resources on Kubernetes
// data map must have key `listToUpdate` of type []client.Object
func (r *Reconciler) Update(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "listToUpdate")
	if err != nil {
		return res, err
	}
	listToUpdate := d.([]client.Object)

	for _, o := range listToUpdate {
		if err = r.Client.Update(ctx, o); err != nil {
			return res, errors.Wrapf(err, "Error when update object of type %s with name %s/%s", o.GetObjectKind().GroupVersionKind().Kind, o.GetNamespace(), o.GetName())
		}
	}

	return res, nil
}

// Delete permit to delete resources on Kubernetes
// data map must have key `listToDelete` of type []client.Object
func (r *Reconciler) Delete(ctx context.Context, resource client.Object, data map[string]interface{}) (res ctrl.Result, err error) {

	var d any

	d, err = helper.Get(data, "listToDelete")
	if err != nil {
		return res, err
	}
	listToDelete := d.([]client.Object)

	for _, o := range listToDelete {
		if err = r.Client.Delete(ctx, o); err != nil {
			return res, errors.Wrapf(err, "Error when delete object of type %s with name %s/%s", o.GetObjectKind().GroupVersionKind().Kind, o.GetNamespace(), o.GetName())
		}
	}

	return res, nil
}

// StdDiff is the standard diff when we need to diff only one resource
// Data map need to have key `currentObject` and `expectedObject`
func (r *Reconciler) StdDiff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "currentObject")
	if err != nil {
		return diff, res, err
	}
	currentObject := d.(client.Object)

	d, err = helper.Get(data, "expectedObject")
	if err != nil {
		return diff, res, err
	}
	expectedObject := d.(client.Object)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	toUpdate := make([]client.Object, 0)
	toCreate := make([]client.Object, 0)
	toDelete := make([]client.Object, 0)

	if currentObject == nil || reflect.ValueOf(currentObject).IsNil() {
		if expectedObject != nil && !reflect.ValueOf(expectedObject).IsNil() {
			// Create new object
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Create %s\n", expectedObject.GetName()))

			// Set owner
			err = ctrl.SetControllerReference(resource, expectedObject, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedObject); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on %s", expectedObject.GetName())
			}

			toCreate = append(toCreate, expectedObject)
		}
	} else {

		if expectedObject != nil && !reflect.ValueOf(expectedObject).IsNil() {
			// Check if need to update object
			patchResult, err := patch.DefaultPatchMaker.Calculate(currentObject, expectedObject, patch.CleanMetadata(), patch.IgnoreStatusFields())
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when diffing %s", currentObject.GetName())
			}
			if !patchResult.IsEmpty() {
				updatedObject := patchResult.Patched.(client.Object)
				diff.NeedUpdate = true
				diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedObject.GetName(), string(patchResult.Patch)))
				toUpdate = append(toUpdate, updatedObject)
				r.Log.Debugf("Need update %s", updatedObject.GetName())
			}
		} else {
			// Need delete object
			diff.NeedDelete = true
			diff.Diff.WriteString(fmt.Sprintf("Delete %s\n", currentObject.GetName()))
			toDelete = append(toDelete, currentObject)
		}

	}

	data["listToCreate"] = toCreate
	data["listToUpdate"] = toUpdate
	data["listToDelete"] = toDelete

	return diff, res, nil
}

// StdListDiff is the standard diff when we need to diff list of resources
// Data map need to have key `currentObjects` and `expectedObjects`
func (r *Reconciler) StdListDiff(ctx context.Context, resource client.Object, data map[string]interface{}, ignoreDiff ...patch.CalculateOption) (diff controller.K8sDiff, res ctrl.Result, err error) {
	var d any

	d, err = helper.Get(data, "currentObjects")
	if err != nil {
		return diff, res, err
	}
	currentObjects := helperdiff.ToSliceOfObject(d)

	d, err = helper.Get(data, "expectedObjects")
	if err != nil {
		return diff, res, err
	}
	expectedObjects := helperdiff.ToSliceOfObject(d)

	tmpCurrentObjects := make([]client.Object, len(currentObjects))
	copy(tmpCurrentObjects, currentObjects)

	diff = controller.K8sDiff{
		NeedCreate: false,
		NeedUpdate: false,
		NeedDelete: false,
	}

	toUpdate := make([]client.Object, 0)
	toCreate := make([]client.Object, 0)

	patchOptions := []patch.CalculateOption{
		patch.CleanMetadata(),
		patch.IgnoreStatusFields(),
	}
	patchOptions = append(patchOptions, ignoreDiff...)

	for _, expectedObject := range expectedObjects {
		isFound := false
		for i, currentObject := range tmpCurrentObjects {
			// Need compare same object
			if currentObject.GetName() == expectedObject.GetName() {
				isFound = true

				// Copy TypeMeta to work with some ignore rules like IgnorePDBSelector()
				mustInjectTypeMeta(currentObject, expectedObject)
				patchResult, err := patch.DefaultPatchMaker.Calculate(currentObject, expectedObject, patchOptions...)
				if err != nil {
					return diff, res, errors.Wrapf(err, "Error when diffing %s", currentObject.GetName())
				}
				if !patchResult.IsEmpty() {
					updatedObject := patchResult.Patched.(client.Object)
					diff.NeedUpdate = true
					diff.Diff.WriteString(fmt.Sprintf("diff %s: %s\n", updatedObject.GetName(), string(patchResult.Patch)))
					toUpdate = append(toUpdate, updatedObject)
					r.Log.Debugf("Need Update %s", updatedObject.GetName())
				}

				// Remove items found
				tmpCurrentObjects = helperdiff.DeleteItemFromSlice(tmpCurrentObjects, i).([]client.Object)

				break
			}
		}

		if !isFound {
			// Need create object
			diff.NeedCreate = true
			diff.Diff.WriteString(fmt.Sprintf("Create %s\n", expectedObject.GetName()))

			// Set owner
			err = ctrl.SetControllerReference(resource, expectedObject, r.Scheme)
			if err != nil {
				return diff, res, errors.Wrapf(err, "Error when set owner reference")
			}

			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(expectedObject); err != nil {
				return diff, res, errors.Wrapf(err, "Error when set diff annotation on %s", expectedObject.GetName())
			}

			toCreate = append(toCreate, expectedObject)

			r.Log.Debugf("Need create %s", expectedObject.GetName())
		}
	}

	if len(tmpCurrentObjects) > 0 {
		diff.NeedDelete = true
		for _, object := range tmpCurrentObjects {
			diff.Diff.WriteString(fmt.Sprintf("Delete %s\n", object.GetName()))
		}
	}

	data["listToCreate"] = toCreate
	data["listToUpdate"] = toUpdate
	data["listToDelete"] = tmpCurrentObjects

	return diff, res, nil
}

func mustInjectTypeMeta(src, dst client.Object) {
	var (
		rt reflect.Type
	)

	rt = reflect.TypeOf(src)
	if rt.Kind() != reflect.Ptr {
		panic("Resource must be pointer")
	}
	rt = reflect.TypeOf(dst)
	if rt.Kind() != reflect.Ptr {
		panic("Resource must be pointer")
	}

	rvSrc := reflect.ValueOf(src).Elem()
	omSrc := rvSrc.FieldByName("TypeMeta")
	if !omSrc.IsValid() {
		panic("src must have field TypeMeta")
	}
	rvDst := reflect.ValueOf(dst).Elem()
	omDst := rvDst.FieldByName("TypeMeta")
	if !omDst.IsValid() {
		panic("dst must have field TypeMeta")
	}

	omDst.Set(omSrc)
}
