package v1alpha2

import (
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KodeSpec defines the desired state of Kode
#KodeSpec: {
    // The reference to a template. Either a ContainerTemplate or VirtualTemplate.
    templateRef: #CrossNamespaceObjectReference

    // Specifies the credentials for the service.
    credentials?: #CredentialsSpec

    // The path to the directory for the user data. Defaults to '/config'.
    home?: string & =~"^.{3,}$" | *"/config"

    // The user specified workspace directory (e.g. my-workspace).
    workspace?: string & =~"^[a-zA-Z0-9_-]{3,}$"

    // Specifies the storage configuration.
    storage?: #KodeStorageSpec

    // Specifies a git repository URL to get user configuration from.
    userConfig?: string & =~"^(https?:\\/\\/)?([\\w\\.-]+@)?([\\w\\.-]+)(:\\d+)?\\/?([\\w\\.-]+)\\/([\\w\\.-]+)(\\.git)?(\\/?)|(\\#[\\w\\.-_]+)?$|^oci:\\/\\/([\\w\\.-]+)(:\\d+)?\\/?([\\w\\.-\\/]+)(@sha256:[a-fA-F0-9]{64})?$"

    // Specifies if the container should run in privileged mode. Will only work if the KodeTemplate allows it. Only set to true if you know what you are doing.
    privileged?: bool | *false

    // Specifies the OCI containers to be run as InitContainers. These containers can be used to prepare the workspace or run some setup scripts. It is an ordered list.
    initPlugins?: [...#InitPluginSpec]
}

// KodeStorageSpec defines the storage configuration
#KodeStorageSpec: {
    // Specifies the access modes for the persistent volume.
    accessModes?: [...corev1.#PersistentVolumeAccessMode]

    // Specifies the storage class name for the persistent volume.
    storageClassName?: string

    // Specifies the resource requirements for the persistent volume.
    resources?: corev1.#VolumeResourceRequirements

    // Specifies if the volume should be kept when the kode is recycled. Defaults to false.
    keepVolume?: bool | *false

    // Specifies an existing PersistentVolumeClaim to use instead of creating a new one.
    existingVolumeClaim?: string
}

// KodeStatus defines the observed state of Kode
#KodeStatus: {
    #CommonStatus

    // The URL to access the Kode.
    kodeUrl?: #KodeUrl

    // The port to access the Kode.
    kodePort?: #Port

    // The URL to the icon for the Kode.
    iconUrl?: #KodeIconUrl

    // The runtime for the Kode.
    runtime?: #Runtime

    // The timestamp when the last activity occurred.
    lastActivityTime?: metav1.#Time

    // RetryCount keeps track of the number of retry attempts for failed states.
    retryCount: int | *0

    // DeletionCycle keeps track of the number of deletion cycles. This is used to determine if the resource is deleting.
    deletionCycle: int | *0
}

// Kode is the Schema for the kodes API
#Kode: {
    metav1.#TypeMeta
    metadata: metav1.#ObjectMeta
    spec:     #KodeSpec
    status?:  #KodeStatus
}

// KodeList contains a list of Kode
#KodeList: {
    metav1.#TypeMeta
    metadata: metav1.#ListMeta
    items: [...#Kode]
}

#Runtime: {
    // KodeRuntime is the runtime for the Kode resource. Can be one of 'container', 'virtual'.
    runtime: #KodeRuntime

    // Type is the container runtime for Kode resource.
    type?: #RuntimeType
}

// KodeRuntime specifies the runtime for the Kode resource.
#KodeRuntime: "container" | "virtual"

// RuntimeType specifies the type of the runtime for the Kode resource.
#RuntimeType: "containerd" | "gvisor"

#KodeHostname: string
#KodeDomain: string
#KodeUrl: string
#KodePath: string
#KodeIconUrl: string
#Port: int
