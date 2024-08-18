//go:build !ignore_autogenerated

/*
Copyright 2024 Emil Larsson.

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

package v1alpha2

import (
	"github.com/envoyproxy/gateway/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Address) DeepCopyInto(out *Address) {
	*out = *in
	out.SocketAddress = in.SocketAddress
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Address.
func (in *Address) DeepCopy() *Address {
	if in == nil {
		return nil
	}
	out := new(Address)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthSpec) DeepCopyInto(out *AuthSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthSpec.
func (in *AuthSpec) DeepCopy() *AuthSpec {
	if in == nil {
		return nil
	}
	out := new(AuthSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BaseSharedSpec) DeepCopyInto(out *BaseSharedSpec) {
	*out = *in
	out.Credentials = in.Credentials
	out.EntryPointRef = in.EntryPointRef
	if in.InactiveAfterSeconds != nil {
		in, out := &in.InactiveAfterSeconds, &out.InactiveAfterSeconds
		*out = new(int64)
		**out = **in
	}
	if in.RecycleAfterSeconds != nil {
		in, out := &in.RecycleAfterSeconds, &out.RecycleAfterSeconds
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BaseSharedSpec.
func (in *BaseSharedSpec) DeepCopy() *BaseSharedSpec {
	if in == nil {
		return nil
	}
	out := new(BaseSharedSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BaseSharedStatus) DeepCopyInto(out *BaseSharedStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BaseSharedStatus.
func (in *BaseSharedStatus) DeepCopy() *BaseSharedStatus {
	if in == nil {
		return nil
	}
	out := new(BaseSharedStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cluster) DeepCopyInto(out *Cluster) {
	*out = *in
	in.TypedExtensionProtocolOptions.DeepCopyInto(&out.TypedExtensionProtocolOptions)
	in.LoadAssignment.DeepCopyInto(&out.LoadAssignment)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cluster.
func (in *Cluster) DeepCopy() *Cluster {
	if in == nil {
		return nil
	}
	out := new(Cluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPodTemplate) DeepCopyInto(out *ClusterPodTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPodTemplate.
func (in *ClusterPodTemplate) DeepCopy() *ClusterPodTemplate {
	if in == nil {
		return nil
	}
	out := new(ClusterPodTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterPodTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPodTemplateList) DeepCopyInto(out *ClusterPodTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterPodTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPodTemplateList.
func (in *ClusterPodTemplateList) DeepCopy() *ClusterPodTemplateList {
	if in == nil {
		return nil
	}
	out := new(ClusterPodTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterPodTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPodTemplateSpec) DeepCopyInto(out *ClusterPodTemplateSpec) {
	*out = *in
	in.PodTemplateSharedSpec.DeepCopyInto(&out.PodTemplateSharedSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPodTemplateSpec.
func (in *ClusterPodTemplateSpec) DeepCopy() *ClusterPodTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterPodTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPodTemplateStatus) DeepCopyInto(out *ClusterPodTemplateStatus) {
	*out = *in
	in.PodTemplateSharedStatus.DeepCopyInto(&out.PodTemplateSharedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPodTemplateStatus.
func (in *ClusterPodTemplateStatus) DeepCopy() *ClusterPodTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterPodTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTofuTemplate) DeepCopyInto(out *ClusterTofuTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTofuTemplate.
func (in *ClusterTofuTemplate) DeepCopy() *ClusterTofuTemplate {
	if in == nil {
		return nil
	}
	out := new(ClusterTofuTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTofuTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTofuTemplateList) DeepCopyInto(out *ClusterTofuTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterTofuTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTofuTemplateList.
func (in *ClusterTofuTemplateList) DeepCopy() *ClusterTofuTemplateList {
	if in == nil {
		return nil
	}
	out := new(ClusterTofuTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTofuTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTofuTemplateSpec) DeepCopyInto(out *ClusterTofuTemplateSpec) {
	*out = *in
	in.TofuSharedSpec.DeepCopyInto(&out.TofuSharedSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTofuTemplateSpec.
func (in *ClusterTofuTemplateSpec) DeepCopy() *ClusterTofuTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterTofuTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTofuTemplateStatus) DeepCopyInto(out *ClusterTofuTemplateStatus) {
	*out = *in
	in.TofuSharedStatus.DeepCopyInto(&out.TofuSharedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTofuTemplateStatus.
func (in *ClusterTofuTemplateStatus) DeepCopy() *ClusterTofuTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterTofuTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CredentialsSpec) DeepCopyInto(out *CredentialsSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CredentialsSpec.
func (in *CredentialsSpec) DeepCopy() *CredentialsSpec {
	if in == nil {
		return nil
	}
	out := new(CredentialsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CrossNamespaceObjectReference) DeepCopyInto(out *CrossNamespaceObjectReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CrossNamespaceObjectReference.
func (in *CrossNamespaceObjectReference) DeepCopy() *CrossNamespaceObjectReference {
	if in == nil {
		return nil
	}
	out := new(CrossNamespaceObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Endpoint) DeepCopyInto(out *Endpoint) {
	*out = *in
	out.Address = in.Address
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Endpoint.
func (in *Endpoint) DeepCopy() *Endpoint {
	if in == nil {
		return nil
	}
	out := new(Endpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Endpoints) DeepCopyInto(out *Endpoints) {
	*out = *in
	if in.LbEndpoints != nil {
		in, out := &in.LbEndpoints, &out.LbEndpoints
		*out = make([]LbEndpoint, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Endpoints.
func (in *Endpoints) DeepCopy() *Endpoints {
	if in == nil {
		return nil
	}
	out := new(Endpoints)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EntryPoint) DeepCopyInto(out *EntryPoint) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EntryPoint.
func (in *EntryPoint) DeepCopy() *EntryPoint {
	if in == nil {
		return nil
	}
	out := new(EntryPoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EntryPoint) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EntryPointList) DeepCopyInto(out *EntryPointList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EntryPoint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EntryPointList.
func (in *EntryPointList) DeepCopy() *EntryPointList {
	if in == nil {
		return nil
	}
	out := new(EntryPointList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EntryPointList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EntryPointSpec) DeepCopyInto(out *EntryPointSpec) {
	*out = *in
	if in.GatewaySpec != nil {
		in, out := &in.GatewaySpec, &out.GatewaySpec
		*out = new(GatewaySpec)
		(*in).DeepCopyInto(*out)
	}
	if in.AuthSpec != nil {
		in, out := &in.AuthSpec, &out.AuthSpec
		*out = new(AuthSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EntryPointSpec.
func (in *EntryPointSpec) DeepCopy() *EntryPointSpec {
	if in == nil {
		return nil
	}
	out := new(EntryPointSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EntryPointStatus) DeepCopyInto(out *EntryPointStatus) {
	*out = *in
	in.BaseSharedStatus.DeepCopyInto(&out.BaseSharedStatus)
	if in.LastErrorTime != nil {
		in, out := &in.LastErrorTime, &out.LastErrorTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EntryPointStatus.
func (in *EntryPointStatus) DeepCopy() *EntryPointStatus {
	if in == nil {
		return nil
	}
	out := new(EntryPointStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatewaySpec) DeepCopyInto(out *GatewaySpec) {
	*out = *in
	out.ExistingGatewayRef = in.ExistingGatewayRef
	if in.Certificate != nil {
		in, out := &in.Certificate, &out.Certificate
		*out = make([]corev1.Secret, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.RouteSecurityPolicy.DeepCopyInto(&out.RouteSecurityPolicy)
	if in.EnvoyPatchPolicySpec != nil {
		in, out := &in.EnvoyPatchPolicySpec, &out.EnvoyPatchPolicySpec
		*out = new(v1alpha1.EnvoyPatchPolicySpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatewaySpec.
func (in *GatewaySpec) DeepCopy() *GatewaySpec {
	if in == nil {
		return nil
	}
	out := new(GatewaySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPFilter) DeepCopyInto(out *HTTPFilter) {
	*out = *in
	in.TypedConfig.DeepCopyInto(&out.TypedConfig)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPFilter.
func (in *HTTPFilter) DeepCopy() *HTTPFilter {
	if in == nil {
		return nil
	}
	out := new(HTTPFilter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InitPluginSpec) DeepCopyInto(out *InitPluginSpec) {
	*out = *in
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]corev1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]corev1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InitPluginSpec.
func (in *InitPluginSpec) DeepCopy() *InitPluginSpec {
	if in == nil {
		return nil
	}
	out := new(InitPluginSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kode) DeepCopyInto(out *Kode) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kode.
func (in *Kode) DeepCopy() *Kode {
	if in == nil {
		return nil
	}
	out := new(Kode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Kode) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeList) DeepCopyInto(out *KodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Kode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeList.
func (in *KodeList) DeepCopy() *KodeList {
	if in == nil {
		return nil
	}
	out := new(KodeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KodeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeSpec) DeepCopyInto(out *KodeSpec) {
	*out = *in
	out.TemplateRef = in.TemplateRef
	out.Credentials = in.Credentials
	in.Storage.DeepCopyInto(&out.Storage)
	if in.Privileged != nil {
		in, out := &in.Privileged, &out.Privileged
		*out = new(bool)
		**out = **in
	}
	if in.InitPlugins != nil {
		in, out := &in.InitPlugins, &out.InitPlugins
		*out = make([]InitPluginSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeSpec.
func (in *KodeSpec) DeepCopy() *KodeSpec {
	if in == nil {
		return nil
	}
	out := new(KodeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeStatus) DeepCopyInto(out *KodeStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LastActivityTime != nil {
		in, out := &in.LastActivityTime, &out.LastActivityTime
		*out = (*in).DeepCopy()
	}
	if in.LastErrorTime != nil {
		in, out := &in.LastErrorTime, &out.LastErrorTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeStatus.
func (in *KodeStatus) DeepCopy() *KodeStatus {
	if in == nil {
		return nil
	}
	out := new(KodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeStorageSpec) DeepCopyInto(out *KodeStorageSpec) {
	*out = *in
	if in.AccessModes != nil {
		in, out := &in.AccessModes, &out.AccessModes
		*out = make([]corev1.PersistentVolumeAccessMode, len(*in))
		copy(*out, *in)
	}
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.KeepVolume != nil {
		in, out := &in.KeepVolume, &out.KeepVolume
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeStorageSpec.
func (in *KodeStorageSpec) DeepCopy() *KodeStorageSpec {
	if in == nil {
		return nil
	}
	out := new(KodeStorageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LbEndpoint) DeepCopyInto(out *LbEndpoint) {
	*out = *in
	out.Endpoint = in.Endpoint
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LbEndpoint.
func (in *LbEndpoint) DeepCopy() *LbEndpoint {
	if in == nil {
		return nil
	}
	out := new(LbEndpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadAssignment) DeepCopyInto(out *LoadAssignment) {
	*out = *in
	if in.Endpoints != nil {
		in, out := &in.Endpoints, &out.Endpoints
		*out = make([]Endpoints, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadAssignment.
func (in *LoadAssignment) DeepCopy() *LoadAssignment {
	if in == nil {
		return nil
	}
	out := new(LoadAssignment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PatchPolicy) DeepCopyInto(out *PatchPolicy) {
	*out = *in
	if in.HTTPFilters != nil {
		in, out := &in.HTTPFilters, &out.HTTPFilters
		*out = make([]HTTPFilter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Clusters != nil {
		in, out := &in.Clusters, &out.Clusters
		*out = make([]Cluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PatchPolicy.
func (in *PatchPolicy) DeepCopy() *PatchPolicy {
	if in == nil {
		return nil
	}
	out := new(PatchPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodTemplate) DeepCopyInto(out *PodTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTemplate.
func (in *PodTemplate) DeepCopy() *PodTemplate {
	if in == nil {
		return nil
	}
	out := new(PodTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PodTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodTemplateList) DeepCopyInto(out *PodTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PodTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTemplateList.
func (in *PodTemplateList) DeepCopy() *PodTemplateList {
	if in == nil {
		return nil
	}
	out := new(PodTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PodTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodTemplateSharedSpec) DeepCopyInto(out *PodTemplateSharedSpec) {
	*out = *in
	in.BaseSharedSpec.DeepCopyInto(&out.BaseSharedSpec)
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]corev1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.AllowPrivileged != nil {
		in, out := &in.AllowPrivileged, &out.AllowPrivileged
		*out = new(bool)
		**out = **in
	}
	if in.InitPlugins != nil {
		in, out := &in.InitPlugins, &out.InitPlugins
		*out = make([]InitPluginSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTemplateSharedSpec.
func (in *PodTemplateSharedSpec) DeepCopy() *PodTemplateSharedSpec {
	if in == nil {
		return nil
	}
	out := new(PodTemplateSharedSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodTemplateSharedStatus) DeepCopyInto(out *PodTemplateSharedStatus) {
	*out = *in
	in.BaseSharedStatus.DeepCopyInto(&out.BaseSharedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTemplateSharedStatus.
func (in *PodTemplateSharedStatus) DeepCopy() *PodTemplateSharedStatus {
	if in == nil {
		return nil
	}
	out := new(PodTemplateSharedStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodTemplateSpec) DeepCopyInto(out *PodTemplateSpec) {
	*out = *in
	in.PodTemplateSharedSpec.DeepCopyInto(&out.PodTemplateSharedSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTemplateSpec.
func (in *PodTemplateSpec) DeepCopy() *PodTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(PodTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodTemplateStatus) DeepCopyInto(out *PodTemplateStatus) {
	*out = *in
	in.PodTemplateSharedStatus.DeepCopyInto(&out.PodTemplateSharedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTemplateStatus.
func (in *PodTemplateStatus) DeepCopy() *PodTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(PodTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SocketAddress) DeepCopyInto(out *SocketAddress) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SocketAddress.
func (in *SocketAddress) DeepCopy() *SocketAddress {
	if in == nil {
		return nil
	}
	out := new(SocketAddress)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Template) DeepCopyInto(out *Template) {
	*out = *in
	if in.PodTemplateSpec != nil {
		in, out := &in.PodTemplateSpec, &out.PodTemplateSpec
		*out = new(PodTemplateSharedSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.TofuTemplateSpec != nil {
		in, out := &in.TofuTemplateSpec, &out.TofuTemplateSpec
		*out = new(TofuSharedSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Template.
func (in *Template) DeepCopy() *Template {
	if in == nil {
		return nil
	}
	out := new(Template)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TofuSharedSpec) DeepCopyInto(out *TofuSharedSpec) {
	*out = *in
	in.BaseSharedSpec.DeepCopyInto(&out.BaseSharedSpec)
	out.EntryPointRef = in.EntryPointRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TofuSharedSpec.
func (in *TofuSharedSpec) DeepCopy() *TofuSharedSpec {
	if in == nil {
		return nil
	}
	out := new(TofuSharedSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TofuSharedStatus) DeepCopyInto(out *TofuSharedStatus) {
	*out = *in
	in.BaseSharedStatus.DeepCopyInto(&out.BaseSharedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TofuSharedStatus.
func (in *TofuSharedStatus) DeepCopy() *TofuSharedStatus {
	if in == nil {
		return nil
	}
	out := new(TofuSharedStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TofuTemplate) DeepCopyInto(out *TofuTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TofuTemplate.
func (in *TofuTemplate) DeepCopy() *TofuTemplate {
	if in == nil {
		return nil
	}
	out := new(TofuTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TofuTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TofuTemplateList) DeepCopyInto(out *TofuTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TofuTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TofuTemplateList.
func (in *TofuTemplateList) DeepCopy() *TofuTemplateList {
	if in == nil {
		return nil
	}
	out := new(TofuTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TofuTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TofuTemplateSpec) DeepCopyInto(out *TofuTemplateSpec) {
	*out = *in
	in.TofuSharedSpec.DeepCopyInto(&out.TofuSharedSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TofuTemplateSpec.
func (in *TofuTemplateSpec) DeepCopy() *TofuTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(TofuTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TofuTemplateStatus) DeepCopyInto(out *TofuTemplateStatus) {
	*out = *in
	in.TofuSharedStatus.DeepCopyInto(&out.TofuSharedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TofuTemplateStatus.
func (in *TofuTemplateStatus) DeepCopy() *TofuTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(TofuTemplateStatus)
	in.DeepCopyInto(out)
	return out
}
