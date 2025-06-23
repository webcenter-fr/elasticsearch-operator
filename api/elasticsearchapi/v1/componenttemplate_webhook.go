/*
Copyright 2024.

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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
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

type componentTemplateValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupComponentTemplateWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr).
			For(&ComponentTemplate{}).
			WithValidator(&componentTemplateValidator{
				logger: logger.WithField("webhook", "componentTemplateValidator"),
				client: client,
			}).
			Complete()
	}
}

//+kubebuilder:webhook:path=/validate-elasticsearchapi-k8s-webcenter-fr-v1-componenttemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=elasticsearchapi.k8s.webcenter.fr,resources=componenttemplates,verbs=create;update,versions=v1,name=componenttemplate.elasticsearchapi.k8s.webcenter.fr,admissionReviewVersions=v1,timeoutSeconds=30

var _ webhook.CustomValidator = &componentTemplateValidator{}

func (r *componentTemplateValidator) validateResourceUnicity(obj *ComponentTemplate) *field.Error {
	// Check if resource already exist with same name on some remote cluster target
	listObjects := &ComponentTemplateList{}
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.externalName=%s,spec.targetCluster=%s", obj.GetExternalName(), obj.Spec.ElasticsearchRef.GetTargetCluster(obj.Namespace)))
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
			return field.Duplicate(field.NewPath("spec").Child("name"), fmt.Sprintf("There are some same resource that already target the same Elasticsearch cluster with the same name: %s", strings.Join(existingResources, ", ")))
		}
	}

	return nil
}

func (r *componentTemplateValidator) validateExplicitTemplateOrRawTemplate(obj *ComponentTemplate) *field.Error {
	if obj.IsRawTemplate() {
		if obj.Spec.Aliases != nil || obj.Spec.Mappings != nil || obj.Spec.Settings != nil {
			return field.Forbidden(field.NewPath("spec").Child("rawTemplate"), "When you set field 'spec.rawTemplate', you can't set field 'spec.settings', 'spec.mappings' and 'spec.aliases'")
		}
	} else {
		if obj.Spec.Aliases == nil && obj.Spec.Mappings == nil && obj.Spec.Settings == nil {
			return field.Required(field.NewPath("spec"), "You need to provide 'spec.rawTemplate' or minimum one of the 'spec.mappings', 'spec.settings' and 'spec.aliases'")
		}
	}

	return nil
}

func (r *componentTemplateValidator) validateRawTemplate(obj *ComponentTemplate) *field.Error {
	if obj.IsRawTemplate() {

		componentTemplate := &olivere.IndicesGetComponentTemplate{}
		if err := json.Unmarshal([]byte(*obj.Spec.RawTemplate), componentTemplate); err != nil {
			return field.Invalid(field.NewPath("spec").Child("rawTemplate"), obj.Spec.RawTemplate, fmt.Sprintf("The JSON is invalid: %s", err.Error()))
		}
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *componentTemplateValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	componentTemplateObj, ok := obj.(*ComponentTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a ComponentTemplate object but got %T", obj)
	}
	r.logger.Debugf("validate create %s/%s", componentTemplateObj.GetNamespace(), componentTemplateObj.GetName())

	if err := r.validateExplicitTemplateOrRawTemplate(componentTemplateObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateRawTemplate(componentTemplateObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := componentTemplateObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(componentTemplateObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			componentTemplateObj.GroupVersionKind().GroupKind(),
			componentTemplateObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *componentTemplateValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	oldO := oldObj.(*ComponentTemplate)

	componentTemplateObj, ok := newObj.(*ComponentTemplate)
	if !ok {
		return nil, fmt.Errorf("expected a ComponentTemplate object but got %T", newObj)
	}
	r.logger.Debugf("validate update %s/%s", componentTemplateObj.Namespace, componentTemplateObj.Name)

	if err := r.validateExplicitTemplateOrRawTemplate(componentTemplateObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateRawTemplate(componentTemplateObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := componentTemplateObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateImmutableName(componentTemplateObj, oldO); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(componentTemplateObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			componentTemplateObj.GroupVersionKind().GroupKind(),
			componentTemplateObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *componentTemplateValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
