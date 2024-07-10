# kode-operator

<span style="color: red;">DISCLAIMER! THIS PROJECT IS UNDER HEAVY DEVELOPMENT AND NOT MEANT FOR PRODUCTION USE JUST YET.</span>

Kode-Operator is designed to enhance the developer experience with a strong focus on security and observability.

## Overview

Kode-Operator is a Kubernetes operator that manages the entire lifecycle of various ephemeral and semi-ephemeral development environments. It integrates a comprehensive suite of security tools (Falco, Envoy proxy) and observability standards (OpenTelemetry), ensuring robust security and transparency.

Currently, Kode-Operator supports Code-server, Webtop, Alnoda, and Jupyter environments, with plans to support more in the future. It is also easily extendable to support other environments and tools beyond those listed.

## Description

Kode-Operator simplifies the setup and management of development environments on Kubernetes. Using custom resource definitions (CRDs), users can declaratively specify their desired environment state, and Kode-Operator automates the deployment process.

## Key Features

* Define your development environments using CRDs for consistent and repeatable setups.
* Integrated security tools like Falco and Envoy proxy protect your environments.
* OpenTelemetry standards provide deep insights into your development environments.
* Manage a variety of development environments such as Code-server, Webtop, Alnoda, and Jupyter.
* Easily extendable to support additional environments and tools beyond the current offerings.

## Key Concepts

### Kode

The Kode instance contains a reference to a Kode template. It can also contain other user specific options. For example, a reference to a git repository with vscode settings and extensions.

```yaml
apiVersion: kode.jacero.io/v1alpha1
kind: Kode
metadata:
  name: my-kode-instance
  namespace: default
spec:
  templateRef:
    kind: KodeTemplate
    name: my-kode-template
  username: myuser
  password: mypassword
  home: /home/myuser
  workspace: my-workspace
  storage:
    accessModes:
      - ReadWriteOnce
    storageClassName: my-storage-class
    resources:
      requests:
        storage: 1Gi
```

### KodeTemplate & KodeClusterTemplate

These are cluster scoped and namespace scoped templates. A template contains an image and some default configuration for that image. You can also include an Envoy Proxy configuration that is then applied to the sidecar of the resulting Kode instance.

**Example for KodeTemplate:**

```yaml
apiVersion: kode.jacero.io/v1alpha1
kind: KodeTemplate
metadata:
  name: my-kode-template
  namespace: default
spec:
  type: code-server
  image: linuxserver/code-server:latest
  tz: UTC
  defaultHome: /config
  defaultWorkspace: workspace
  envoyProxyRef:
    name: my-envoy-proxy-config
    namespace: default
```

**Example for KodeClusterTemplate:**

```yaml
apiVersion: kode.jacero.io/v1alpha1
kind: KodeClusterTemplate
metadata:
  name: my-kode-cluster-template
spec:
  type: webtop
  image: linuxserver/webtop:ubuntu-mate
  tz: UTC
  defaultHome: /config
  defaultWorkspace: workspace
  envoyProxyRef:
    name: my-cluster-envoy-proxy-config
```

### EnvoyProxyConfig & EnvoyProxyClusterConfig

These are cluster scoped and namespace scoped template for the Envoy Proxy sidecar. A way to define a standard Envoy Proxy configuration that a Kode template should use. This could be a HTTP filter to an Open Policy Agent (OPA) deployment within the cluster.

```yaml
apiVersion: kode.jacero.io/v1alpha1
kind: EnvoyProxyConfig
metadata:
  labels:
    app.kubernetes.io/name: kode-operator
    app.kubernetes.io/managed-by: kustomize
  name: my-envoy-proxy-config
  namespace: default
spec:
  image: envoyproxy/envoy:v1.30-latest
  httpFilters:
    - name: envoy.filters.http.ext_authz
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
        with_request_body:
          max_request_bytes: 8192
          allow_partial_message: true
        failure_mode_allow: false
        grpc_service:
          envoy_grpc:
            cluster_name: ext_authz-opa-service1
          timeout: 0.250s
        transport_api_version: V3
  clusters:
    - name: ext_authz-opa-service1
      connect_timeout: 0.250s
      lb_policy: round_robin
      type: strict_dns
      load_assignment:
        cluster_name: ext_authz-opa-service1
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: opa.default.svc.cluster.local
                      port_value: 8181
```

### Features

* [x] *Provisioning and update of [Code-server](https://docs.linuxserver.io/images/docker-code-server/).
* [ ] *Provisioning and update of [Webtop](https://docs.linuxserver.io/images/docker-webtop/).
* [ ] Authentication & Authorization support using Envoy Proxy sidecar.
  * [OAuth2](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/oauth2_filter) With external Oauth2 provider.
  * [Ext_Authz](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter) HTTP and GRPC (Used by for example OPA to authorize the users).
* [ ] [Falco](https://falco.org/) sidecar support
* [ ] Kode instance bound to the user identity and namespaced for isolation, an identity provided by an IAM (e.g Keycloak).
* [x] Ability to include InitPlugins which are executed in order. InitPlugins can mutate the instance in any way the administrator or user like.
  * Could for example add VSCode extensions or install software that is not built into the image.
* [ ] Include dotfiles and other user settings in the Kode instance.
* [ ] Pause/Prune container on inactivity, keeping the persistent storage.
  * [ ] Backup & restore of the Kode instance state. Maybe not feasible.
* [ ] Backup & Restore of user data to S3.
* [ ] A Kode CLI to manage Kode resources

## Usage Scenarios

### Scenario 1: Setting Up a Code-Server Environment

You want to set up a VSCode-like development environment using Code-server for your team. This setup allows developers to access their development environment from any browser.

**1. Create a KodeTemplate for Code-server:**

```yaml
apiVersion: v1alpha1
kind: KodeTemplate
metadata:
  name: code-server-template
spec:
  type: code-server
  image: linuxserver/code-server:latest
  defaultHome: /config
  defaultWorkspace: workspace
```

**2. Create a Kode Instance Using the Template:**

```yaml
apiVersion: v1alpha1
kind: Kode
metadata:
  name: my-code-server
spec:
  templateRef:
    kind: KodeTemplate
    name: code-server-template
  username: devuser
  workspace: my-project # Overrides the template workspace
```

**3. Apply the Configuration:**

```yaml
kubectl apply -f code-server-template.yaml
kubectl apply -f my-code-server.yaml
```

### Scenario 2: Using Envoy Proxy for Authentication

You want to secure your Code-server environment using Envoy Proxy with Basic Auth.

**1. Create an EnvoyProxyConfig:**

```yaml
apiVersion: v1alpha1
kind: EnvoyProxyClusterConfig
metadata:
  name: basic-auth-proxy
spec:
  authType: basic
```

**2. Create a KodeTemplate with Envoy Proxy Configuration:**

```yaml
apiVersion: v1alpha1
kind: KodeTemplate
metadata:
  name: secure-code-server-template
spec:
  type: code-server
  image: linuxserver/code-server:latest
  envoyProxyRef:
    kind: EnvoyProxyClusterConfig
    name: basic-auth-proxy
  defaultHome: /config
  defaultWorkspace: workspace
```

**2. Create a Kode Instance Using the Template:**

```yaml
apiVersion: v1alpha1
kind: Kode
metadata:
  name: secure-code-server
spec:
  templateRef:
    kind: KodeTemplate
    name: secure-code-server-template
  username: devuser # Sent to the Envoy proxy Basic Auth
  password: devpassword # Sent to the Envoy proxy Basic Auth
  workspace: my-secure-project # Overrides the template workspace
```

**3. Apply the Configuration:**

```yaml
kubectl apply -f basic-auth-proxy.yaml
kubectl apply -f secure-code-server-template.yaml
kubectl apply -f secure-code-server.yaml
```

## Install using Kustomize

### Kustomize Prerequisites

* kubectl version v1.28.0+.
* Access to a Kubernetes v1.28.0+ cluster.

### To Deploy on the cluster

TBD

### To Uninstall

TBD

## Install using Helm

### Helm Prerequisites

* helm version 3.10+.

### Helm install

TBD

### Helm uninstall

TBD

## Contributing

We welcome contributions to the Kode-Operator project! Here are some guidelines to help you get started:

### Branch naming scheme

Source: <https://dev.to/varbsan/a-simplified-convention-for-naming-branches-and-commits-in-git-il4>

* `feature` is for adding, refactoring or removing a feature
* `bugfix` is for fixing a bug
* `hotfix` is for changing code with a temporary solution and/or without following the usual process (usually because of an emergency)
* `test` is for experimenting outside of an issue/ticket

### How to Contribute

1. **Fork the Repository**: Click the "Fork" button at the top of this repository to create a copy of the repository in your own GitHub account.

2. **Clone Your Fork**: Use `git clone` to clone your forked repository to your local machine.

    ```sh
    git clone https://github.com/<your-username>/kode-operator.git
    cd kode-operator
    ```

3. **Create a Branch**: Create a new branch for your feature or bugfix.

    ```sh
    git checkout -b feature/name
    git checkout -b bugfix/name
    ```

4. **Make Changes**: Make your changes to the code. Ensure your code follows the project's coding standards and includes appropriate tests.

5. **Commit Your Changes**: Commit your changes with a descriptive commit message.

    ```sh
    git add .
    git commit -m "Description of the feature or fix"
    ```

6. **Push to Your Fork**: Push your changes to your forked repository.

    ```sh
    git push origin feature/name
    git push origin bugfix/name
    ```

7. **Create a Pull Request**: Go to the original repository and create a pull request from your fork. Provide a clear and detailed description of your changes and the problem they solve.

### Code of Conduct

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project, you agree to abide by its terms.

### Reporting Issues

If you find a bug or have a feature request, please create an issue in the [issue tracker](https://github.com/jacero-io/kode-operator/issues) with as much detail as possible. Include steps to reproduce the issue and any relevant logs or screenshots.

### Development Setup

1. **Install Dependencies**: Ensure you have the required dependencies installed:
    * Go version v1.22.0+
    * Docker version 25.0.0+
    * kubectl version v1.29.1+
    * Access to a Kubernetes v1.29.1+ cluster or kind

2. **Run the Project**: Use `make` to run the controller.

    ```sh
    make install
    ```

3. **Run Tests**: Ensure all tests pass before submitting your pull request.

    ```sh
    make test-unit
    make test-integration
    ```

### Documentation

If you are adding a new feature, please include relevant documentation updates. Documentation is located in the `docs/` directory.

### Getting Help

If you need help with the project, feel free to join the [discussion forum](https://github.com/org/repo/discussions) or reach out on our [Slack channel](https://join.slack.com/t/yourworkspace/shared_invite/).

Thank you for your interest in contributing to Kode-Operator! Your contributions are greatly appreciated.

## License

```
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
```
