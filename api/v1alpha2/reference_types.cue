package v1alpha2

// CrossNamespaceObjectReference defines a reference to a Kubernetes object across namespaces
#CrossNamespaceObjectReference: {
    // API version of the referent.
    apiVersion?: string

    // Kind is the resource kind.
    kind: #Kind

    // Name is the name of the resource.
    name: #ObjectName

    // Namespace is the namespace of the resource.
    namespace?: #Namespace
}
