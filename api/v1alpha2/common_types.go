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

package v1alpha2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type CredentialsSpec struct {
	// Is both the the HTTP Basic auth username (when used) and the user the container should run as. Defaults to 'abc'.
	Username string `json:"username,omitempty" yaml:"username,omitempty"`

	// HTTP Basic auth password. If unset, there will be no authentication.
	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	// ExistingSecret is a reference to an existing secret containing user and password. If set, User and Password fields are ignored.
	// MUST set "username" and "password" in lowercase in the secret. CAN set either "username" or "password" or both.
	ExistingSecret *string `json:"existingSecret,omitempty" yaml:"existingSecret,omitempty"`

	// EnableBuiltinAuth enables the built-in HTTP Basic auth.
	// +kubebuilder:default=false
	EnableBuiltinAuth bool `json:"enableBuiltinAuth,omitempty" yaml:"enableBuiltinAuth,omitempty"`
}

// CommonSpec defines the common fields for both Container and Virtualization specs
type CommonSpec struct {
	// Credentials specifies the credentials for the service.
	Credentials *CredentialsSpec `json:"credentials,omitempty" yaml:"credentials,omitempty"`

	// EntryPointSpec defines the desired state of the entrypoint.
	EntryPointRef *CrossNamespaceObjectReference `json:"entryPointRef,omitempty" yaml:"entryPointRef,omitempty"`

	// Specifies the period before controller inactive the resource (delete all resources except volume).
	// +kubebuilder:default=600
	InactiveAfterSeconds *int64 `json:"inactiveAfterSeconds,omitempty" yaml:"inactiveAfterSeconds,omitempty"`

	// Specifies the period before controller recycle the resource (delete all resources).
	// +kubebuilder:default=28800
	RecycleAfterSeconds *int64 `json:"recycleAfterSeconds,omitempty" yaml:"recycleAfterSeconds,omitempty"`

	// Port is the port for the service process. Used by EnvoyProxy to expose the kode.
	// +kubebuilder:default=8000
	Port Port `json:"port,omitempty" yaml:"port,omitempty"`
}

// CommonStatus defines the common observed state
type CommonStatus struct {
	// ObservedGeneration is the last observed generation of the resource.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`

	// Conditions reflect the current state of the resource
	ConditionedStatus `json:",inline" yaml:",inline"`

	// Contains the last error message encountered during reconciliation.
	LastError *string `json:"lastError,omitempty" yaml:"lastError,omitempty"`

	// The timestamp when the last error occurred.
	LastErrorTime *metav1.Time `json:"lastErrorTime,omitempty" yaml:"lastErrorTime,omitempty"`

	// Phase is the current state of the resource.
	Phase Phase `json:"phase,omitempty" yaml:"phase,omitempty"`
}

// Template represents a unified structure for different types of Kode templates
type Template struct {
	// Kind specifies the type of template (e.g., "ContainerTemplate", "ClusterContainerTemplate")
	Kind Kind `json:"kind" yaml:"kind"`

	// Name is the name of the template resource
	Name ObjectName `json:"name" yaml:"name"`

	// Namespace is the namespace of the template resource
	Namespace Namespace `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Port is the port to expose the kode instance
	Port Port `json:"port" yaml:"port"`

	// ContainerTemplateSpec is a reference to a ContainerTemplate or ClusterContainerTemplate
	ContainerTemplateSpec *ContainerTemplateSharedSpec `json:"container,omitempty" yaml:"container,omitempty"`
	
	// VirtualTemplateSpec is a reference to a VirtualTemplate or ClusterVirtualTemplate
	// VirtualTemplateSpec *VirtualTemplateSharedSpec `json:"virtual,omitempty" yaml:"virtual,omitempty"`

	// EntryPointSpecRef is a reference to an EntryPointSpec
	EntryPointRef *CrossNamespaceObjectReference `json:"entryPointSpecRef,omitempty" yaml:"entryPointSpecRef,omitempty"`

	EntryPointSpec *EntryPointSpec `json:"entryPoint,omitempty" yaml:"entryPoint,omitempty"`
}

type TemplateKind string

const (
	TemplateKindContainer        TemplateKind = "ContainerTemplate"
	TemplateKindClusterContainer TemplateKind = "ClusterContainerTemplate"
	TemplateKindVirtual          TemplateKind = "VirtualTemplate"
	TemplateKindClusterVirtual   TemplateKind = "ClusterVirtualTemplate"
)

type Phase string

const (
	// PhasePending indicates the initial state when a new resource is created.
	// The controller has acknowledged the resource but hasn't started processing it yet.
	PhasePending Phase = "Pending"
	// PhaseConfiguring indicates that the controller is actively setting up the resource.
	// This includes creating necessary Kubernetes resources, configuring storage, and applying user configurations.
	PhaseConfiguring Phase = "Configuring"
	// PhaseProvisioning indicates that all necessary resources have been created,
	// but the system is waiting for these resources to become fully operational.
	// This may include waiting for pods to be scheduled and reach a ready state or for any initialization processes to complete.
	PhaseProvisioning Phase = "Provisioning"
	// PhaseActive indicates that the resource is fully operational.
	// All associated Kubernetes resources are created and ready to serve requests.
	PhaseActive Phase = "Active"
	// PhaseUpdating indicates that the resource is in the process of being updated.
	// The controller is actively working on updating the environment with new configurations or resources.
	PhaseUpdating Phase = "Updating"
	// PhaseDeleting indicates the resource is being permanently removed.
	// The controller is in the process of deleting all associated Kubernetes resources.
	PhaseDeleting Phase = "Deleting"
	// PhaseFailed indicates that an error occurred during the lifecycle of the resource.
	// This could be during creation, updating, or management of the resource or its associated resources.
	// The controller will typically attempt to recover from this state automatically.
	PhaseFailed Phase = "Failed"
	// PhaseUnknown indicates that the resource is in an indeterminate state.
	// This may occur if the controller loses connection with the resource or encounters unexpected conditions.
	// The controller will attempt to reconcile and determine the correct state.
	PhaseUnknown Phase = "Unknown"
)

// Port for the service. Used by EnvoyProxy to expose the container. Defaults to '8000'.
// +kubebuilder:validation:Minimum=1
// +kubebuilder:default=8000
type Port int32

// Group refers to a Kubernetes Group. It must either be an empty string or a
// RFC 1123 subdomain.
//
// This validation is based off of the corresponding Kubernetes validation:
// https://github.com/kubernetes/apimachinery/blob/02cfb53916346d085a6c6c7c66f882e3c6b0eca6/pkg/util/validation/validation.go#L208
//
// Valid values include:
//
// * "" - empty string implies core Kubernetes API group
// * "gateway.networking.k8s.io"
// * "foo.example.com"
//
// Invalid values include:
//
// * "example.com/bar" - "/" is an invalid character
//
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern=`^$|^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
type Group string

// Kind refers to a Kubernetes Kind.
//
// Valid values include:
//
// * "Service"
// * "HTTPRoute"
//
// Invalid values include:
//
// * "invalid/kind" - "/" is an invalid character
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=63
// +kubebuilder:validation:Pattern=`^[a-zA-Z]([-a-zA-Z0-9]*[a-zA-Z0-9])?$`
type Kind string

// ObjectName refers to the name of a Kubernetes object.
// Object names can have a variety of forms, including RFC 1123 subdomains,
// RFC 1123 labels, or RFC 1035 labels.
//
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
type ObjectName string

// Namespace refers to a Kubernetes namespace. It must be a RFC 1123 label.
//
// This validation is based off of the corresponding Kubernetes validation:
// https://github.com/kubernetes/apimachinery/blob/02cfb53916346d085a6c6c7c66f882e3c6b0eca6/pkg/util/validation/validation.go#L187
//
// This is used for Namespace name validation here:
// https://github.com/kubernetes/apimachinery/blob/02cfb53916346d085a6c6c7c66f882e3c6b0eca6/pkg/api/validation/generic.go#L63
//
// Valid values include:
//
// * "example"
//
// Invalid values include:
//
// * "example.com" - "." is an invalid character
//
// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=63
type Namespace string

// Protocol is the protocol used by the service.
type Protocol string

const (
	ProtocolHTTP  Protocol = "http"
	ProtocolHTTPS Protocol = "https"
)
