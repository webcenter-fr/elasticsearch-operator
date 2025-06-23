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

type userValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupUserWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr).
			For(&User{}).
			WithValidator(&userValidator{
				logger: logger.WithField("webhook", "userValidator"),
				client: client,
			}).
			Complete()
	}
}

// +kubebuilder:webhook:path=/validate-elasticsearchapi-k8s-webcenter-fr-v1-user,mutating=false,failurePolicy=fail,sideEffects=None,groups=elasticsearchapi.k8s.webcenter.fr,resources=users,verbs=create;update,versions=v1,name=user.elasticsearchapi.k8s.webcenter.fr,admissionReviewVersions=v1

var _ webhook.CustomValidator = &userValidator{}

func (r *userValidator) validateResourceUnicity(obj *User) *field.Error {
	// Check if resource already exist with same name on some remote cluster target
	listObjects := &UserList{}
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

func (r *userValidator) validateRequiredPassword(obj *User) *field.Error {
	if obj.IsAutoGeneratePassword() {
		if obj.Spec.PasswordHash != "" || obj.Spec.SecretRef != nil {
			return field.Forbidden(field.NewPath("spec").Child("autoGeneratePassword"), "When you set 'spec.autoGeneratePassword', you can't set 'spec.passwordHash' or 'spec.secretRef'")
		}
	} else {
		if obj.Spec.PasswordHash == "" && obj.Spec.SecretRef == nil {
			return field.Required(field.NewPath("spec"), "You need to provide one one them 'spec.autoGeneratePassword', 'spec.passwordHash' or 'spec.secretRef'")
		}
		if obj.Spec.PasswordHash != "" && obj.Spec.SecretRef != nil {
			return field.Forbidden(field.NewPath("spec"), "You can set only one of them 'spec.passwordHash' or 'spec.secretRef'")
		}
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *userValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	userObj, ok := obj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected a User object but got %T", obj)
	}
	r.logger.Debugf("validate create %s/%s", userObj.GetNamespace(), userObj.GetName())

	if err := userObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(userObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateRequiredPassword(userObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			userObj.GroupVersionKind().GroupKind(),
			userObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *userValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	oldO := oldObj.(*User)

	userObj, ok := newObj.(*User)
	if !ok {
		return nil, fmt.Errorf("expected a User object but got %T", newObj)
	}
	r.logger.Debugf("validate update %s/%s", userObj.Namespace, userObj.Name)

	if err := userObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateImmutableName(userObj, oldO); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(userObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateRequiredPassword(userObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			userObj.GroupVersionKind().GroupKind(),
			userObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *userValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
