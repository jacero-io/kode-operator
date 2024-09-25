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
	event "github.com/jacero-io/kode-operator/internal/event"
)

func (r *EntryPointReconciler) ensureHTTPRoutes(ctx context.Context, entrypoint *kodev1alpha2.EntryPoint, kode *kodev1alpha2.Kode, config *common.EntryPointResourceConfig, kodeHostname kodev1alpha2.KodeHostname, kodeDomain kodev1alpha2.KodeDomain) (bool, error) {
	log := r.Log.WithName("HTTPRoutesEnsurer").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))

	ctx, cancel := common.ContextWithTimeout(ctx, 30) // 30 seconds timeout
	defer cancel()

	log.V(1).Info("Ensuring HTTPRoutes")

	var routes []*gwapiv1.HTTPRoute
	var httpsRouteName string

	// Construct HTTP Route
	// httpRoute, err := r.constructHTTPRoute(config, kode, true, kodeHostname, kodeDomain)
	// if err != nil {
	// 	return false, fmt.Errorf("failed to construct HTTP route: %v", err)
	// }
	// routes = append(routes, httpRoute)

	// Construct HTTPS Route
	httpsRoute, err := r.constructHTTPRoute(config, kode, false, kodeHostname, kodeDomain)
	if err != nil {
		return false, fmt.Errorf("failed to construct HTTPS route: %v", err)
	}
	routes = append(routes, httpsRoute)
	httpsRouteName = httpsRoute.Name

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
		err = r.EventManager.Record(ctx, kode, event.EventTypeNormal, event.ReasonCreated, message)
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
		err = r.EventManager.Record(ctx, kode, event.EventTypeNormal, event.ReasonCreated, message)
		if err != nil {
			log.Error(err, "Failed to record event")
			return created, err
		}

	}

	return created, nil
}

func (r *EntryPointReconciler) constructHTTPRoute(config *common.EntryPointResourceConfig, kode *kodev1alpha2.Kode, isRedirect bool, kodeHostname kodev1alpha2.KodeHostname, kodeDomain kodev1alpha2.KodeDomain) (*gwapiv1.HTTPRoute, error) {
	log := r.Log.WithName("HTTPRouteConstructor").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))
	log.V(1).Info("Constructing HTTPRoute", "isRedirect", isRedirect)

	// Construct ParentReference
	namespace := gwapiv1.Namespace(config.GatewayNamespace)
	gatewayName := gwapiv1.ObjectName(config.GatewayName)
	parentRef := gwapiv1.ParentReference{
		Name:      gatewayName,
		Namespace: &namespace,
	}

	var routeName string
	var rules []gwapiv1.HTTPRouteRule
	if kode.GetPort() == 0 {
		return nil, fmt.Errorf("kode port is not set")
	}
	kodePort := gwapiv1.PortNumber(kode.GetPort())

	if isRedirect {
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
	} else {
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
	}

	log.V(1).Info("Constructed HTTPRoute", "name", routeName, "hostname", kodeHostname, "domain", kodeDomain, "port", kodePort, "isRedirect", isRedirect, "protocol", config.Protocol, "rules", rules)

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

	policyName := fmt.Sprintf("%s-security-policy", httpsRouteName)

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

	if config.EntryPointSpec.AuthSpec == nil || config.EntryPointSpec.AuthSpec.SecurityPolicySpec == nil {
		return nil, fmt.Errorf("security policy spec is not defined")
	}

	securityPolicySpec := config.EntryPointSpec.AuthSpec.SecurityPolicySpec

	// Handle OIDC configuration
	if securityPolicySpec.OIDC != nil {
		spec.OIDC = securityPolicySpec.OIDC
		configureOIDC(spec.OIDC, config)
	}

	// Handle JWT configuration
	if securityPolicySpec.JWT != nil {
		spec.JWT = securityPolicySpec.JWT
		configureJWT(spec.JWT, config)
	}

	// Handle ExtAuth configuration
	if securityPolicySpec.ExtAuth != nil {
		spec.ExtAuth = securityPolicySpec.ExtAuth
		configureExtAuth(spec.ExtAuth, config)
	}

	// Handle BasicAuth configuration
	if securityPolicySpec.BasicAuth != nil {
		spec.BasicAuth = securityPolicySpec.BasicAuth
	}

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

func configureOIDC(oidc *egv1alpha1.OIDC, config *common.EntryPointResourceConfig) {
	if config.HasIdentityReference() && config.EntryPointSpec.AuthSpec.SecurityPolicySpec.ExtAuth != nil {
		if oidc.Scopes == nil {
			oidc.Scopes = []string{}
		}
		oidc.Scopes = append(oidc.Scopes, "openid", "profile")

		forwardAccessToken := true
		oidc.ForwardAccessToken = &forwardAccessToken
	}
}

func configureJWT(jwt *egv1alpha1.JWT, config *common.EntryPointResourceConfig) {
	if config.HasIdentityReference() && config.EntryPointSpec.AuthSpec.SecurityPolicySpec.ExtAuth != nil {
		identityRef := string(*config.IdentityReference)

		// Ensure we have at least one provider
		if len(jwt.Providers) == 0 {
			jwt.Providers = []egv1alpha1.JWTProvider{{
				Name: "default-provider",
			}}
		}

		// Configure the first provider (or the only one if there's just one)
		provider := &jwt.Providers[0]

		// Set up claim to header mapping
		provider.ClaimToHeaders = append(provider.ClaimToHeaders, egv1alpha1.ClaimToHeader{
			Header: "x-auth-user-id",
			Claim:  identityRef,
		})

		// Enable route recomputation
		recomputeRoute := true
		provider.RecomputeRoute = &recomputeRoute
	}
}

func configureExtAuth(extAuth *egv1alpha1.ExtAuth, config *common.EntryPointResourceConfig) {
	if config.HasIdentityReference() {
		// Ensure HeadersToExtAuth includes the necessary headers
		if extAuth.HeadersToExtAuth == nil {
			extAuth.HeadersToExtAuth = []string{}
		}
		extAuth.HeadersToExtAuth = append(extAuth.HeadersToExtAuth,
			"authorization",
			"x-forwarded-for",
			"x-forwarded-host",
			"x-forwarded-proto",
			"x-auth-user-id")

		// If using HTTP ExtAuth, configure HeadersToBackend
		if extAuth.HTTP != nil {
			if extAuth.HTTP.HeadersToBackend == nil {
				extAuth.HTTP.HeadersToBackend = []string{}
			}
			extAuth.HTTP.HeadersToBackend = append(extAuth.HTTP.HeadersToBackend, "x-auth-user-id")
		}

		// If using gRPC ExtAuth, you might want to add additional configuration here
		if extAuth.GRPC != nil {
			// Add gRPC-specific configuration if needed
		}
	}
}
