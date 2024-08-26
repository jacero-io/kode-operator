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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"github.com/jacero-io/kode-operator/internal/events"
)

func (r *EntryPointReconciler) ensureHTTPRoutes(ctx context.Context, entrypoint *kodev1alpha2.EntryPoint, kode *kodev1alpha2.Kode, config *common.EntryPointResourceConfig, kodeHostname kodev1alpha2.KodeHostname, kodeDomain kodev1alpha2.KodeDomain) (bool, error) {
	log := r.Log.WithName("HTTPRoutesEnsurer").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring HTTPRoutes")

	var routes []*gwapiv1.HTTPRoute
	var httpsRouteName string

	// Construct HTTP Route
	httpRoute, err := r.constructHTTPRoute(config, kode, false, kodeHostname, kodeDomain)
	if err != nil {
		return false, fmt.Errorf("failed to construct HTTP route: %v", err)
	}
	routes = append(routes, httpRoute)

	// Construct HTTPS Route
	if config.IsHTTPS() {
		httpsRoute, err := r.constructHTTPRoute(config, kode, true, kodeHostname, kodeDomain)
		if err != nil {
			return false, fmt.Errorf("failed to construct HTTPS route: %v", err)
		}
		routes = append(routes, httpsRoute)
		httpsRouteName = httpsRoute.Name
	}

	created := false // Flag to indicate if any HTTPRoute was created

	for _, route := range routes {
		err := r.ResourceManager.CreateOrPatch(ctx, route, func() error {
			return controllerutil.SetControllerReference(entrypoint, route, r.Scheme)
		})
		if err != nil {
			return false, fmt.Errorf("failed to ensure HTTPRoute: %v", err)
		}

		created = true

		// Update EntryPoint status on the Kode resource
		entrypointStatusErr := r.updatePhaseActive(ctx, entrypoint)
		if entrypointStatusErr != nil {
			return false, fmt.Errorf("failed to update EntryPoint status: %v", entrypointStatusErr)
		}

		// Record event for created HTTPRoute on the Kode resource
		message := fmt.Sprintf("HTTPRoute created, %s", route.Name)
		err = r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonCreated, message)
		if err != nil {
			log.Error(err, "Failed to record event")
			return created, err
		}
	}

	// Construct Security Policy
	if config.EntryPointSpec.AuthSpec != nil && config.EntryPointSpec.AuthSpec.SecurityPolicySpec != nil && config.Protocol == kodev1alpha2.ProtocolHTTPS {

		policy := &egv1alpha1.SecurityPolicy{}
		policy, err := r.constructSecurityPolicy(config, kode, httpsRouteName)
		if err != nil {
			return false, fmt.Errorf("failed to construct SecurityPolicy: %v", err)
		}

		err = r.ResourceManager.CreateOrPatch(ctx, policy, func() error {
			return controllerutil.SetControllerReference(entrypoint, policy, r.Scheme)
		})
		if err != nil {
			return false, fmt.Errorf("failed to ensure SecurityPolicy: %v", err)
		}

		created = true

		// Update EntryPoint status on the Kode resource
		entrypointStatusErr := r.updatePhaseActive(ctx, entrypoint)
		if entrypointStatusErr != nil {
			return false, fmt.Errorf("failed to update EntryPoint status: %v", entrypointStatusErr)
		}

		// Record event for created SecurityPolicy on the Kode resource
		message := fmt.Sprintf("SecurityPolicy created, %s", policy.Name)
		err = r.EventManager.Record(ctx, kode, events.EventTypeNormal, events.ReasonCreated, message)
		if err != nil {
			log.Error(err, "Failed to record event")
			return created, err
		}

	}

	return created, nil
}

func (r *EntryPointReconciler) constructHTTPRoute(config *common.EntryPointResourceConfig, kode *kodev1alpha2.Kode, isHTTPS bool, kodeHostname kodev1alpha2.KodeHostname, kodeDomain kodev1alpha2.KodeDomain) (*gwapiv1.HTTPRoute, error) {
	log := r.Log.WithName("HTTPRouteConstructor").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))
	log.V(1).Info("Constructing HTTPRoute", "isHTTPS", isHTTPS)

	namespace := gwapiv1.Namespace(config.GatewayNamespace)
	parentRef := gwapiv1.ParentReference{
		Name:      gwapiv1.ObjectName(config.GatewayName),
		Namespace: &namespace,
	}

	var routeName string
	var rules []gwapiv1.HTTPRouteRule
	if kode.GetPort() == 0 {
		return nil, fmt.Errorf("kode port is not set")
	}
	kodePort := gwapiv1.PortNumber(kode.GetPort())

	if isHTTPS {
		pathMatchType := gwapiv1.PathMatchPathPrefix
		pathString := "/"
		service := gwapiv1.Kind("Service")
		kodeServiceName := kode.GetServiceName()

		log.V(1).Info("HTTPRoute details", "name", kodeServiceName, "port", kodePort)

		routeName = kode.Name

		if kodePort > 0 {
			rules = []gwapiv1.HTTPRouteRule{{
				Matches: []gwapiv1.HTTPRouteMatch{{
					Path: &gwapiv1.HTTPPathMatch{
						Type:  &pathMatchType,
						Value: &pathString,
					},
				}},
				BackendRefs: []gwapiv1.HTTPBackendRef{{
					BackendRef: gwapiv1.BackendRef{
						BackendObjectReference: gwapiv1.BackendObjectReference{
							Kind: (*gwapiv1.Kind)(&service),
							Name: gwapiv1.ObjectName(kodeServiceName),
							Port: &kodePort,
						},
					},
				}},
			}}
		}
	} else {
		routeName = fmt.Sprintf("%s-tls-redirect", kode.Name)
		httpsScheme := string(config.Protocol)

		rules = []gwapiv1.HTTPRouteRule{{
			Filters: []gwapiv1.HTTPRouteFilter{{
				Type: gwapiv1.HTTPRouteFilterRequestRedirect,
				RequestRedirect: &gwapiv1.HTTPRequestRedirectFilter{
					Scheme: &httpsScheme,
				},
			}},
		}}
	}

	log.Info("Constructed HTTPRoute", "name", routeName, "hostname", kodeHostname, "rules", rules)

	return &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
		Spec: gwapiv1.HTTPRouteSpec{
			CommonRouteSpec: gwapiv1.CommonRouteSpec{ParentRefs: []gwapiv1.ParentReference{parentRef}},
			Hostnames:       []gwapiv1.Hostname{gwapiv1.Hostname(kodeDomain)},
			Rules:           rules,
		},
	}, nil
}

func (r *EntryPointReconciler) constructSecurityPolicy(config *common.EntryPointResourceConfig, kode *kodev1alpha2.Kode, httpsRouteName string) (*egv1alpha1.SecurityPolicy, error) {
	log := r.Log.WithName("SecurityPolicyConstructor").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))
	log.V(1).Info("Constructing SecurityPolicy")

	var policyName string
	var policyType string

	targetRef := egv1alpha1.PolicyTargetReferences{
		TargetRef: &gwapiv1a2.LocalPolicyTargetReferenceWithSectionName{
			LocalPolicyTargetReference: gwapiv1a2.LocalPolicyTargetReference{
				Group: gwapiv1.Group("gateway.networking.k8s.io"),
				Kind:  gwapiv1.Kind("HTTPRoute"),
				Name:  gwapiv1.ObjectName(httpsRouteName),
			},
		},
	}

	spec := &egv1alpha1.SecurityPolicySpec{
		PolicyTargetReferences: targetRef,
	}

	if config.EntryPointSpec.AuthSpec != nil && config.EntryPointSpec.AuthSpec.SecurityPolicySpec != nil {
		securityPolicySpec := config.EntryPointSpec.AuthSpec.SecurityPolicySpec

		switch {
		case securityPolicySpec.OIDC != nil:
			policyType = "oidc"
			spec.OIDC = securityPolicySpec.OIDC
		case securityPolicySpec.BasicAuth != nil:
			policyType = "basic-auth"
			spec.BasicAuth = securityPolicySpec.BasicAuth
		case securityPolicySpec.JWT != nil:
			policyType = "jwt"
			spec.JWT = securityPolicySpec.JWT
		case securityPolicySpec.ExtAuth != nil:
			policyType = "ext-auth"
			spec.ExtAuth = securityPolicySpec.ExtAuth
		default:
			return nil, fmt.Errorf("no valid security policy type specified")
		}

		// Handle IdentityReference if present
		if config.HasIdentityReference() {
			// TODO: Implement logic to handle IdentityReference
		}
	} else {
		return nil, fmt.Errorf("security policy spec is not defined")
	}

	policyName = fmt.Sprintf("%s-%s-policy", httpsRouteName, policyType)

	log.Info("Constructed Policy", "name", policyName)

	return &egv1alpha1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: config.CommonConfig.Namespace,
			Labels:    config.CommonConfig.Labels,
		},
		Spec: *spec,
	}, nil
}
