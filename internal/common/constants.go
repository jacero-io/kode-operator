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

package common

const (
	// General constants
	KodeFinalizerName       = "kode.jacero.io/kode-finalizer"
	EntryPointFinalizerName = "kode.jacero.io/entrypoint-finalizer"
	PVCFinalizerName        = "kode.jacero.io/kode-pvc-finalizer"

	// Resource related constants
	KodeVolumeStorageName = "kode-storage"

	// Credential related constants
	Username = "abc"

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

	// ConditionTypeHTTPRouteAvailable indicates that the HTTP route is available and can be accessed.
	ConditionTypeHTTPRouteAvailable = "HTTPRouteAvailable"

	// ConditionTypeHTTPSRouteAvailable indicates that the HTTPS route is available and can be accessed.
	ConditionTypeHTTPSRouteAvailable = "HTTPSRouteAvailable"
)
