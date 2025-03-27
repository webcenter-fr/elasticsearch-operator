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
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cerebro) DeepCopyInto(out *Cerebro) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cerebro.
func (in *Cerebro) DeepCopy() *Cerebro {
	if in == nil {
		return nil
	}
	out := new(Cerebro)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cerebro) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CerebroDeploymentSpec) DeepCopyInto(out *CerebroDeploymentSpec) {
	*out = *in
	in.Deployment.DeepCopyInto(&out.Deployment)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CerebroDeploymentSpec.
func (in *CerebroDeploymentSpec) DeepCopy() *CerebroDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(CerebroDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CerebroList) DeepCopyInto(out *CerebroList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cerebro, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CerebroList.
func (in *CerebroList) DeepCopy() *CerebroList {
	if in == nil {
		return nil
	}
	out := new(CerebroList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CerebroList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CerebroSpec) DeepCopyInto(out *CerebroSpec) {
	*out = *in
	in.ImageSpec.DeepCopyInto(&out.ImageSpec)
	in.Endpoint.DeepCopyInto(&out.Endpoint)
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(string)
		**out = **in
	}
	if in.ExtraConfigs != nil {
		in, out := &in.ExtraConfigs, &out.ExtraConfigs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Deployment.DeepCopyInto(&out.Deployment)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CerebroSpec.
func (in *CerebroSpec) DeepCopy() *CerebroSpec {
	if in == nil {
		return nil
	}
	out := new(CerebroSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CerebroStatus) DeepCopyInto(out *CerebroStatus) {
	*out = *in
	in.BasicMultiPhaseObjectStatus.DeepCopyInto(&out.BasicMultiPhaseObjectStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CerebroStatus.
func (in *CerebroStatus) DeepCopy() *CerebroStatus {
	if in == nil {
		return nil
	}
	out := new(CerebroStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ElasticsearchExternalRef) DeepCopyInto(out *ElasticsearchExternalRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ElasticsearchExternalRef.
func (in *ElasticsearchExternalRef) DeepCopy() *ElasticsearchExternalRef {
	if in == nil {
		return nil
	}
	out := new(ElasticsearchExternalRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ElasticsearchRef) DeepCopyInto(out *ElasticsearchRef) {
	*out = *in
	if in.ManagedElasticsearchRef != nil {
		in, out := &in.ManagedElasticsearchRef, &out.ManagedElasticsearchRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
	if in.ExternalElasticsearchRef != nil {
		in, out := &in.ExternalElasticsearchRef, &out.ExternalElasticsearchRef
		*out = new(ElasticsearchExternalRef)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ElasticsearchRef.
func (in *ElasticsearchRef) DeepCopy() *ElasticsearchRef {
	if in == nil {
		return nil
	}
	out := new(ElasticsearchRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Host) DeepCopyInto(out *Host) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Host.
func (in *Host) DeepCopy() *Host {
	if in == nil {
		return nil
	}
	out := new(Host)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Host) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostCerebroRef) DeepCopyInto(out *HostCerebroRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostCerebroRef.
func (in *HostCerebroRef) DeepCopy() *HostCerebroRef {
	if in == nil {
		return nil
	}
	out := new(HostCerebroRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostList) DeepCopyInto(out *HostList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Host, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostList.
func (in *HostList) DeepCopy() *HostList {
	if in == nil {
		return nil
	}
	out := new(HostList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HostList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostSpec) DeepCopyInto(out *HostSpec) {
	*out = *in
	out.CerebroRef = in.CerebroRef
	in.ElasticsearchRef.DeepCopyInto(&out.ElasticsearchRef)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostSpec.
func (in *HostSpec) DeepCopy() *HostSpec {
	if in == nil {
		return nil
	}
	out := new(HostSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostStatus) DeepCopyInto(out *HostStatus) {
	*out = *in
	in.BasicMultiPhaseObjectStatus.DeepCopyInto(&out.BasicMultiPhaseObjectStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostStatus.
func (in *HostStatus) DeepCopy() *HostStatus {
	if in == nil {
		return nil
	}
	out := new(HostStatus)
	in.DeepCopyInto(out)
	return out
}
