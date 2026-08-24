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

	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type logstashPipelineValidator struct {
	logger *logrus.Entry
	client client.Client
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupLogstashPipelineWebhookWithManager(logger *logrus.Entry) controller.WebhookRegister {
	return func(mgr ctrl.Manager, client client.Client) error {
		return ctrl.NewWebhookManagedBy(mgr, &LogstashPipeline{}).
			WithValidator(&logstashPipelineValidator{
				logger: logger.WithField("webhook", "logstashPipelineValidator"),
				client: client,
			}).
			Complete()
	}
}

// +kubebuilder:webhook:path=/validate-kibanaapi-k8s-webcenter-fr-v1-logstashpipeline,mutating=false,failurePolicy=fail,sideEffects=None,groups=kibanaapi.k8s.webcenter.fr,resources=logstashpipelines,verbs=create;update,versions=v1,name=logstashpipeline.kibanaapi.k8s.webcenter.fr,admissionReviewVersions=v1

var _ admission.Validator[*LogstashPipeline] = &logstashPipelineValidator{}

func (r *logstashPipelineValidator) validateResourceUnicity(obj *LogstashPipeline) *field.Error {
	// Check if resource already exist with same name on some remote cluster target
	listObjects := &LogstashPipelineList{}
	fs := fields.ParseSelectorOrDie(fmt.Sprintf("spec.externalName=%s,spec.targetCluster=%s", obj.GetExternalName(), obj.Spec.KibanaRef.GetTargetCluster(obj.Namespace)))
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

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *logstashPipelineValidator) ValidateCreate(ctx context.Context, obj *LogstashPipeline) (admission.Warnings, error) {
	var allErrs field.ErrorList

	r.logger.Debugf("validate create %s/%s", obj.GetNamespace(), obj.GetName())

	if err := obj.Spec.KibanaRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(obj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			obj.GroupVersionKind().GroupKind(),
			obj.Name, allErrs)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *logstashPipelineValidator) ValidateUpdate(ctx context.Context, oldObj *LogstashPipeline, newObj *LogstashPipeline) (admission.Warnings, error) {
	var allErrs field.ErrorList

	r.logger.Debugf("validate update %s/%s", newObj.Namespace, newObj.Name)

	if err := newObj.Spec.KibanaRef.ValidateField(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateImmutableName(newObj, oldObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateResourceUnicity(newObj); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(
			newObj.GroupVersionKind().GroupKind(),
			newObj.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *logstashPipelineValidator) ValidateDelete(ctx context.Context, obj *LogstashPipeline) (admission.Warnings, error) {
	return nil, nil
}
