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

package shared

import (
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Deployment) DeepCopyInto(out *Deployment) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]v1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodTemplate != nil {
		in, out := &in.PodTemplate, &out.PodTemplate
		*out = new(v1.PodTemplateSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Deployment.
func (in *Deployment) DeepCopy() *Deployment {
	if in == nil {
		return nil
	}
	out := new(Deployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentAntiAffinitySpec) DeepCopyInto(out *DeploymentAntiAffinitySpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentAntiAffinitySpec.
func (in *DeploymentAntiAffinitySpec) DeepCopy() *DeploymentAntiAffinitySpec {
	if in == nil {
		return nil
	}
	out := new(DeploymentAntiAffinitySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentPersistenceSpec) DeepCopyInto(out *DeploymentPersistenceSpec) {
	*out = *in
	if in.VolumeClaim != nil {
		in, out := &in.VolumeClaim, &out.VolumeClaim
		*out = new(DeploymentVolumeClaim)
		(*in).DeepCopyInto(*out)
	}
	if in.Volume != nil {
		in, out := &in.Volume, &out.Volume
		*out = new(v1.VolumeSource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentPersistenceSpec.
func (in *DeploymentPersistenceSpec) DeepCopy() *DeploymentPersistenceSpec {
	if in == nil {
		return nil
	}
	out := new(DeploymentPersistenceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentVolumeClaim) DeepCopyInto(out *DeploymentVolumeClaim) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.PersistentVolumeClaimSpec.DeepCopyInto(&out.PersistentVolumeClaimSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentVolumeClaim.
func (in *DeploymentVolumeClaim) DeepCopy() *DeploymentVolumeClaim {
	if in == nil {
		return nil
	}
	out := new(DeploymentVolumeClaim)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentVolumeSpec) DeepCopyInto(out *DeploymentVolumeSpec) {
	*out = *in
	in.VolumeMount.DeepCopyInto(&out.VolumeMount)
	in.VolumeSource.DeepCopyInto(&out.VolumeSource)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentVolumeSpec.
func (in *DeploymentVolumeSpec) DeepCopy() *DeploymentVolumeSpec {
	if in == nil {
		return nil
	}
	out := new(DeploymentVolumeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ElasticsearchExternalRef) DeepCopyInto(out *ElasticsearchExternalRef) {
	*out = *in
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
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
func (in *ElasticsearchManagedRef) DeepCopyInto(out *ElasticsearchManagedRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ElasticsearchManagedRef.
func (in *ElasticsearchManagedRef) DeepCopy() *ElasticsearchManagedRef {
	if in == nil {
		return nil
	}
	out := new(ElasticsearchManagedRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ElasticsearchRef) DeepCopyInto(out *ElasticsearchRef) {
	*out = *in
	if in.ManagedElasticsearchRef != nil {
		in, out := &in.ManagedElasticsearchRef, &out.ManagedElasticsearchRef
		*out = new(ElasticsearchManagedRef)
		**out = **in
	}
	if in.ExternalElasticsearchRef != nil {
		in, out := &in.ExternalElasticsearchRef, &out.ExternalElasticsearchRef
		*out = new(ElasticsearchExternalRef)
		(*in).DeepCopyInto(*out)
	}
	if in.ElasticsearchCaSecretRef != nil {
		in, out := &in.ElasticsearchCaSecretRef, &out.ElasticsearchCaSecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
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
func (in *EndpointIngressSpec) DeepCopyInto(out *EndpointIngressSpec) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.IngressSpec != nil {
		in, out := &in.IngressSpec, &out.IngressSpec
		*out = new(networkingv1.IngressSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointIngressSpec.
func (in *EndpointIngressSpec) DeepCopy() *EndpointIngressSpec {
	if in == nil {
		return nil
	}
	out := new(EndpointIngressSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointLoadBalancerSpec) DeepCopyInto(out *EndpointLoadBalancerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointLoadBalancerSpec.
func (in *EndpointLoadBalancerSpec) DeepCopy() *EndpointLoadBalancerSpec {
	if in == nil {
		return nil
	}
	out := new(EndpointLoadBalancerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointRouteSpec) DeepCopyInto(out *EndpointRouteSpec) {
	*out = *in
	if in.TlsEnabled != nil {
		in, out := &in.TlsEnabled, &out.TlsEnabled
		*out = new(bool)
		**out = **in
	}
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.RouteSpec != nil {
		in, out := &in.RouteSpec, &out.RouteSpec
		*out = new(routev1.RouteSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointRouteSpec.
func (in *EndpointRouteSpec) DeepCopy() *EndpointRouteSpec {
	if in == nil {
		return nil
	}
	out := new(EndpointRouteSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointSpec) DeepCopyInto(out *EndpointSpec) {
	*out = *in
	if in.Ingress != nil {
		in, out := &in.Ingress, &out.Ingress
		*out = new(EndpointIngressSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Route != nil {
		in, out := &in.Route, &out.Route
		*out = new(EndpointRouteSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.LoadBalancer != nil {
		in, out := &in.LoadBalancer, &out.LoadBalancer
		*out = new(EndpointLoadBalancerSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointSpec.
func (in *EndpointSpec) DeepCopy() *EndpointSpec {
	if in == nil {
		return nil
	}
	out := new(EndpointSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSpec.
func (in *ImageSpec) DeepCopy() *ImageSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Ingress) DeepCopyInto(out *Ingress) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Ingress.
func (in *Ingress) DeepCopy() *Ingress {
	if in == nil {
		return nil
	}
	out := new(Ingress)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KibanaExternalRef) DeepCopyInto(out *KibanaExternalRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KibanaExternalRef.
func (in *KibanaExternalRef) DeepCopy() *KibanaExternalRef {
	if in == nil {
		return nil
	}
	out := new(KibanaExternalRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KibanaManagedRef) DeepCopyInto(out *KibanaManagedRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KibanaManagedRef.
func (in *KibanaManagedRef) DeepCopy() *KibanaManagedRef {
	if in == nil {
		return nil
	}
	out := new(KibanaManagedRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KibanaRef) DeepCopyInto(out *KibanaRef) {
	*out = *in
	if in.ManagedKibanaRef != nil {
		in, out := &in.ManagedKibanaRef, &out.ManagedKibanaRef
		*out = new(KibanaManagedRef)
		**out = **in
	}
	if in.ExternalKibanaRef != nil {
		in, out := &in.ExternalKibanaRef, &out.ExternalKibanaRef
		*out = new(KibanaExternalRef)
		**out = **in
	}
	if in.KibanaCaSecretRef != nil {
		in, out := &in.KibanaCaSecretRef, &out.KibanaCaSecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.KibanaCredentialSecretRef != nil {
		in, out := &in.KibanaCredentialSecretRef, &out.KibanaCredentialSecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KibanaRef.
func (in *KibanaRef) DeepCopy() *KibanaRef {
	if in == nil {
		return nil
	}
	out := new(KibanaRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MonitoringMetricbeatSpec) DeepCopyInto(out *MonitoringMetricbeatSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	in.ElasticsearchRef.DeepCopyInto(&out.ElasticsearchRef)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MonitoringMetricbeatSpec.
func (in *MonitoringMetricbeatSpec) DeepCopy() *MonitoringMetricbeatSpec {
	if in == nil {
		return nil
	}
	out := new(MonitoringMetricbeatSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MonitoringPrometheusSpec) DeepCopyInto(out *MonitoringPrometheusSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	in.ImageSpec.DeepCopyInto(&out.ImageSpec)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	if in.InstallPlugin != nil {
		in, out := &in.InstallPlugin, &out.InstallPlugin
		*out = new(bool)
		**out = **in
	}
	if in.ScrapInterval != nil {
		in, out := &in.ScrapInterval, &out.ScrapInterval
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MonitoringPrometheusSpec.
func (in *MonitoringPrometheusSpec) DeepCopy() *MonitoringPrometheusSpec {
	if in == nil {
		return nil
	}
	out := new(MonitoringPrometheusSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MonitoringSpec) DeepCopyInto(out *MonitoringSpec) {
	*out = *in
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(MonitoringPrometheusSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Metricbeat != nil {
		in, out := &in.Metricbeat, &out.Metricbeat
		*out = new(MonitoringMetricbeatSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MonitoringSpec.
func (in *MonitoringSpec) DeepCopy() *MonitoringSpec {
	if in == nil {
		return nil
	}
	out := new(MonitoringSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Route) DeepCopyInto(out *Route) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Route.
func (in *Route) DeepCopy() *Route {
	if in == nil {
		return nil
	}
	out := new(Route)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Service) DeepCopyInto(out *Service) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Service.
func (in *Service) DeepCopy() *Service {
	if in == nil {
		return nil
	}
	out := new(Service)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TlsSelfSignedCertificateSpec) DeepCopyInto(out *TlsSelfSignedCertificateSpec) {
	*out = *in
	if in.AltIps != nil {
		in, out := &in.AltIps, &out.AltIps
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AltNames != nil {
		in, out := &in.AltNames, &out.AltNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TlsSelfSignedCertificateSpec.
func (in *TlsSelfSignedCertificateSpec) DeepCopy() *TlsSelfSignedCertificateSpec {
	if in == nil {
		return nil
	}
	out := new(TlsSelfSignedCertificateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TlsSpec) DeepCopyInto(out *TlsSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.SelfSignedCertificate != nil {
		in, out := &in.SelfSignedCertificate, &out.SelfSignedCertificate
		*out = new(TlsSelfSignedCertificateSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CertificateSecretRef != nil {
		in, out := &in.CertificateSecretRef, &out.CertificateSecretRef
		*out = new(v1.LocalObjectReference)
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
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TlsSpec.
func (in *TlsSpec) DeepCopy() *TlsSpec {
	if in == nil {
		return nil
	}
	out := new(TlsSpec)
	in.DeepCopyInto(out)
	return out
}
