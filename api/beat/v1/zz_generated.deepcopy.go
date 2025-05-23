//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Filebeat) DeepCopyInto(out *Filebeat) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Filebeat.
func (in *Filebeat) DeepCopy() *Filebeat {
	if in == nil {
		return nil
	}
	out := new(Filebeat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Filebeat) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatDeploymentSpec) DeepCopyInto(out *FilebeatDeploymentSpec) {
	*out = *in
	in.Deployment.DeepCopyInto(&out.Deployment)
	if in.AntiAffinity != nil {
		in, out := &in.AntiAffinity, &out.AntiAffinity
		*out = new(shared.DeploymentAntiAffinitySpec)
		**out = **in
	}
	if in.PodDisruptionBudgetSpec != nil {
		in, out := &in.PodDisruptionBudgetSpec, &out.PodDisruptionBudgetSpec
		*out = new(policyv1.PodDisruptionBudgetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.InitContainerResources != nil {
		in, out := &in.InitContainerResources, &out.InitContainerResources
		*out = new(corev1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.AdditionalVolumes != nil {
		in, out := &in.AdditionalVolumes, &out.AdditionalVolumes
		*out = make([]shared.DeploymentVolumeSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Persistence != nil {
		in, out := &in.Persistence, &out.Persistence
		*out = new(shared.DeploymentPersistenceSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]corev1.ContainerPort, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatDeploymentSpec.
func (in *FilebeatDeploymentSpec) DeepCopy() *FilebeatDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(FilebeatDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatList) DeepCopyInto(out *FilebeatList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Filebeat, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatList.
func (in *FilebeatList) DeepCopy() *FilebeatList {
	if in == nil {
		return nil
	}
	out := new(FilebeatList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FilebeatList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatLogstashExternalRef) DeepCopyInto(out *FilebeatLogstashExternalRef) {
	*out = *in
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatLogstashExternalRef.
func (in *FilebeatLogstashExternalRef) DeepCopy() *FilebeatLogstashExternalRef {
	if in == nil {
		return nil
	}
	out := new(FilebeatLogstashExternalRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatLogstashManagedRef) DeepCopyInto(out *FilebeatLogstashManagedRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatLogstashManagedRef.
func (in *FilebeatLogstashManagedRef) DeepCopy() *FilebeatLogstashManagedRef {
	if in == nil {
		return nil
	}
	out := new(FilebeatLogstashManagedRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatLogstashRef) DeepCopyInto(out *FilebeatLogstashRef) {
	*out = *in
	if in.ManagedLogstashRef != nil {
		in, out := &in.ManagedLogstashRef, &out.ManagedLogstashRef
		*out = new(FilebeatLogstashManagedRef)
		**out = **in
	}
	if in.ExternalLogstashRef != nil {
		in, out := &in.ExternalLogstashRef, &out.ExternalLogstashRef
		*out = new(FilebeatLogstashExternalRef)
		(*in).DeepCopyInto(*out)
	}
	if in.LogstashCaSecretRef != nil {
		in, out := &in.LogstashCaSecretRef, &out.LogstashCaSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatLogstashRef.
func (in *FilebeatLogstashRef) DeepCopy() *FilebeatLogstashRef {
	if in == nil {
		return nil
	}
	out := new(FilebeatLogstashRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatPkiSpec) DeepCopyInto(out *FilebeatPkiSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.ValidityDays != nil {
		in, out := &in.ValidityDays, &out.ValidityDays
		*out = new(int)
		**out = **in
	}
	if in.RenewalDays != nil {
		in, out := &in.RenewalDays, &out.RenewalDays
		*out = new(int)
		**out = **in
	}
	if in.KeySize != nil {
		in, out := &in.KeySize, &out.KeySize
		*out = new(int)
		**out = **in
	}
	if in.Tls != nil {
		in, out := &in.Tls, &out.Tls
		*out = make(map[string]shared.TlsSelfSignedCertificateSpec, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatPkiSpec.
func (in *FilebeatPkiSpec) DeepCopy() *FilebeatPkiSpec {
	if in == nil {
		return nil
	}
	out := new(FilebeatPkiSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatSpec) DeepCopyInto(out *FilebeatSpec) {
	*out = *in
	in.ImageSpec.DeepCopyInto(&out.ImageSpec)
	in.ElasticsearchRef.DeepCopyInto(&out.ElasticsearchRef)
	in.LogstashRef.DeepCopyInto(&out.LogstashRef)
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = (*in).DeepCopy()
	}
	if in.ExtraConfigs != nil {
		in, out := &in.ExtraConfigs, &out.ExtraConfigs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Modules != nil {
		in, out := &in.Modules, &out.Modules
		*out = (*in).DeepCopy()
	}
	in.Deployment.DeepCopyInto(&out.Deployment)
	in.Monitoring.DeepCopyInto(&out.Monitoring)
	if in.Ingresses != nil {
		in, out := &in.Ingresses, &out.Ingresses
		*out = make([]shared.Ingress, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Routes != nil {
		in, out := &in.Routes, &out.Routes
		*out = make([]shared.Route, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]shared.Service, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Pki.DeepCopyInto(&out.Pki)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatSpec.
func (in *FilebeatSpec) DeepCopy() *FilebeatSpec {
	if in == nil {
		return nil
	}
	out := new(FilebeatSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilebeatStatus) DeepCopyInto(out *FilebeatStatus) {
	*out = *in
	in.DefaultMultiPhaseObjectStatus.DeepCopyInto(&out.DefaultMultiPhaseObjectStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilebeatStatus.
func (in *FilebeatStatus) DeepCopy() *FilebeatStatus {
	if in == nil {
		return nil
	}
	out := new(FilebeatStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metricbeat) DeepCopyInto(out *Metricbeat) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metricbeat.
func (in *Metricbeat) DeepCopy() *Metricbeat {
	if in == nil {
		return nil
	}
	out := new(Metricbeat)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Metricbeat) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricbeatDeploymentSpec) DeepCopyInto(out *MetricbeatDeploymentSpec) {
	*out = *in
	in.Deployment.DeepCopyInto(&out.Deployment)
	if in.AntiAffinity != nil {
		in, out := &in.AntiAffinity, &out.AntiAffinity
		*out = new(shared.DeploymentAntiAffinitySpec)
		**out = **in
	}
	if in.PodDisruptionBudgetSpec != nil {
		in, out := &in.PodDisruptionBudgetSpec, &out.PodDisruptionBudgetSpec
		*out = new(policyv1.PodDisruptionBudgetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.InitContainerResources != nil {
		in, out := &in.InitContainerResources, &out.InitContainerResources
		*out = new(corev1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.AdditionalVolumes != nil {
		in, out := &in.AdditionalVolumes, &out.AdditionalVolumes
		*out = make([]shared.DeploymentVolumeSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Persistence != nil {
		in, out := &in.Persistence, &out.Persistence
		*out = new(shared.DeploymentPersistenceSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricbeatDeploymentSpec.
func (in *MetricbeatDeploymentSpec) DeepCopy() *MetricbeatDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(MetricbeatDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricbeatList) DeepCopyInto(out *MetricbeatList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Metricbeat, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricbeatList.
func (in *MetricbeatList) DeepCopy() *MetricbeatList {
	if in == nil {
		return nil
	}
	out := new(MetricbeatList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MetricbeatList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricbeatPersistenceSpec) DeepCopyInto(out *MetricbeatPersistenceSpec) {
	*out = *in
	if in.VolumeClaimSpec != nil {
		in, out := &in.VolumeClaimSpec, &out.VolumeClaimSpec
		*out = new(corev1.PersistentVolumeClaimSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Volume != nil {
		in, out := &in.Volume, &out.Volume
		*out = new(corev1.VolumeSource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricbeatPersistenceSpec.
func (in *MetricbeatPersistenceSpec) DeepCopy() *MetricbeatPersistenceSpec {
	if in == nil {
		return nil
	}
	out := new(MetricbeatPersistenceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricbeatSpec) DeepCopyInto(out *MetricbeatSpec) {
	*out = *in
	in.ImageSpec.DeepCopyInto(&out.ImageSpec)
	in.ElasticsearchRef.DeepCopyInto(&out.ElasticsearchRef)
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = (*in).DeepCopy()
	}
	if in.ExtraConfigs != nil {
		in, out := &in.ExtraConfigs, &out.ExtraConfigs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Modules != nil {
		in, out := &in.Modules, &out.Modules
		*out = (*in).DeepCopy()
	}
	in.Deployment.DeepCopyInto(&out.Deployment)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricbeatSpec.
func (in *MetricbeatSpec) DeepCopy() *MetricbeatSpec {
	if in == nil {
		return nil
	}
	out := new(MetricbeatSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricbeatStatus) DeepCopyInto(out *MetricbeatStatus) {
	*out = *in
	in.DefaultMultiPhaseObjectStatus.DeepCopyInto(&out.DefaultMultiPhaseObjectStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricbeatStatus.
func (in *MetricbeatStatus) DeepCopy() *MetricbeatStatus {
	if in == nil {
		return nil
	}
	out := new(MetricbeatStatus)
	in.DeepCopyInto(out)
	return out
}
