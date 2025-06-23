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

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type filebeatValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupFilebeatWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr).
			For(&Filebeat{}).
			WithValidator(&filebeatValidator{
				logger: logger,
				client: client,
			}).
			Complete()
	}
}

//+kubebuilder:webhook:path=/validate-beat-k8s-webcenter-fr-v1-filebeat,mutating=false,failurePolicy=fail,sideEffects=None,groups=beat.k8s.webcenter.fr,resources=filebeats,verbs=create;update,versions=v1,name=filebeat.beat.k8s.webcenter.fr,admissionReviewVersions=v1

// Use webhook.CustomValidator for controller-runtime v0.15+ or webhook.Validator for older versions
var _ webhook.CustomValidator = &filebeatValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *filebeatValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	filebeatObj, ok := obj.(*Filebeat)
	if !ok {
		return nil, errors.Errorf("expected a Filebeat object but got %T", obj)
	}

	// Check only one target
	if filebeatObj.Spec.LogstashRef != nil && filebeatObj.Spec.ElasticsearchRef != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), filebeatObj.Spec, "ElasticsearchRef and LogstashRef are mutually exclusive"))
	}

	// Check is set excatly one target
	if filebeatObj.Spec.ElasticsearchRef == nil && filebeatObj.Spec.LogstashRef == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), filebeatObj.Spec, "You need to provide Elasticsearch target or Logstash target"))
	}

	// Check logstash target
	if filebeatObj.Spec.LogstashRef != nil {
		if err := filebeatObj.Spec.LogstashRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check Elasticsearch target
	if filebeatObj.Spec.ElasticsearchRef != nil {
		if err := filebeatObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			filebeatObj.GroupVersionKind().GroupKind(),
			filebeatObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *filebeatValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	filebeatObj, ok := newObj.(*Filebeat)
	if !ok {
		return nil, errors.Errorf("expected a Filebeat object but got %T", newObj)
	}

	// Check only one target
	if filebeatObj.Spec.LogstashRef != nil && filebeatObj.Spec.ElasticsearchRef != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), filebeatObj.Spec, "ElasticsearchRef and LogstashRef are mutually exclusive"))
	}

	// Check is set excatly one target
	if filebeatObj.Spec.ElasticsearchRef == nil && filebeatObj.Spec.LogstashRef == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), filebeatObj.Spec, "You need to provide Elasticsearch target or Logstash target"))
	}

	// Check logstash target
	if filebeatObj.Spec.LogstashRef != nil {
		if err := filebeatObj.Spec.LogstashRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check Opensearch target
	if filebeatObj.Spec.ElasticsearchRef != nil {
		if err := filebeatObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			filebeatObj.GroupVersionKind().GroupKind(),
			filebeatObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *filebeatValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
