//go:build !ignore_autogenerated

/*
Copyright emil@jacero.se 2024.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apisv1 "sigs.k8s.io/gateway-api/apis/v1"
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
	out.Spec = in.Spec
	out.Status = in.Status
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
func (in *EnvoyProxyClusterConfig) DeepCopyInto(out *EnvoyProxyClusterConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyClusterConfig.
func (in *EnvoyProxyClusterConfig) DeepCopy() *EnvoyProxyClusterConfig {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EnvoyProxyClusterConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyClusterConfigList) DeepCopyInto(out *EnvoyProxyClusterConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EnvoyProxyClusterConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyClusterConfigList.
func (in *EnvoyProxyClusterConfigList) DeepCopy() *EnvoyProxyClusterConfigList {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyClusterConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EnvoyProxyClusterConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyClusterConfigSpec) DeepCopyInto(out *EnvoyProxyClusterConfigSpec) {
	*out = *in
	in.SharedEnvoyProxyConfigSpec.DeepCopyInto(&out.SharedEnvoyProxyConfigSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyClusterConfigSpec.
func (in *EnvoyProxyClusterConfigSpec) DeepCopy() *EnvoyProxyClusterConfigSpec {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyClusterConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyClusterConfigStatus) DeepCopyInto(out *EnvoyProxyClusterConfigStatus) {
	*out = *in
	out.SharedEnvoyProxyStatus = in.SharedEnvoyProxyStatus
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyClusterConfigStatus.
func (in *EnvoyProxyClusterConfigStatus) DeepCopy() *EnvoyProxyClusterConfigStatus {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyClusterConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyConfig) DeepCopyInto(out *EnvoyProxyConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyConfig.
func (in *EnvoyProxyConfig) DeepCopy() *EnvoyProxyConfig {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EnvoyProxyConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyConfigList) DeepCopyInto(out *EnvoyProxyConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EnvoyProxyConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyConfigList.
func (in *EnvoyProxyConfigList) DeepCopy() *EnvoyProxyConfigList {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EnvoyProxyConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyConfigSpec) DeepCopyInto(out *EnvoyProxyConfigSpec) {
	*out = *in
	in.SharedEnvoyProxyConfigSpec.DeepCopyInto(&out.SharedEnvoyProxyConfigSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyConfigSpec.
func (in *EnvoyProxyConfigSpec) DeepCopy() *EnvoyProxyConfigSpec {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyConfigStatus) DeepCopyInto(out *EnvoyProxyConfigStatus) {
	*out = *in
	out.SharedEnvoyProxyStatus = in.SharedEnvoyProxyStatus
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyConfigStatus.
func (in *EnvoyProxyConfigStatus) DeepCopy() *EnvoyProxyConfigStatus {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvoyProxyReference) DeepCopyInto(out *EnvoyProxyReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvoyProxyReference.
func (in *EnvoyProxyReference) DeepCopy() *EnvoyProxyReference {
	if in == nil {
		return nil
	}
	out := new(EnvoyProxyReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatewaySpec) DeepCopyInto(out *GatewaySpec) {
	*out = *in
	if in.GatewayClassName != nil {
		in, out := &in.GatewayClassName, &out.GatewayClassName
		*out = new(string)
		**out = **in
	}
	if in.Listeners != nil {
		in, out := &in.Listeners, &out.Listeners
		*out = make([]apisv1.Listener, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Routes != nil {
		in, out := &in.Routes, &out.Routes
		*out = make([]apisv1.HTTPRoute, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
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
func (in *IngressSpec) DeepCopyInto(out *IngressSpec) {
	*out = *in
	if in.IngressClassName != nil {
		in, out := &in.IngressClassName, &out.IngressClassName
		*out = new(string)
		**out = **in
	}
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]v1.IngressRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = make([]v1.IngressTLS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressSpec.
func (in *IngressSpec) DeepCopy() *IngressSpec {
	if in == nil {
		return nil
	}
	out := new(IngressSpec)
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
func (in *KodeClusterTemplate) DeepCopyInto(out *KodeClusterTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeClusterTemplate.
func (in *KodeClusterTemplate) DeepCopy() *KodeClusterTemplate {
	if in == nil {
		return nil
	}
	out := new(KodeClusterTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KodeClusterTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeClusterTemplateList) DeepCopyInto(out *KodeClusterTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KodeClusterTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeClusterTemplateList.
func (in *KodeClusterTemplateList) DeepCopy() *KodeClusterTemplateList {
	if in == nil {
		return nil
	}
	out := new(KodeClusterTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KodeClusterTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeClusterTemplateSpec) DeepCopyInto(out *KodeClusterTemplateSpec) {
	*out = *in
	in.SharedKodeTemplateSpec.DeepCopyInto(&out.SharedKodeTemplateSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeClusterTemplateSpec.
func (in *KodeClusterTemplateSpec) DeepCopy() *KodeClusterTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(KodeClusterTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeClusterTemplateStatus) DeepCopyInto(out *KodeClusterTemplateStatus) {
	*out = *in
	in.SharedKodeTemplateStatus.DeepCopyInto(&out.SharedKodeTemplateStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeClusterTemplateStatus.
func (in *KodeClusterTemplateStatus) DeepCopy() *KodeClusterTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(KodeClusterTemplateStatus)
	in.DeepCopyInto(out)
	return out
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
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
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
func (in *KodeTemplate) DeepCopyInto(out *KodeTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeTemplate.
func (in *KodeTemplate) DeepCopy() *KodeTemplate {
	if in == nil {
		return nil
	}
	out := new(KodeTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KodeTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeTemplateList) DeepCopyInto(out *KodeTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KodeTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeTemplateList.
func (in *KodeTemplateList) DeepCopy() *KodeTemplateList {
	if in == nil {
		return nil
	}
	out := new(KodeTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KodeTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeTemplateReference) DeepCopyInto(out *KodeTemplateReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeTemplateReference.
func (in *KodeTemplateReference) DeepCopy() *KodeTemplateReference {
	if in == nil {
		return nil
	}
	out := new(KodeTemplateReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeTemplateSpec) DeepCopyInto(out *KodeTemplateSpec) {
	*out = *in
	in.SharedKodeTemplateSpec.DeepCopyInto(&out.SharedKodeTemplateSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeTemplateSpec.
func (in *KodeTemplateSpec) DeepCopy() *KodeTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(KodeTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KodeTemplateStatus) DeepCopyInto(out *KodeTemplateStatus) {
	*out = *in
	in.SharedKodeTemplateStatus.DeepCopyInto(&out.SharedKodeTemplateStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KodeTemplateStatus.
func (in *KodeTemplateStatus) DeepCopy() *KodeTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(KodeTemplateStatus)
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
func (in *SharedEnvoyProxyConfigSpec) DeepCopyInto(out *SharedEnvoyProxyConfigSpec) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SharedEnvoyProxyConfigSpec.
func (in *SharedEnvoyProxyConfigSpec) DeepCopy() *SharedEnvoyProxyConfigSpec {
	if in == nil {
		return nil
	}
	out := new(SharedEnvoyProxyConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SharedEnvoyProxyStatus) DeepCopyInto(out *SharedEnvoyProxyStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SharedEnvoyProxyStatus.
func (in *SharedEnvoyProxyStatus) DeepCopy() *SharedEnvoyProxyStatus {
	if in == nil {
		return nil
	}
	out := new(SharedEnvoyProxyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SharedKodeTemplateSpec) DeepCopyInto(out *SharedKodeTemplateSpec) {
	*out = *in
	out.EnvoyProxyRef = in.EnvoyProxyRef
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
	if in.AllowPrivileged != nil {
		in, out := &in.AllowPrivileged, &out.AllowPrivileged
		*out = new(bool)
		**out = **in
	}
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
	if in.InitPlugins != nil {
		in, out := &in.InitPlugins, &out.InitPlugins
		*out = make([]InitPluginSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SharedKodeTemplateSpec.
func (in *SharedKodeTemplateSpec) DeepCopy() *SharedKodeTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(SharedKodeTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SharedKodeTemplateStatus) DeepCopyInto(out *SharedKodeTemplateStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SharedKodeTemplateStatus.
func (in *SharedKodeTemplateStatus) DeepCopy() *SharedKodeTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(SharedKodeTemplateStatus)
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
