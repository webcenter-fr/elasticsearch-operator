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

package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type licenseValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupLicenseWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr).
			For(&License{}).
			WithValidator(&licenseValidator{
				logger: logger.WithField("webhook", "licenseValidator"),
				client: client,
			}).
			Complete()
	}
}

// +kubebuilder:webhook:path=/validate-elasticsearchapi-k8s-webcenter-fr-v1-license,mutating=false,failurePolicy=fail,sideEffects=None,groups=elasticsearchapi.k8s.webcenter.fr,resources=licenses,verbs=create;update,versions=v1,name=license.elasticsearchapi.k8s.webcenter.fr,admissionReviewVersions=v1

var _ webhook.CustomValidator = &licenseValidator{}

func (r *licenseValidator) validateBasicOrLicense(obj *License) *field.Error {
	if obj.IsBasicLicense() {
		if obj.Spec.SecretRef != nil {
			return field.Forbidden(field.NewPath("spec").Child("secretRef"), "When you set field 'spec.isBasic' to true, you can't set field 'spec.secretRef'")
		}
	} else {
		if obj.Spec.SecretRef == nil {
			return field.Required(field.NewPath("spec").Child("secretRef"), "You need to provide the secret ref that store license")
		}
	}

	return nil
}

func (r *licenseValidator) validateResourceUnicity(obj *License) *field.Error {
	// Check if resource already exist with same name on some remote cluster target
	listObjects := &LicenseList{}
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.targetCluster=%s", obj.Spec.ElasticsearchRef.GetTargetCluster(obj.Namespace)))
	if err := r.client.List(context.Background(), listObjects, &client.ListOptions{FieldSelector: fs}); err != nil {
		panic(err)
	}
	if len(listObjects.Items) > 0 {
		isError := false
		existingResources := make([]string, 0, len(listObjects.Items))
		for _, ag := range listObjects.Items {
			// exclude themself
			if ag.UID != obj.UID {
				existingResources = append(existingResources, fmt.Sprintf("'%s/%s'", ag.Namespace, ag.Name))
				isError = true
			}
		}
		if isError {
			return field.Duplicate(field.NewPath("spec").Child("name"), fmt.Sprintf("You can set only one license per Elasticsearch cluster: %s", strings.Join(existingResources, ", ")))
		}
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *licenseValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	licenseObj, ok := obj.(*License)
	if !ok {
		return nil, fmt.Errorf("expected a License object but got %T", obj)
	}
	r.logger.Debugf("validate create %s/%s", licenseObj.GetNamespace(), licenseObj.GetName())

	if err := licenseObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateBasicOrLicense(licenseObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(licenseObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			licenseObj.GroupVersionKind().GroupKind(),
			licenseObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *licenseValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	oldO := oldObj.(*License)

	licenseObj, ok := newObj.(*License)
	if !ok {
		return nil, fmt.Errorf("expected a License object but got %T", newObj)
	}
	r.logger.Debugf("validate update %s/%s", licenseObj.Namespace, licenseObj.Name)

	if err := licenseObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateImmutableName(licenseObj, oldO); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateBasicOrLicense(licenseObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(licenseObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			licenseObj.GroupVersionKind().GroupKind(),
			licenseObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *licenseValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
