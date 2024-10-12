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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	event "github.com/jacero-io/kode-operator/internal/event"
	"github.com/jacero-io/kode-operator/pkg/constant"
)

func (r *EntryPointReconciler) ensureHTTPRoutes(ctx context.Context, entrypoint *kodev1alpha2.EntryPoint, kode *kodev1alpha2.Kode, config *common.EntryPointResourceConfig, kodeHostname kodev1alpha2.KodeHostname, kodeDomain kodev1alpha2.KodeDomain) (ctrl.Result, error) {
	log := r.Log.WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))
	log.V(1).Info("Ensuring HTTPRoutes")

	httpsRoute, err := r.constructHTTPRoute(config, kode, false, kodeHostname, kodeDomain)
	if err != nil {
		log.Error(err, "Failed to construct HTTPS route")
		entrypoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "HTTPRouteConstructionFailed", fmt.Sprintf("Failed to construct HTTPS route: %v", err))
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateRoute(ctx, entrypoint, kode, httpsRoute); err != nil {
		log.Error(err, "Failed to create or update HTTPRoute")
		entrypoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "HTTPRouteCreationFailed", fmt.Sprintf("Failed to create or update HTTPRoute: %v", err))
		return ctrl.Result{}, err
	}

	if config.EntryPointSpec.AuthSpec != nil && config.EntryPointSpec.AuthSpec.SecurityPolicySpec != nil && config.Protocol == kodev1alpha2.ProtocolHTTPS {
		if err := r.createOrUpdateSecurityPolicy(ctx, entrypoint, kode, config, httpsRoute.Name); err != nil {
			log.Error(err, "Failed to create or update SecurityPolicy")
			entrypoint.SetCondition(constant.ConditionTypeReady, metav1.ConditionFalse, "SecurityPolicyCreationFailed", fmt.Sprintf("Failed to create or update SecurityPolicy: %v", err))
			return ctrl.Result{}, err
		}
	}

	// Update conditions
	kode.SetCondition(constant.ConditionTypeReady, metav1.ConditionTrue, "HTTPRoutesEnsured", "HTTPRoutes have been successfully ensured")
	kode.SetCondition(constant.ConditionTypeAvailable, metav1.ConditionTrue, "HTTPRoutesAvailable", "HTTPRoutes are available")
	kode.SetCondition(constant.ConditionTypeProgressing, metav1.ConditionFalse, "HTTPRoutesReady", "HTTPRoutes are ready")

	// Clear any previous error
	kode.Status.LastError = nil
	kode.Status.LastErrorTime = nil

	err = kode.UpdateStatus(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to update Kode status")
		return ctrl.Result{}, err
	}

	log.Info("HTTPRoutes ensured successfully")
	return ctrl.Result{}, nil
}

func (r *EntryPointReconciler) createOrUpdateRoute(ctx context.Context, entrypoint *kodev1alpha2.EntryPoint, kode *kodev1alpha2.Kode, route *gwapiv1.HTTPRoute) error {
	result, err := r.Resource.CreateOrPatch(ctx, route, func() error {
		return controllerutil.SetControllerReference(entrypoint, route, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to ensure HTTPRoute: %w", err)
	}

	var eventReason event.EventReason
	var message string
	switch result {
	case controllerutil.OperationResultCreated:
		eventReason = event.ReasonCreated
		message = fmt.Sprintf("HTTPRoute created, %s", route.Name)
	case controllerutil.OperationResultUpdated:
		eventReason = event.ReasonUpdated
		message = fmt.Sprintf("HTTPRoute updated, %s", route.Name)
	case controllerutil.OperationResultNone:
		// No changes were made, so we don't need to record an event
		return nil
	}

	if err := r.Event.Record(ctx, kode, event.EventTypeNormal, eventReason, message); err != nil {
		r.Log.Error(err, "Failed to record event")
	}

	return nil
}

func (r *EntryPointReconciler) createOrUpdateSecurityPolicy(ctx context.Context, entrypoint *kodev1alpha2.EntryPoint, kode *kodev1alpha2.Kode, config *common.EntryPointResourceConfig, httpsRouteName string) error {
	policy, err := r.constructSecurityPolicy(config, kode, httpsRouteName)
	if err != nil {
		return fmt.Errorf("failed to construct SecurityPolicy: %w", err)
	}

	result, err := r.Resource.CreateOrPatch(ctx, policy, func() error {
		return controllerutil.SetControllerReference(entrypoint, policy, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to ensure SecurityPolicy: %w", err)
	}

	var eventReason event.EventReason
	var message string
	switch result {
	case controllerutil.OperationResultCreated:
		eventReason = event.ReasonCreated
		message = fmt.Sprintf("SecurityPolicy created, %s", policy.Name)
	case controllerutil.OperationResultUpdated:
		eventReason = event.ReasonUpdated
		message = fmt.Sprintf("SecurityPolicy updated, %s", policy.Name)
	case controllerutil.OperationResultNone:
		// No changes were made, so we don't need to record an event
		return nil
	}

	if err := r.Event.Record(ctx, kode, event.EventTypeNormal, eventReason, message); err != nil {
		r.Log.Error(err, "Failed to record event")
	}

	return nil
}

func (r *EntryPointReconciler) constructHTTPRoute(config *common.EntryPointResourceConfig, kode *kodev1alpha2.Kode, isRedirect bool, kodeHostname kodev1alpha2.KodeHostname, kodeDomain kodev1alpha2.KodeDomain) (*gwapiv1.HTTPRoute, error) {
	log := r.Log.WithName("HTTPRouteConstructor").WithValues("entrypoint", common.ObjectKeyFromConfig(config.CommonConfig))
	log.Info("Constructing HTTPRoute", "isRedirect", isRedirect)

	kodePort := gwapiv1.PortNumber(kode.GetPort())
	if kodePort == 0 {
		return nil, fmt.Errorf("kode port is not set")
	}

	parentRef := constructParentReference(config)
	rules := constructHTTPRouteRules(isRedirect, config, kode, kodePort)
	routeName := constructRouteName(kode, isRedirect)

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

func constructParentReference(config *common.EntryPointResourceConfig) gwapiv1.ParentReference {
	namespace := gwapiv1.Namespace(config.GatewayNamespace)
	gatewayName := gwapiv1.ObjectName(config.GatewayName)
	return gwapiv1.ParentReference{
		Name:      gatewayName,
		Namespace: &namespace,
	}
}

func constructHTTPRouteRules(isRedirect bool, config *common.EntryPointResourceConfig, kode *kodev1alpha2.Kode, kodePort gwapiv1.PortNumber) []gwapiv1.HTTPRouteRule {
	if isRedirect {
		return constructRedirectRule(config)
	}
	return constructBackendRule(kode, kodePort)
}

func constructRedirectRule(config *common.EntryPointResourceConfig) []gwapiv1.HTTPRouteRule {
	httpsScheme := string(config.Protocol)
	return []gwapiv1.HTTPRouteRule{{
		Filters: []gwapiv1.HTTPRouteFilter{{
			Type: gwapiv1.HTTPRouteFilterRequestRedirect,
			RequestRedirect: &gwapiv1.HTTPRequestRedirectFilter{
				Scheme: &httpsScheme,
			},
		}},
	}}
}

func constructBackendRule(kode *kodev1alpha2.Kode, kodePort gwapiv1.PortNumber) []gwapiv1.HTTPRouteRule {
	pathMatchType := gwapiv1.PathMatchPathPrefix
	pathString := "/"
	service := gwapiv1.Kind("Service")
	kodeServiceName := kode.GetServiceName()

	return []gwapiv1.HTTPRouteRule{{
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

func constructRouteName(kode *kodev1alpha2.Kode, isRedirect bool) string {
	if isRedirect {
		return fmt.Sprintf("%s-tls-redirect", kode.Name)
	}
	return kode.Name
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
