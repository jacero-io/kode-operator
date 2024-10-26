package v1alpha2

import (
    corev1 "k8s.io/api/core/v1"
)

// InitPluginSpec defines the specification for an init container plugin
#InitPluginSpec: {
    // The name of the container.
    name: string

    // The OCI image for the container.
    image: string

    // The command to run in the container.
    command?: [...string]

    // The arguments that will be passed to the command in the main container.
    args?: [...string]

    // The environment variables for the main container.
    env?: [...corev1.#EnvVar]

    // The environment variables taken from a Secret or ConfigMap for the main container.
    envFrom?: [...corev1.#EnvFromSource]

    // The volume mounts for the container. Can be used to mount a ConfigMap or Secret.
    volumeMounts?: [...corev1.#VolumeMount]
}
