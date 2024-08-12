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

package kode

import (
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

// returns the name of the PersistentVolumeClaim for the Kode instance
func GetPVCName(kode *kodev1alpha2.Kode) string {
	return kode.Name + "-pvc"
}

// returns the name of the Kode service
func GetServiceName(kode *kodev1alpha2.Kode) string {
	return kode.Name + "-svc"
}
