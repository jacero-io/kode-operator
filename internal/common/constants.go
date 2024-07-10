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

package common

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	// General constants
	OperatorName     = "kode-operator"
	FinalizerName    = "kode.jacero.io/finalizer"
	PVCFinalizerName = "kode.jacero.io/pvc-finalizer"

	// Resource-related constants
	DefaultNamespace        = "default"
	KodeVolumeStorageName   = "kode-storage"
	DefaultLocalServicePort = 3000

	// Container-related constants
	EnvoyProxyContainerName = "envoy-proxy"
	EnvoyProxyRunAsUser     = 1111
	ProxyInitContainerName  = "proxy-init"
	ProxyInitContainerImage = "openpolicyagent/proxy_init:v8"
	BasicAuthContainerPort  = 9001

	// Basic Auth-related constants
	BasicAuthContainerImage = "ghcr.io/emil-jacero/grpc-basic-auth:latest"
	BasicAuthContainerName  = "basic-auth-service"

	// Time-related constants
	ReconcileTimeout            = 1 * time.Minute
	RequeueInterval             = 10 * time.Second
	DefaultInactiveAfterSeconds = 600
	DefaultRecycleAfterSeconds  = 28800

	// Label keys
	LabelAppName   = "app.kubernetes.io/name"
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelKodeName  = "kode.jacero.io/name"

	// Annotation keys
	AnnotationLastUpdated = "kode.jacero.io/last-updated"

	// These are the condition types that are used in the status of the Kode and EntryPoint resources
	// ConditionTypeReady indicates that the resource is fully operational and prepared to serve its intended purpose.
	ConditionTypeReady = "Ready"

	// ConditionTypeAvailable indicates that the resource is accessible and can actively serve requests or perform its function.
	ConditionTypeAvailable = "Available"

	// ConditionTypeProgressing indicates that the resource is actively working towards a desired state.
	ConditionTypeProgressing = "Progressing"

	// ConditionTypeDegraded indicates that the resource is operational but not functioning optimally or with full capabilities.
	ConditionTypeDegraded = "Degraded"

	// ConditionTypeError indicates that the resource has encountered an error state that requires attention.
	ConditionTypeError = "Error"

	// ConditionTypeConfigured indicates that the resource has been properly configured with all necessary settings.
	ConditionTypeConfigured = "Configured"

	// ConditionTypeCreating indicates that the resource is in the process of being created.
	ConditionTypeCreating = "Creating"

	// ConditionTypeCreated indicates that the resource has been successfully created.
	ConditionTypeCreated = "Created"

	// ConditionTypeRecycling indicates that the resource is in the process of being recycled or restarted.
	ConditionTypeRecycling = "Recycling"

	// ConditionTypeRecycled indicates that the resource has completed the recycling process.
	ConditionTypeRecycled = "Recycled"

	// ConditionTypeUpdating indicates that the resource is in the process of being updated.
	ConditionTypeUpdating = "Updating"

	// ConditionTypeDeleting indicates that the resource is in the process of being deleted.
	ConditionTypeDeleting = "Deleting"

	// Validation constants
	MinPasswordLength = 8
	MinUsernameLength = 3
	MaxUsernameLength = 256
)

var (
	// DefaultLabels are the base labels added to all resources created by the operator
	DefaultLabels = map[string]string{
		LabelManagedBy: OperatorName,
	}

	// DefaultAccessModes are the default access modes for PersistentVolumeClaims
	DefaultAccessModes = []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteOnce,
	}
)
