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

package v1alpha1

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// EntryPointSpec defines the desired state of EntryPoint
type EntryPointSpec struct {
	// Type is the type of the gateway. It could be ingress-api or gateway-api.
	// +kubebuilder:validation:description=Type is the type of the gateway. It could be ingress-api or gateway-api.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ingress-api;gateway-api
	ApiType string `json:"apiType"`

	// Type is the way the Kode resource is accessed. It could be subdomain or path.
	// +kubebuilder:validation:description=Type is the way the Kode resource is accessed. It could be subdomain or path.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=domain;path
	Type string `json:"type"`

	// URL is the domain name to use either as a suffix in the case of Type=domain or as a prefix/domain in the case of Type=path.
	// When the type is domain, the controller will try to publish the Kode resource as a subdomain of the given domain (e.g <kode-resource>.kode.example.com).
	// When the type is path, the controller will try to publish the Kode resource as a path of the given URL (e.g kode.example.com/<kode-resource>).
	// +kubebuilder:validation:description=URL is the domain name to use either as a suffix in the case of Type=domain or as a prefix/domain in the case of Type=path. When the type is domain, the controller will try to publish the Kode resource as a subdomain of the given domain (e.g <kode-resource>.kode.example.com). When the type is path, the controller will try to publish the Kode resource as a path of the given URL (e.g kode.example.com/<kode-resource>).
	// +kubebuilder:validation:Required
	URL string `json:"url"`
}

// EntryPointPhase defines the phase of the EntryPoint
type EntryPointPhase string

const (
	// EntryPointPhaseCreating means the EntryPoint is being created.
	EntryPointPhaseCreating EntryPointPhase = "Creating"

	// KodePhaseCreated indicates that the Kode resource has been created.
	EntryPointPhaseCreated EntryPointPhase = "Created"

	// EntryPointPhaseFailed means the EntryPoint has failed.
	EntryPointPhaseFailed EntryPointPhase = "Failed"

	// EntryPointPhasePending means the EntryPoint is pending.
	EntryPointPhasePending EntryPointPhase = "Pending"

	// EntryPointPhaseDeleting means the EntryPoint is being deleted.
	EntryPointPhaseDeleting EntryPointPhase = "Deleting"
)

// EntryPointStatus defines the observed state of EntryPoint
type EntryPointStatus struct {
	// Phase represents the current phase of the Kode resource.
	Phase EntryPointPhase `json:"phase"`

	// Conditions represent the latest available observations of a Kode's state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastError contains the last error message encountered during reconciliation.
	LastError string `json:"lastError,omitempty"`

	// LastErrorTime is the timestamp when the last error occurred.
	LastErrorTime *metav1.Time `json:"lastErrorTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EntryPoint is the Schema for the entrypoints API
type EntryPoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EntryPointSpec   `json:"spec,omitempty"`
	Status EntryPointStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EntryPointList contains a list of EntryPoint
type EntryPointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EntryPoint `json:"items"`
}

type IngressSpec struct {
	// IngressClassName is the name of the IngressClass cluster resource.
	// +kubebuilder:validation:Description="Name of the IngressClass cluster resource."
	IngressClassName *string `json:"ingressClassName,omitempty"`

	// Rules defines the rules mapping the paths under a specified host to the related backend services.
	// +kubebuilder:validation:Description="Defines the rules mapping the paths under a specified host to the related backend services."
	Rules []networkingv1.IngressRule `json:"rules,omitempty"`

	// TLS contains the TLS configuration for the Ingress.
	// +kubebuilder:validation:Description="Contains the TLS configuration for the Ingress."
	TLS []networkingv1.IngressTLS `json:"tls,omitempty"`
}

type GatewaySpec struct {
	// GatewayClassName is the name of the GatewayClass resource.
	// +kubebuilder:validation:Description="Name of the GatewayClass resource."
	GatewayClassName *string `json:"gatewayClassName,omitempty"`

	// Listeners contains the listener configuration for the Gateway.
	// +kubebuilder:validation:Description="Contains the listener configuration for the Gateway."
	Listeners []gatewayv1.Listener `json:"listeners,omitempty"`

	// Routes contains the route configuration for the Gateway.
	// +kubebuilder:validation:Description="Contains the route configuration for the Gateway."
	Routes []gatewayv1.HTTPRoute `json:"routes,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EntryPoint{}, &EntryPointList{})
}
