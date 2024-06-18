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

package v1alpha1

import (
	networkingv1 "k8s.io/api/networking/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewaySpec defines the desired state of the ingress or gateway. It will inform the kode-operator how to publish the Kode resource.
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
