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

package v1alpha2

import (
	"context"

	egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/jacero-io/kode-operator/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EntryPointSpec defines the desired state of EntryPoint
type EntryPointSpec struct {
	// The way the Kode resource is accessed by the user. It could be subdomain or path.
	// Path means the Kode resource is accessed as a path of the BaseDomain (e.g kode.example.com/<kode-resource>).
	// Subdomain means the Kode resource is accessed as a subdomain of the BaseDomain (e.g <kode-resource>.kode.example.com).
	// +kubebuilder:validation:Enum=subdomain;path
	// +kubebuilder:default=path
	RoutingType RoutingType `json:"routingType" yaml:"routingType"`

	// The domain name to use either as a suffix in the case of Type=domain or as a prefix/domain in the case of Type=path.
	// When the type is domain, the controller will try to publish the Kode resource as a subdomain of the given domain (e.g <kode-resource>.kode.example.com).
	// When the type is path, the controller will try to publish the Kode resource as a path of the given BaseDomain (e.g kode.example.com/<kode-resource>).
	// +kubebuilder:validation:Pattern=^([a-zA-Z0-9_]+\.)*[a-zA-Z0-9_]+$
	BaseDomain string `json:"baseDomain" yaml:"baseDomain"`

	// GatewaySpec defines the GatewaySpec for the EntryPoint.
	// +kubebuilder:validation:Optional
	GatewaySpec *GatewaySpec `json:"gatewaySpec,omitempty" yaml:"gatewaySpec,omitempty"`

	// AuthSpec defines the AuthSpec for the EntryPoint. Use this to influence the authentication and authorization policies of the EntryPoint.
	// +kubebuilder:validation:Optional
	AuthSpec *AuthSpec `json:"authSpec,omitempty" yaml:"authSpec,omitempty"`
}

type GatewaySpec struct {
	// Reference to an existing Gateway to use for the EntryPoint.
	ExistingGatewayRef *CrossNamespaceObjectReference `json:"existingGatewayRef" yaml:"existingGatewayRef"`

	// Protocol defines the protocol to use for the HTTPRoutes. Can be either "http" or "https".
	// +kubebuilder:validation:Enum=http;https
	// +kubebuilder:default=http
	Protocol Protocol `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

type AuthSpec struct {
	// The Envoy Gateway SecurityPolicy to use for the authentication. Can be either "none", "basicAuth", "oidc", "extAuth". Reference: https://gateway.envoyproxy.io/contributions/design/security-policy/
	// +kubebuilder:validation:Enum=none;basicAuth;extAuth
	// +kubebuilder:default=none
	AuthType AuthType `json:"authType" yaml:"authType"`

	// Defines the SecurityPolicies to be applied to the Route. Reference: https://gateway.envoyproxy.io/contributions/design/security-policy/
	// +kubebuilder:validation:Optional
	SecurityPolicySpec *SecurityPolicySpec `json:"securityPolicySpec,omitempty" yaml:"securityPolicySpec,omitempty"`

	// Reference to a field in the JWT token of OIDC or JWT. It will influence the controller on how to route the request and authorize the user.
	// +kubebuilder:validation:Optional
	IdentityReference *IdentityReference `json:"identityReference,omitempty" yaml:"identityReference,omitempty"`
}

type SecurityPolicySpec struct {
	// ExtAuth defines the configuration for External Authorization.
	//
	// +optional
	ExtAuth *egv1alpha1.ExtAuth `json:"extAuth,omitempty"`
}

// EntryPointStatus defines the observed state of EntryPoint
type EntryPointStatus struct {
	CommonStatus `json:",inline" yaml:",inline"`

	// RetryCount keeps track of the number of retry attempts for failed states.
	RetryCount int `json:"retryCount,omitempty"`

	// DeletionCycle keeps track of the number of deletion cycles. This is used to determine if the resource is deleting.
	DeletionCycle int `json:"deletionCycle,omitempty"` // TODO: Remove this field and use RetryCount instead. Deleting and failure are not executing at the same time.
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

// IdentityReference is a reference to a field in the JWT token of OIDC or JWT.
// TODO: Add validation pattern: "^[a-zA-Z0-9_-]+$"
type IdentityReference string

// AuthType is the type of authentication to use for the EntryPoint.
// +kubebuilder:validation:Enum=none;basicAuth;jwt;oidc;extAuth
type AuthType string

const (
	AuthTypeNone          AuthType = "none"
	AuthTypeBasicAuth     AuthType = "basicAuth"
	AuthTypeJWT           AuthType = "jwt"
	AuthTypeOIDC          AuthType = "oidc"
	AuthTypeExtAuth       AuthType = "extAuth"
	AuthTypeAuthorization AuthType = "authorization"
)

// BaseDomain is the domain name to use either as a suffix in the case of Type=domain or as a prefix/domain in the case of Type=path.
// TODO: Add validation pattern: "^([a-zA-Z0-9_]+\.)*[a-zA-Z0-9_]+$"
type BaseDomain string

// RoutingType is the way the Kode resource is accessed by the user. It could be subdomain or path.
type RoutingType string

const (
	RoutingTypeSubdomain RoutingType = "subdomain"
	RoutingTypePath      RoutingType = "path"
)

func init() {
	SchemeBuilder.Register(&EntryPoint{}, &EntryPointList{})
}

func (e *EntryPoint) GetName() string {
	return e.Name
}

func (e *EntryPoint) GetNamespace() string {
	return e.Namespace
}

func (e *EntryPoint) GetPhase() Phase {
	return e.Status.Phase
}

func (e *EntryPoint) SetPhase(phase Phase) {
	e.Status.Phase = phase
}

func (e *EntryPoint) UpdateStatus(ctx context.Context, c client.Client) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the EntryPoint
		latestEntryPoint := &EntryPoint{}
		if err := c.Get(ctx, client.ObjectKey{Name: e.Name, Namespace: e.Namespace}, latestEntryPoint); err != nil {
			return client.IgnoreNotFound(err)
		}

		// Create a patch
		patch := client.MergeFrom(latestEntryPoint.DeepCopy())

		// Update the status
		latestEntryPoint.Status = e.Status

		// When the EntryPoint resource transitions from Failed to another state, reset the LastError and LastErrorTime fields
		if e.Status.Phase != latestEntryPoint.Status.Phase && latestEntryPoint.Status.Phase != PhaseFailed {
			e.Status.LastError = nil
			e.Status.LastErrorTime = nil
		}

		// Apply the patch
		return c.Status().Patch(ctx, latestEntryPoint, patch)
	})
}

func (e *EntryPoint) SetCondition(conditionType constant.ConditionType, status metav1.ConditionStatus, reason, message string) {
	e.Status.SetCondition(conditionType, status, reason, message)
}

func (e *EntryPoint) GetCondition(conditionType constant.ConditionType) *metav1.Condition {
	return e.Status.GetCondition(conditionType)
}

func (e *EntryPoint) DeleteCondition(conditionType constant.ConditionType) {
	e.Status.DeleteCondition(conditionType)
}

func (e *EntryPoint) GetFinalizer() string {
	return constant.KodeFinalizerName
}

func (e *EntryPoint) AddFinalizer(ctx context.Context, c client.Client) error {
	controllerutil.AddFinalizer(e, e.GetFinalizer())
	if err := c.Update(ctx, e); err != nil {
		return err
	}
	return nil
}

func (e *EntryPoint) RemoveFinalizer(ctx context.Context, c client.Client) error {
	controllerutil.RemoveFinalizer(e, e.GetFinalizer())
	if err := c.Update(ctx, e); err != nil {
		return err
	}
	return nil
}

func (e *EntryPoint) IsSubdomainRouting() bool {
	return e.Spec.RoutingType == RoutingTypeSubdomain
}

func (e *EntryPoint) IsPathRouting() bool {
	return e.Spec.RoutingType == RoutingTypePath
}

func (e *EntryPoint) HasExistingGateway() bool {
	return e.Spec.GatewaySpec != nil && e.Spec.GatewaySpec.ExistingGatewayRef.Name != ""
}
