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

	// Status condition types
	ConditionTypeCreated  = "Created"
	ConditionTypeReady    = "Ready"
	ConditionTypeRecycled = "Recycled"
	ConditionTypeInactive = "Inactive"
	ConditionTypeError    = "Error"

	// Default values
	DefaultUser        = "abc"
	DefaultHome        = "/config"
	DefaultWorkspace   = "workspace"
	DefaultStorageSize = "1Gi"

	// Environment variable names
	EnvVarPassword = "PASSWORD"
	EnvVarUser     = "USER"
	EnvVarHome     = "HOME"
	EnvVarPUID     = "PUID"
	EnvVarPGID     = "PGID"
	EnvVarTZ       = "TZ"

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
