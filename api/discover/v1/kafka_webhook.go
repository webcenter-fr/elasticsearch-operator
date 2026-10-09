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

type kafkaValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupKafkaWebhookWithManager will setup the manager to manage the webhooks
func SetupKafkaWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr, &Kafka{}).
			WithValidator(&kafkaValidator{
				logger: logger,
				client: client,
			}).
			Complete()
	}
}

//+kubebuilder:webhook:path=/validate-discover-k8s-webcenter-fr-v1-kafka,mutating=false,failurePolicy=fail,sideEffects=None,groups=discover.k8s.webcenter.fr,resources=kafkas,verbs=create;update,versions=v1,name=kafka.discover.k8s.webcenter.fr,admissionReviewVersions=v1

var _ admission.Validator[*Kafka] = &kafkaValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *kafkaValidator) ValidateCreate(ctx context.Context, obj *Kafka) (admission.Warnings, error) {
	var allErrs field.ErrorList

	if err := obj.Spec.KafkaRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			obj.GroupVersionKind().GroupKind(),
			obj.Name, allErrs,
		)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *kafkaValidator) ValidateUpdate(ctx context.Context, oldObj *Kafka, newObj *Kafka) (admission.Warnings, error) {
	var allErrs field.ErrorList

	if err := newObj.Spec.KafkaRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			newObj.GroupVersionKind().GroupKind(),
			newObj.Name, allErrs,
		)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *kafkaValidator) ValidateDelete(ctx context.Context, obj *Kafka) (admission.Warnings, error) {
	return nil, nil
}
