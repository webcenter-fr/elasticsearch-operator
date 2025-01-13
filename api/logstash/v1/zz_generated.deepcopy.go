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
func (in *Logstash) DeepCopyInto(out *Logstash) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Logstash.
func (in *Logstash) DeepCopy() *Logstash {
	if in == nil {
		return nil
	}
	out := new(Logstash)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Logstash) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogstashDeploymentSpec) DeepCopyInto(out *LogstashDeploymentSpec) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogstashDeploymentSpec.
func (in *LogstashDeploymentSpec) DeepCopy() *LogstashDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(LogstashDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogstashList) DeepCopyInto(out *LogstashList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Logstash, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogstashList.
func (in *LogstashList) DeepCopy() *LogstashList {
	if in == nil {
		return nil
	}
	out := new(LogstashList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LogstashList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogstashPkiSpec) DeepCopyInto(out *LogstashPkiSpec) {
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
		*out = make(map[string]LogstashTlsSpec, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogstashPkiSpec.
func (in *LogstashPkiSpec) DeepCopy() *LogstashPkiSpec {
	if in == nil {
		return nil
	}
	out := new(LogstashPkiSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogstashSpec) DeepCopyInto(out *LogstashSpec) {
	*out = *in
	in.ImageSpec.DeepCopyInto(&out.ImageSpec)
	in.ElasticsearchRef.DeepCopyInto(&out.ElasticsearchRef)
	if in.PluginsList != nil {
		in, out := &in.PluginsList, &out.PluginsList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
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
	if in.Pipelines != nil {
		in, out := &in.Pipelines, &out.Pipelines
		*out = (*in).DeepCopy()
	}
	if in.Patterns != nil {
		in, out := &in.Patterns, &out.Patterns
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.KeystoreSecretRef != nil {
		in, out := &in.KeystoreSecretRef, &out.KeystoreSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogstashSpec.
func (in *LogstashSpec) DeepCopy() *LogstashSpec {
	if in == nil {
		return nil
	}
	out := new(LogstashSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogstashStatus) DeepCopyInto(out *LogstashStatus) {
	*out = *in
	in.BasicMultiPhaseObjectStatus.DeepCopyInto(&out.BasicMultiPhaseObjectStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogstashStatus.
func (in *LogstashStatus) DeepCopy() *LogstashStatus {
	if in == nil {
		return nil
	}
	out := new(LogstashStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogstashTlsSpec) DeepCopyInto(out *LogstashTlsSpec) {
	*out = *in
	in.TlsSelfSignedCertificateSpec.DeepCopyInto(&out.TlsSelfSignedCertificateSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogstashTlsSpec.
func (in *LogstashTlsSpec) DeepCopy() *LogstashTlsSpec {
	if in == nil {
		return nil
	}
	out := new(LogstashTlsSpec)
	in.DeepCopyInto(out)
	return out
}
