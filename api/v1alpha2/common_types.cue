package v1alpha2

import (
    "strings"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CredentialsSpec defines the credentials for the service
#CredentialsSpec: {
    // Is both the HTTP Basic auth username (when used) and the user the container should run as. Defaults to 'abc'.
    username?: string

    // HTTP Basic auth password. If unset, there will be no authentication.
    password?: string

    // ExistingSecret is a reference to an existing secret containing user and password. If set, User and Password fields are ignored.
    // MUST set "username" and "password" in lowercase in the secret. CAN set either "username" or "password" or both.
    existingSecret?: string

    // EnableBuiltinAuth enables the built-in HTTP Basic auth.
    enableBuiltinAuth?: bool | *false
}

// CommonSpec defines the common fields for both Container and Virtualization specs
#CommonSpec: {
    // Credentials specifies the credentials for the service.
    credentials?: #CredentialsSpec

    // EntryPointSpec defines the desired state of the entrypoint.
    entryPointRef?: #CrossNamespaceObjectReference

    // Specifies the period before controller inactive the resource (delete all resources except volume).
    inactiveAfterSeconds?: int64 | *600

    // Specifies the period before controller recycle the resource (delete all resources).
    recycleAfterSeconds?: int64 | *28800

    // Port is the port for the service process. Used by EnvoyProxy to expose the kode.
    port?: #Port | *8000
}

// CommonStatus defines the common observed state
#CommonStatus: {
    // ObservedGeneration is the last observed generation of the resource.
    observedGeneration?: int64

    // Conditions reflect the current state of the resource
    #ConditionedStatus

    // Contains the last error message encountered during reconciliation.
    lastError?: string

    // The timestamp when the last error occurred.
    lastErrorTime?: metav1.#Time

    // Phase is the current state of the resource.
    phase?: #Phase
}

// Template represents a unified structure for different types of Kode templates
#Template: {
    // Kind specifies the type of template (e.g., "ContainerTemplate", "ClusterContainerTemplate")
    kind: #Kind

    // Name is the name of the template resource
    name: #ObjectName

    // Namespace is the namespace of the template resource
    namespace?: #Namespace

    // Port is the port to expose the kode instance
    port: #Port

    // ContainerTemplateSpec is a reference to a ContainerTemplate or ClusterContainerTemplate
    container?: #ContainerTemplateSharedSpec

    // EntryPointSpecRef is a reference to an EntryPointSpec
    entryPointRef?: #CrossNamespaceObjectReference

    entryPoint?: #EntryPointSpec
}

#TemplateKind: "ContainerTemplate" | "ClusterContainerTemplate" | "VirtualTemplate" | "ClusterVirtualTemplate"

#Phase: "Pending" | "Configuring" | "Provisioning" | "Active" | "Updating" | "Deleting" | "Failed" | "Unknown"

// Port for the service. Used by EnvoyProxy to expose the container. Defaults to '8000'.
#Port: int32 & >=1 | *8000

// Group refers to a Kubernetes Group. It must either be an empty string or a RFC 1123 subdomain.
#Group: string & =~"^$|^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$" & strings.MaxRunes(253)

// Kind refers to a Kubernetes Kind. It must start with an uppercase letter, followed by alphanumeric characters or dashes, and end with an alphanumeric character.
#Kind: string & =~"^[A-Z][a-zA-Z0-9]*(-[a-zA-Z0-9]+)*$" & strings.MinRunes(1) & strings.MaxRunes(63)

// ObjectName refers to the name of a Kubernetes object. It must be a valid RFC 1123 label.
#ObjectName: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(253)

// Namespace refers to a Kubernetes namespace. It must be a valid RFC 1123 label.
#Namespace: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(63)
