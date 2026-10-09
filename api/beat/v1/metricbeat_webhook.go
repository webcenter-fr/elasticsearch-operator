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

	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type metricbeatValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupMetricbeatWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr, &Metricbeat{}).
			WithValidator(&metricbeatValidator{
				logger: logger,
				client: client,
			}).
			Complete()
	}
}

//+kubebuilder:webhook:path=/validate-beat-k8s-webcenter-fr-v1-metricbeat,mutating=false,failurePolicy=fail,sideEffects=None,groups=beat.k8s.webcenter.fr,resources=metricbeats,verbs=create;update,versions=v1,name=metricbeat.beat.k8s.webcenter.fr,admissionReviewVersions=v1

var _ admission.Validator[*Metricbeat] = &metricbeatValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *metricbeatValidator) ValidateCreate(ctx context.Context, obj *Metricbeat) (admission.Warnings, error) {
	var allErrs field.ErrorList

	// Check only one target pattern: either legacy ref or discover refs
	hasLegacyRef := obj.Spec.ElasticsearchRef.IsManaged() || obj.Spec.ElasticsearchRef.IsExternal()
	hasDiscoverRef := len(obj.Spec.DiscoverRef) > 0

	if hasLegacyRef && hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), obj.Spec, "You can't use elasticsearchRef and discoverRef at the same time"))
	}

	// Check is set at least one target
	if !hasLegacyRef && !hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), obj.Spec, "You need to provide Elasticsearch target or Discover target"))
	}

	// Check Elasticsearch target
	if hasLegacyRef {
		if err := obj.Spec.ElasticsearchRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check discover output name is set in discoverRef if DiscoverOutputName is set
	if obj.Spec.DiscoverOutputName != nil && *obj.Spec.DiscoverOutputName != "" {
		found := false
		for _, dr := range obj.Spec.DiscoverRef {
			if dr.GetName() == *obj.Spec.DiscoverOutputName {
				found = true
				break
			}
		}
		if !found {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("discoverOutputName"), *obj.Spec.DiscoverOutputName, "DiscoverOutputName must reference a name in discoverRef"))
		}
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			obj.GroupVersionKind().GroupKind(),
			obj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *metricbeatValidator) ValidateUpdate(ctx context.Context, oldObj *Metricbeat, newObj *Metricbeat) (admission.Warnings, error) {
	var allErrs field.ErrorList

	// Check only one target pattern: either legacy ref or discover refs
	hasLegacyRef := newObj.Spec.ElasticsearchRef.IsManaged() || newObj.Spec.ElasticsearchRef.IsExternal()
	hasDiscoverRef := len(newObj.Spec.DiscoverRef) > 0

	if hasLegacyRef && hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), newObj.Spec, "You can't use elasticsearchRef and discoverRef at the same time"))
	}

	// Check is set at least one target
	if !hasLegacyRef && !hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), newObj.Spec, "You need to provide Elasticsearch target or Discover target"))
	}

	// Check Elasticsearch target
	if hasLegacyRef {
		if err := newObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check discover output name is set in discoverRef if DiscoverOutputName is set
	if newObj.Spec.DiscoverOutputName != nil && *newObj.Spec.DiscoverOutputName != "" {
		found := false
		for _, dr := range newObj.Spec.DiscoverRef {
			if dr.GetName() == *newObj.Spec.DiscoverOutputName {
				found = true
				break
			}
		}
		if !found {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("discoverOutputName"), *newObj.Spec.DiscoverOutputName, "DiscoverOutputName must reference a name in discoverRef"))
		}
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			newObj.GroupVersionKind().GroupKind(),
			newObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *metricbeatValidator) ValidateDelete(ctx context.Context, obj *Metricbeat) (admission.Warnings, error) {
	return nil, nil
}
