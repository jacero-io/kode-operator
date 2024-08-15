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

	// egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *EntryPointReconciler) ensureHTTPRoute(ctx context.Context, config *common.EntryPointResourceConfig, entrypoint *kodev1alpha2.ClusterEntryPoint, kode kodev1alpha2.Kode) error {
    log := r.Log.WithName("HTTPRoutesEnsurer").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

    ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
    defer cancel()

    log.V(1).Info("Ensuring HTTPRoutes")

    httpRoutes := &gwapiv1.HTTPRoute{
        ObjectMeta: metav1.ObjectMeta{
            Name:      config.CommonConfig.Name,
            Namespace: config.CommonConfig.Namespace,
            Labels:   config.CommonConfig.Labels,
        },
    }

    err := r.ResourceManager.CreateOrPatch(ctx, httpRoutes, func() error {
        constructedHTTPRoutes, err := r.constructHTTPRoutesSpec(config, entrypoint, kode)
        if err != nil {
            return fmt.Errorf("failed to construct HTTPRoutes spec: %v", err)
        }

        httpRoutes.Spec = constructedHTTPRoutes.Spec
        httpRoutes.ObjectMeta.Labels = constructedHTTPRoutes.ObjectMeta.Labels

        return controllerutil.SetControllerReference(entrypoint, httpRoutes, r.Scheme)
    })
    if err != nil {
        return fmt.Errorf("failed to ensure HTTPRoutes: %v", err)
    }

    return nil
}

func (r *EntryPointReconciler) constructHTTPRoutesSpec(config *common.EntryPointResourceConfig, entrypoint *kodev1alpha2.ClusterEntryPoint, kode kodev1alpha2.Kode) (*gwapiv1.HTTPRoute, error) {
    log := r.Log.WithName("HTTPRoutesSpecConstructor").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

    log.V(1).Info("Constructing HTTPRoutes spec")

    httpRoutes := &gwapiv1.HTTPRoute{
        Spec: gwapiv1.HTTPRouteSpec{
            Rules: []gwapiv1.HTTPRouteRule{},
        },
    }


    // Construct HTTPRouteRules
    // for _, ruleConfig := range config.HTTPRouteRules {
    //     rule := gwapiv1.HTTPRouteRule{
    //         Matches: []gwapiv1.HTTPMatch{
    //             {
    //                 "uri": &gwapiv1.HTTPPathMatch{
    //                     Type:  gwapiv1.PathMatchPrefix,
    //                     Value: ruleConfig.Path,
    //                 },
    //             },
    //         },
    //         ForwardTo: []gwapiv1.HTTPRouteForwardTo{
    //             {
    //                 ServiceName: ruleConfig.ServiceName,
    //                 Port:        ruleConfig.ServicePort,
    //             },
    //         },
    //     }

    //     httpRoutes.Spec.Rules = append(httpRoutes.Spec.Rules, rule)
    // }

    log.V(1).Info("Constructed HTTPRoutes spec", "httpRoutes", httpRoutes)

    return httpRoutes, nil

}