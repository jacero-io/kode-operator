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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// The reference to a template. Either a ContainerTemplate or VirtualTemplate.
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
	CommonStatus `json:",inline" yaml:",inline"`

	// The URL to access the Kode.
	KodeUrl KodeUrl `json:"kodeUrl,omitempty"`

	// The port to access the Kode.
	KodePort Port `json:"kodePort,omitempty"`

	// The URL to the icon for the Kode.
	KodeIconUrl KodeIconUrl `json:"iconUrl,omitempty"`

	// The runtime for the Kode.
	Runtime *Runtime `json:"runtime,omitempty"`

	// The timestamp when the last activity occurred.
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`

	// RetryCount keeps track of the number of retry attempts for failed states.
	RetryCount int `json:"retryCount,omitempty"`

	// DeletionCycle keeps track of the number of deletion cycles. This is used to determine if the resource is deleting.
	DeletionCycle int `json:"deletionCycle,omitempty"` // TODO: Remove this field and use RetryCount instead. Deleting and failure are not executing at the same time.
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the Kode"
// +kubebuilder:printcolumn:name="Runtime",type="string",JSONPath=".status.runtime.kodeRuntime",description="Runtime of the Kode"
// +kubebuilder:printcolumn:name="Runtime-Type",type="string",JSONPath=".status.runtime.type",description="Runtime type of the Kode"
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

type Runtime struct {
	// KodeRuntime is the runtime for the Kode resource. Can be one of 'container', 'virtual'.
	// +kubebuilder:validation:Enum=container;virtual
	Runtime KodeRuntime `json:"kodeRuntime"`

	// Type is the container runtime for Kode resource.
	Type RuntimeType `json:"type,omitempty"`
}

// KodeRuntime specifies the runtime for the Kode resource.
// Can be one of 'container', 'virtual'.
type KodeRuntime string

const (
	RuntimeContainer KodeRuntime = "container"
	RuntimeVirtual   KodeRuntime = "virtual"
)

// RuntimeType specifies the type of the runtime for the Kode resource.
type RuntimeType string

const (
	ContainerRuntimeContainerd RuntimeType = "containerd"
	ContainerRuntimeGvisor     RuntimeType = "gvisor"
)

type KodeHostname string

type KodeDomain string

type KodeUrl string

type KodePath string

type KodeIconUrl string

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
}

func (k *Kode) GetName() string {
	return k.Name
}

func (k *Kode) GetNamespace() string {
	return k.Namespace
}

func (k *Kode) GetPhase() Phase {
	return k.Status.Phase
}

func (k *Kode) SetPhase(phase Phase) {
	k.Status.Phase = phase
}

func (k *Kode) UpdateStatus(ctx context.Context, c client.Client) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of the Kode
		latestKode := &Kode{}
		if err := c.Get(ctx, client.ObjectKey{Name: k.Name, Namespace: k.Namespace}, latestKode); err != nil {
			return client.IgnoreNotFound(err)
		}

		// Create a patch
		patch := client.MergeFrom(latestKode.DeepCopy())

		// Update the status
		latestKode.Status = k.Status

		// When the Kode resource transitions from Failed to another state, reset the LastError and LastErrorTime fields
		if k.Status.Phase != latestKode.Status.Phase && latestKode.Status.Phase != PhaseFailed {
			k.Status.LastError = nil
			k.Status.LastErrorTime = nil
		}

		// Apply the patch
		return c.Status().Patch(ctx, latestKode, patch)
	})
}

func (k *Kode) SetCondition(conditionType constant.ConditionType, status metav1.ConditionStatus, reason, message string) {
	k.Status.SetCondition(conditionType, status, reason, message)
}

func (k *Kode) GetCondition(conditionType constant.ConditionType) *metav1.Condition {
	return k.Status.GetCondition(conditionType)
}

func (k *Kode) DeleteCondition(conditionType constant.ConditionType) {
	k.Status.DeleteCondition(conditionType)
}

func (k *Kode) GetFinalizer() string {
	return constant.KodeFinalizerName
}

func (k *Kode) AddFinalizer(ctx context.Context, c client.Client) error {
	// Fetch the latest version of the Kode
	latestKode := &Kode{}
	if err := c.Get(ctx, client.ObjectKey{Name: k.Name, Namespace: k.Namespace}, latestKode); err != nil {
		return client.IgnoreNotFound(err)
	}
	controllerutil.AddFinalizer(latestKode, k.GetFinalizer())
	if err := c.Update(ctx, latestKode); err != nil {
		return err
	}
	return nil
}

func (k *Kode) RemoveFinalizer(ctx context.Context, c client.Client) error {
	// Fetch the latest version of the Kode
	latestKode := &Kode{}
	if err := c.Get(ctx, client.ObjectKey{Name: k.Name, Namespace: k.Namespace}, latestKode); err != nil {
		return client.IgnoreNotFound(err)
	}
	controllerutil.RemoveFinalizer(latestKode, k.GetFinalizer())
	if err := c.Update(ctx, latestKode); err != nil {
		return err
	}
	return nil
}

func (k *Kode) GetSecretName() string {
	if k.Spec.Credentials != nil && k.Spec.Credentials.ExistingSecret != nil {
		return *k.Spec.Credentials.ExistingSecret
	} else { // If ExistingSecret is not specified, use Kode secret name
		return fmt.Sprintf("%s-auth", k.Name)
	}
}

func (k *Kode) GetServiceName() string {
	return k.Name + "-svc"
}

func (k *Kode) GetStatefulSetName() string {
	return k.Name
}

func (k *Kode) GetPVCName() string {
	if k.Spec.Storage != nil && k.Spec.Storage.ExistingVolumeClaim != nil {
		return *k.Spec.Storage.ExistingVolumeClaim
	} else {
		return k.Name + "-pvc"
	}
}

func (k *Kode) GetPort() Port {
	return k.Status.KodePort
}

func (k *Kode) SetRuntime(runtime Runtime, ctx context.Context, c client.Client) {
	k.Status.Runtime = &runtime
	k.UpdateStatus(ctx, c)
}

func (k *Kode) GetRuntime() *Runtime {
	return k.Status.Runtime
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
