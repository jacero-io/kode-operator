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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// The reference to a template. Either a PodTemplate, VirtualTemplate or TofuTemplate.
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
	IconUrl IconUrl `json:"iconUrl,omitempty"`

	// The runtime for the kode. Can be one of 'pod', 'virtual', 'tofu'.
	Runtime Runtime `json:"runtime,omitempty"`

	// The timestamp when the last activity occurred.
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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

type TemplateKind string

const (
	TemplateKindPodTemplate            TemplateKind = "PodTemplate"
	TemplateKindClusterPodTemplate     TemplateKind = "ClusterPodTemplate"
	TemplateKindVirtualTemplate        TemplateKind = "VirtualTemplate"
	TemplateKindClusterVirtualTemplate TemplateKind = "ClusterVirtualTemplate"
	TemplateKindTofuTemplate           TemplateKind = "TofuTemplate"
	TemplateKindClusterTofuTemplate    TemplateKind = "ClusterTofuTemplate"
)

// KodePhase represents the current phase of the Kode resource.
type KodePhase string

const (
	// KodePhaseCreating indicates that the Kode resource is in the process of being created.
	// This includes the initial setup and creation of associated Kubernetes resources.
	KodePhaseCreating KodePhase = "Creating"

	// KodePhaseCreated indicates that all the necessary Kubernetes resources for the Kode
	// have been successfully created, but may not yet be fully operational.
	KodePhaseCreated KodePhase = "Created"

	// KodePhaseFailed indicates that an error occurred during the creation, updating,
	// or management of the Kode resource or its associated Kubernetes resources.
	KodePhaseFailed KodePhase = "Failed"

	// KodePhasePending indicates that the Kode resource has been created, but is waiting
	// for its associated resources to become ready or for external dependencies to be met.
	KodePhasePending KodePhase = "Pending"

	// KodePhaseActive indicates that the Kode resource and all its associated Kubernetes
	// resources are fully operational and ready to serve requests.
	KodePhaseActive KodePhase = "Active"

	// KodePhaseInactive indicates that the Kode resource has been marked for deletion
	// due to inactivity. Resources may be partially or fully removed in this state.
	KodePhaseInactive KodePhase = "Inactive"

	// KodePhaseRecycling indicates that the Kode resource is in the process of being
	// cleaned up and its resources are being partially or fully deleted.
	KodePhaseRecycling KodePhase = "Recycling"

	// KodePhaseRecycled indicates that the Kode resource has been fully recycled,
	// with all associated resources either partially or fully deleted.
	KodePhaseRecycled KodePhase = "Recycled"
)

type KodeHostname string

type KodeDomain string

type KodeUrl string

type IconUrl string

type Runtime string

const (
	RuntimePod     Runtime = "pod"
	RuntimeVirtual Runtime = "virtual"
	RuntimeTofu    Runtime = "tofu"
)

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
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

func (k *Kode) IsActive() bool {
	return k.Status.Phase == KodePhaseActive
}

func (k *Kode) IsRecycled() bool {
	return k.Status.Phase == KodePhaseRecycled
}

func (k *Kode) SetCondition(conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}

	for i, condition := range k.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Status != status {
				k.Status.Conditions[i] = newCondition
			}
			return
		}
	}

	k.Status.Conditions = append(k.Status.Conditions, newCondition)
}

func (k *Kode) GetCondition(conditionType string) *metav1.Condition {
	for _, condition := range k.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func (k *Kode) IsInactiveFor(duration time.Duration) bool {
	if k.Status.LastActivityTime == nil {
		return false
	}
	return time.Since(k.Status.LastActivityTime.Time) > duration
}

func (k *Kode) SetRuntime() Runtime {
	var runtime Runtime
	if TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindPodTemplate || TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindClusterPodTemplate {
		runtime = RuntimePod
	} else if TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindVirtualTemplate || TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindClusterVirtualTemplate {
		runtime = RuntimeVirtual
	} else if TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindTofuTemplate || TemplateKind(k.Spec.TemplateRef.Kind) == TemplateKindClusterTofuTemplate {
		runtime = RuntimeTofu
	} else {
		return ""
	}
	return runtime
}

func (k *Kode) UpdateKodePort(ctx context.Context, c client.Client, port Port) error {
	patch := client.MergeFrom(k.DeepCopy())
	k.Status.KodePort = port

	err := c.Status().Patch(ctx, k, patch)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kode) UpdateKodeUrl(ctx context.Context, c client.Client, kodeUrl KodeUrl) error {
	patch := client.MergeFrom(k.DeepCopy())
	k.Status.KodeUrl = kodeUrl

	err := c.Status().Patch(ctx, k, patch)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kode) GenerateKodeUrlForEntryPoint(
	routingType RoutingType,
	domain string,
	name string,
	namespace string,
	protocol Protocol,
) (KodeHostname, Namespace, KodeDomain, KodeUrl, error) {

	var kodeUrl KodeUrl
	var kodeDomain KodeDomain

	if routingType == RoutingTypeSubdomain {
		kodeDomain = KodeDomain(fmt.Sprintf("%s.%s.%s", name, namespace, domain))
		kodeUrl = KodeUrl(fmt.Sprintf("%s://%s", protocol, kodeDomain))

		return KodeHostname(name), Namespace(namespace), kodeDomain, kodeUrl, nil

	} else if routingType == RoutingTypePath {
		kodeDomain = KodeDomain(fmt.Sprintf("%s/%s/%s", domain, namespace, name))
		kodeUrl = KodeUrl(fmt.Sprintf("%s://%s", protocol, kodeDomain))

		return KodeHostname(name), Namespace(namespace), kodeDomain, kodeUrl, nil

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
