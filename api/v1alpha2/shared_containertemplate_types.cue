package v1alpha2

import (
    corev1 "k8s.io/api/core/v1"
)

// ContainerTemplateSharedSpec defines the desired state of KodeContainer
#ContainerTemplateSharedSpec: {
    #CommonSpec

    // Type is the type of container to use.
    type?: #ContainerType

    // Runtime is the runtime for the service.
    runtime?: #RuntimeType

    // Image is the container image for the service.
    image?: string & !=""

    // Command is the command to run in the container.
    command?: [...string]

    // Specifies the environment variables.
    env?: [...corev1.#EnvVar]

    // EnvFrom are the environment variables taken from a Secret or ConfigMap.
    envFrom?: [...corev1.#EnvFromSource]

    // Specifies the args.
    args?: [...string]

    // Specifies the resources for the pod.
    resources?: corev1.#ResourceRequirements

    // Timezone for the service process. Defaults to 'UTC'.
    tz?: string | *"UTC"

    // User ID for the service process. Defaults to '1000'.
    puid?: int64 | *1000

    // Group ID for the service process. Defaults to '1000'.
    pgid?: int64 | *1000

    // Default home is the path to the directory for the user data. Defaults to '/config'.
    defaultHome?: string & =~"^.{3,}$" | *"/config"

    // Default workspace is the default directory for the kode instance on first launch. Defaults to 'workspace'.
    defaultWorkspace?: string & =~"^[^/].*$" & strings.MinRunes(3) | *"workspace"

    // Flag to allow privileged execution of the container.
    allowPrivileged?: bool | *false

    // OCI containers to be run before the main container. It is an ordered list.
    initPlugins?: [...#InitPluginSpec]
}

// ContainerTemplateSharedStatus defines the observed state for Container
#ContainerTemplateSharedStatus: {
    #CommonStatus
}

// ContainerType defines the type of container to use.
ContainerType: string & "code-server" | "webtop"
