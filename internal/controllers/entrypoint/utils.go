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

package entrypoint

import (
	"context"
	"fmt"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
)

func (r *EntryPointReconciler) findEntryPointForKode(ctx context.Context, kode *kodev1alpha2.Kode) (*kodev1alpha2.EntryPoint, error) {
	// Fetch the template object
	template, err := r.Template.Fetch(ctx, kode.Spec.TemplateRef)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	var entryPointRef *kodev1alpha2.CrossNamespaceObjectReference

	switch template.Kind {
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindContainer),
		kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterContainer):
		if template.ContainerTemplateSpec == nil {
			return nil, fmt.Errorf("invalid ContainerTemplate: missing ContainerTemplateSpec")
		}
		entryPointRef = template.ContainerTemplateSpec.CommonSpec.EntryPointRef
	default:
		return nil, fmt.Errorf("unknown template kind: %s", template.Kind)
	}

	if entryPointRef == nil {
		return nil, fmt.Errorf("missing EntryPointRef in template")
	}

	// Fetch the latest EntryPoint object
	latestEntryPoint, err := common.GetLatestEntryPoint(ctx, r.Client, string(entryPointRef.Name), string(*entryPointRef.Namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get EntryPoint: %w", err)
	}

	return latestEntryPoint, nil
}
