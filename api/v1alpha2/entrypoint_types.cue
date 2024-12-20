package v1alpha2

import (
    egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// EntryPointSpec defines the desired state of EntryPoint
#EntryPointSpec: {
    // The way the Kode resource is accessed by the user. It could be subdomain or path.
    routingType: #RoutingType | *"path"

    // The domain name to use either as a suffix in the case of Type=domain or as a prefix/domain in the case of Type=path.
    baseDomain: string & =~"^([a-zA-Z0-9_]+\\.)*[a-zA-Z0-9_]+$"

    // GatewaySpec defines the GatewaySpec for the EntryPoint.
    gatewaySpec?: #GatewaySpec

    // AuthSpec defines the AuthSpec for the EntryPoint.
    authSpec?: #AuthSpec
}

#GatewaySpec: {
    // Reference to an existing Gateway to use for the EntryPoint.
    existingGatewayRef: #CrossNamespaceObjectReference

    // Protocol defines the protocol to use for the HTTPRoutes. Can be either "http" or "https".
    protocol: "http" | *"https"
}

#AuthSpec: {
    // The Envoy Gateway SecurityPolicy to use for the authentication.
    authType: #AuthType | *"none"

    // Defines the SecurityPolicies to be applied to the Route.
    securityPolicySpec?: #SecurityPolicySpec

    // Reference to a field in the JWT token of OIDC or JWT.
    identityReference?: #IdentityReference
}

#SecurityPolicySpec: {
    // ExtAuth defines the configuration for External Authorization.
    extAuth?: egv1alpha1.#ExtAuth
}

// EntryPointStatus defines the observed state of EntryPoint
#EntryPointStatus: {
    #CommonStatus

    // RetryCount keeps track of the number of retry attempts for failed states.
    retryCount?: int

    // DeletionCycle keeps track of the number of deletion cycles.
    deletionCycle?: int
}

// EntryPoint is the Schema for the entrypoints API
#EntryPoint: {
    metav1.#TypeMeta
    metadata: metav1.#ObjectMeta
    spec: #EntryPointSpec
    status?: #EntryPointStatus
}

// EntryPointList contains a list of EntryPoint
#EntryPointList: {
    metav1.#TypeMeta
    metadata: metav1.#ListMeta
    items: [...#EntryPoint]
}

// IdentityReference is a reference to a field in the JWT token of OIDC or JWT.
#IdentityReference: string & =~"^[a-zA-Z0-9_-]+$"

// AuthType is the type of authentication to use for the EntryPoint.
#AuthType: "none" | "basicAuth" | "jwt" | "oidc" | "extAuth" | "authorization"

// BaseDomain is the domain name to use either as a suffix in the case of Type=domain or as a prefix/domain in the case of Type=path.
#BaseDomain: string & =~"^([a-zA-Z0-9_]+\\.)*[a-zA-Z0-9_]+$"

// RoutingType is the way the Kode resource is accessed by the user.
#RoutingType: "subdomain" | "path"