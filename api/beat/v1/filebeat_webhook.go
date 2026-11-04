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
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type filebeatValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupFilebeatWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr, &Filebeat{}).
			WithValidator(&filebeatValidator{
				logger: logger,
				client: client,
			}).
			Complete()
	}
}

//+kubebuilder:webhook:path=/validate-beat-k8s-webcenter-fr-v1-filebeat,mutating=false,failurePolicy=fail,sideEffects=None,groups=beat.k8s.webcenter.fr,resources=filebeats,verbs=create;update,versions=v1,name=filebeat.beat.k8s.webcenter.fr,admissionReviewVersions=v1

var _ admission.Validator[*Filebeat] = &filebeatValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *filebeatValidator) ValidateCreate(ctx context.Context, obj *Filebeat) (admission.Warnings, error) {
	var allErrs field.ErrorList

	// Check only one target pattern: either legacy refs or discover refs
	hasLegacyRef := obj.Spec.LogstashRef != nil || obj.Spec.ElasticsearchRef != nil
	hasDiscoverRef := len(obj.Spec.DiscoverRef) > 0

	if hasLegacyRef && hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), obj.Spec, "You can't use elasticsearchRef/logstashRef and discoverRef at the same time"))
	}

	// Check only one legacy target
	if obj.Spec.LogstashRef != nil && obj.Spec.ElasticsearchRef != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), obj.Spec, "ElasticsearchRef and LogstashRef are mutually exclusive"))
	}

	// Check is set at least one target
	if !hasLegacyRef && !hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), obj.Spec, "You need to provide Elasticsearch target, Logstash target or Discover target"))
	}

	// Check each discover ref references exactly one discover
	if hasDiscoverRef {
		allErrs = append(allErrs, discovercrd.ValidateDiscoverRefs(obj.Spec.DiscoverRef, field.NewPath("spec").Child("discoverRef"))...)
	}

	// Check logstash target
	if obj.Spec.LogstashRef != nil {
		if err := obj.Spec.LogstashRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check Elasticsearch target
	if obj.Spec.ElasticsearchRef != nil {
		if err := obj.Spec.ElasticsearchRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check discover output name is set in discoverRef if DiscoverOutputName is set
	if obj.Spec.DiscoverOutputName != nil && *obj.Spec.DiscoverOutputName != "" {
		found := false
		for _, dr := range obj.Spec.DiscoverRef {
			if dr == nil {
				continue
			}
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
func (r *filebeatValidator) ValidateUpdate(ctx context.Context, oldObj *Filebeat, newObj *Filebeat) (admission.Warnings, error) {
	var allErrs field.ErrorList

	// Check only one target pattern: either legacy refs or discover refs
	hasLegacyRef := newObj.Spec.LogstashRef != nil || newObj.Spec.ElasticsearchRef != nil
	hasDiscoverRef := len(newObj.Spec.DiscoverRef) > 0

	if hasLegacyRef && hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), newObj.Spec, "You can't use elasticsearchRef/logstashRef and discoverRef at the same time"))
	}

	// Check only one legacy target
	if newObj.Spec.LogstashRef != nil && newObj.Spec.ElasticsearchRef != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), newObj.Spec, "ElasticsearchRef and LogstashRef are mutually exclusive"))
	}

	// Check is set at least one target
	if !hasLegacyRef && !hasDiscoverRef {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), newObj.Spec, "You need to provide Elasticsearch target, Logstash target or Discover target"))
	}

	// Check each discover ref references exactly one discover
	if hasDiscoverRef {
		allErrs = append(allErrs, discovercrd.ValidateDiscoverRefs(newObj.Spec.DiscoverRef, field.NewPath("spec").Child("discoverRef"))...)
	}

	// Check logstash target
	if newObj.Spec.LogstashRef != nil {
		if err := newObj.Spec.LogstashRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check Elasticsearch target
	if newObj.Spec.ElasticsearchRef != nil {
		if err := newObj.Spec.ElasticsearchRef.ValidateField(); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	// Check discover output name is set in discoverRef if DiscoverOutputName is set
	if newObj.Spec.DiscoverOutputName != nil && *newObj.Spec.DiscoverOutputName != "" {
		found := false
		for _, dr := range newObj.Spec.DiscoverRef {
			if dr == nil {
				continue
			}
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
func (r *filebeatValidator) ValidateDelete(ctx context.Context, obj *Filebeat) (admission.Warnings, error) {
	return nil, nil
}
