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
	"fmt"
	"time"

	"github.com/jacero-io/kode-operator/internal/constant"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// The reference to a template. Either a ContainerTemplate, VirtualTemplate or TofuTemplate.
	// +kubebuilder:validation:Required
	TemplateRef CrossNamespaceObjectReference `json:"templateRef"`

	// Specifies the credentials for the service.
	Credentials *CredentialsSpec `json:"credentials,omitempty"`

	// The path to the directory for the user data. Defaults to '/config'.
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	Home *string `json:"home,omitempty"`

	// The user specified workspace directory (e.g. my-workspace).
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	Workspace *string `json:"workspace,omitempty"`

	// Specifies the storage configuration.
	Storage *KodeStorageSpec `json:"storage,omitempty"`

	// Specifies a git repository URL to get user configuration from.
	// +kubebuilder:validation:Pattern=`^(https?:\/\/)?([\w\.-]+@)?([\w\.-]+)(:\d+)?\/?([\w\.-]+)\/([\w\.-]+)(\.git)?(\/?|\#[\w\.\-_]+)?$|^oci:\/\/([\w\.-]+)(:\d+)?\/?([\w\.-\/]+)(@sha256:[a-fA-F0-9]{64})?$`
	// +kubebuilder:validation:Optional
	UserConfig *string `json:"userConfig,omitempty"`

	// Specifies if the container should run in privileged mode. Will only work if the KodeTemplate allows it. Only set to true if you know what you are doing.
	// +kubebuilder:default=false
	Privileged *bool `json:"privileged,omitempty"`

	// Specifies the OCI containers to be run as InitContainers. These containers can be used to prepare the workspace or run some setup scripts. It is an ordered list.
	InitPlugins []InitPluginSpec `json:"initPlugins,omitempty"`
}

// KodeStorageSpec defines the storage configuration
type KodeStorageSpec struct {
	// Specifies the access modes for the persistent volume.
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// Specifies the storage class name for the persistent volume.
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Specifies the resource requirements for the persistent volume.
	Resources *corev1.VolumeResourceRequirements `json:"resources,omitempty"`

	// Specifies if the volume should be kept when the kode is recycled. Defaults to false.
	// +kubebuilder:default=false
	KeepVolume *bool `json:"keepVolume,omitempty"`

	// Specifies an existing PersistentVolumeClaim to use instead of creating a new one.
	ExistingVolumeClaim *string `json:"existingVolumeClaim,omitempty"`
}

// KodeStatus defines the observed state of Kode
type KodeStatus struct {
	BaseSharedStatus `json:",inline" yaml:",inline"`

	// Phase represents the current phase of the Kode resource.
	Phase KodePhase `json:"phase"`

	// The URL to access the Kode.
	KodeUrl KodeUrl `json:"kodeUrl,omitempty"`

	// The port to access the Kode.
	KodePort Port `json:"kodePort,omitempty"`

	// The URL to the icon for the Kode.
	KodeIconUrl KodeIconUrl `json:"iconUrl,omitempty"`

	// The runtime for the kode. Can be one of 'pod', 'virtual', 'tofu'.
	Runtime KodeRuntime `json:"runtime,omitempty"`

	// The timestamp when the last activity occurred.
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`

	// RetryCount keeps track of the number of retry attempts for failed states.
	RetryCount int `json:"retryCount,omitempty"`

	// DeletionCycle keeps track of the number of deletion cycles. This is used to determine if the resource is deleting.
	DeletionCycle int `json:"deletionCycle,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the Kode"
// +kubebuilder:printcolumn:name="Runtime",type="string",JSONPath=".status.runtime",description="Runtime of the Kode"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.kodeUrl",description="URL to access the Kode"
// +kubebuilder:printcolumn:name="Template",type="string",JSONPath=".spec.templateRef.name",description="Template name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Kode is the Schema for the kodes API
type Kode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KodeSpec   `json:"spec,omitempty"`
	Status KodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KodeList contains a list of Kode
type KodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kode `json:"items"`
}

// KodePhase represents the current state of the Kode resource.
type KodePhase string

const (
	// KodePhasePending indicates the initial state when a new Kode resource is created.
	// The controller has acknowledged the resource but hasn't started processing it yet.
	KodePhasePending KodePhase = "Pending"

	// KodePhaseConfiguring indicates that the controller is actively setting up the Kode resource.
	// This includes creating necessary Kubernetes resources, configuring storage, and applying user configurations.
	KodePhaseConfiguring KodePhase = "Configuring"

	// KodePhaseProvisioning indicates that all necessary resources for the Kode have been created,
	// but the system is waiting for these resources to become fully operational.
	// This may include waiting for pods to be scheduled and reach a ready state or for any initialization processes to complete.
	KodePhaseProvisioning KodePhase = "Provisioning"

	// KodePhaseActive indicates that the Kode resource is fully operational.
	// All associated Kubernetes resources are created and ready to serve requests.
	// The Kode environment is accessible to users in this state.
	KodePhaseActive KodePhase = "Active"

	// KodePhaseInactive indicates that the Kode resource has been flagged for suspension.
	// This could be due to inactivity or to free up resources.
	// Some resources may be partially or fully removed to free up cluster resources.
	KodePhaseInactive KodePhase = "Inactive"

	// KodePhaseSuspending indicates that the Kode resource is in the process of being suspended.
	// The controller is actively working on stopping the Kode environment and preserving its state.
	KodePhaseSuspending KodePhase = "Suspending"

	// KodePhaseSuspended indicates that the Kode resource is in a suspended state.
	// The Kode environment is not running, but its configuration and data are preserved.
	// It can be resumed later without loss of user data or settings.
	KodePhaseSuspended KodePhase = "Suspended"

	// KodePhaseResuming indicates that the Kode resource is in the process of being reactivated from a suspended state.
	// The controller is recreating necessary resources and restoring the Kode environment to its previous active state.
	KodePhaseResuming KodePhase = "Resuming"

	// KodePhaseDeleting indicates the Kode resource is being permanently removed.
	// The controller is in the process of deleting all associated Kubernetes resources.
	KodePhaseDeleting KodePhase = "Deleting"

	// KodePhaseFailed indicates that an error occurred during the lifecycle of the Kode resource.
	// This could be during creation, updating, or management of the Kode or its associated resources.
	// The controller will typically attempt to recover from this state automatically.
	KodePhaseFailed KodePhase = "Failed"

	// KodePhaseUnknown indicates that the Kode resource is in an indeterminate state.
	// This may occur if the controller loses connection with the resource or encounters unexpected conditions.
	// The controller will attempt to reconcile and determine the correct state.
	KodePhaseUnknown KodePhase = "Unknown"
)

type KodeHostname string

type KodeDomain string

type KodeUrl string

type KodePath string

type KodeIconUrl string

type KodeRuntime string

const (
	RuntimeContainer KodeRuntime = "container"
	RuntimeVirtual   KodeRuntime = "virtual"
	RuntimeTofu      KodeRuntime = "tofu"
)

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
}

func (k *Kode) SetCondition(conditionType constant.ConditionType, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}

	// Update existing condition if it exists
	for i, condition := range k.Status.Conditions {
		if condition.Type == string(conditionType) {
			if condition.Status != status {
				k.Status.Conditions[i] = newCondition
			}
			return
		}
	}

	k.Status.Conditions = append(k.Status.Conditions, newCondition)
}

func (k *Kode) GetCondition(conditionType constant.ConditionType) *metav1.Condition {
	for _, condition := range k.Status.Conditions {
		if condition.Type == string(conditionType) {
			return &condition
		}
	}
	return nil
}

func (k *Kode) DeleteCondition(conditionType constant.ConditionType) {
	conditions := []metav1.Condition{}
	for _, condition := range k.Status.Conditions {
		if condition.Type != string(conditionType) {
			conditions = append(conditions, condition)
		}
	}
	k.Status.Conditions = conditions
}

func (k *Kode) SetRuntime() KodeRuntime {
	var runtime KodeRuntime
	if TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindContainerTemplate || TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindClusterContainerTemplate {
		runtime = RuntimeContainer
	} else if TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindVirtualTemplate || TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindClusterVirtualTemplate {
		runtime = RuntimeVirtual
	} else if TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindTofuTemplate || TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindClusterTofuTemplate {
		runtime = RuntimeTofu
	} else {
		return ""
	}
	return runtime
}

func (k *Kode) SetPhase(phase KodePhase) {
	k.Status.Phase = phase
}

func (k *Kode) UpdatePort(ctx context.Context, c client.Client, port Port) error {
	patch := client.MergeFrom(k.DeepCopy())
	k.Status.KodePort = port

	err := c.Status().Patch(ctx, k, patch)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kode) UpdateUrl(ctx context.Context, c client.Client, kodeUrl KodeUrl) error {
	patch := client.MergeFrom(k.DeepCopy())
	k.Status.KodeUrl = kodeUrl

	err := c.Status().Patch(ctx, k, patch)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kode) GetSecretName() string {
	return fmt.Sprintf("%s-auth", k.Name)
}

func (k *Kode) GetServiceName() string {
	return k.Name + "-svc"
}

func (k *Kode) GetPVCName() string {
	return k.Name + "-pvc"
}

func (k *Kode) GetPort() Port {
	return k.Status.KodePort
}

func (k *Kode) IsInactiveFor(duration time.Duration) bool {
	if k.Status.LastActivityTime == nil {
		return false
	}
	return time.Since(k.Status.LastActivityTime.Time) > duration
}

func (k *Kode) GenerateKodeUrlForEntryPoint(
	routingType RoutingType,
	domain string,
	name string,
	protocol Protocol,
) (KodeHostname, KodeDomain, KodeUrl, KodePath, error) {

	var kodeDomain KodeDomain
	var kodePath KodePath
	var kodeUrl KodeUrl

	if routingType == RoutingTypeSubdomain {
		kodeDomain = KodeDomain(fmt.Sprintf("%s.%s", name, domain)) // eg. my-workspace.kode.dev
		kodePath = "/"
		kodeUrl = KodeUrl(fmt.Sprintf("%s://%s%s", protocol, kodeDomain, kodePath)) // eg. https://my-workspace.kode.dev/

		return KodeHostname(name), kodeDomain, kodeUrl, kodePath, nil

	} else if routingType == RoutingTypePath {
		kodeDomain = KodeDomain(domain)                                             // eg. kode.dev
		kodePath = KodePath(fmt.Sprintf("/%s", name))                               // eg. /my-workspace
		kodeUrl = KodeUrl(fmt.Sprintf("%s://%s%s", protocol, kodeDomain, kodePath)) // eg. https://kode.dev/my-workspace

		return KodeHostname(name), kodeDomain, kodeUrl, kodePath, nil

	} else {
		return "", "", "", "", fmt.Errorf("unsupported routing type: %s", routingType)
	}
}

func (s *KodeStorageSpec) IsEmpty() bool {
	if s == nil {
		return true
	}
	return len(s.AccessModes) == 0 &&
		s.StorageClassName == nil &&
		s.ExistingVolumeClaim == nil &&
		(s.Resources == nil || s.Resources.Requests == nil || s.Resources.Requests.Storage().IsZero())
}
