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

type metricbeatValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupMetricbeatWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr).
			For(&Metricbeat{}).
			WithValidator(&metricbeatValidator{
				logger: logger,
				client: client,
			}).
			Complete()
	}
}

//+kubebuilder:webhook:path=/validate-beat-k8s-webcenter-fr-v1-metricbeat,mutating=false,failurePolicy=fail,sideEffects=None,groups=beat.k8s.webcenter.fr,resources=metricbeats,verbs=create;update,versions=v1,name=metricbeat.beat.k8s.webcenter.fr,admissionReviewVersions=v1

// Use webhook.CustomValidator for controller-runtime v0.15+ or webhook.Validator for older versions
var _ webhook.CustomValidator = &metricbeatValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *metricbeatValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	metricbeatObj, ok := obj.(*Metricbeat)
	if !ok {
		return nil, errors.Errorf("expected a Metricbeat object but got %T", obj)
	}

	// Check Elasticsearch target
	if err := metricbeatObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			metricbeatObj.GroupVersionKind().GroupKind(),
			metricbeatObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *metricbeatValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList

	metricbeatObj, ok := newObj.(*Metricbeat)
	if !ok {
		return nil, errors.Errorf("expected a Metricbeat object but got %T", newObj)
	}

	// Check Elasticsearch target
	if err := metricbeatObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			metricbeatObj.GroupVersionKind().GroupKind(),
			metricbeatObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *metricbeatValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
