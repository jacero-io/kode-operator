# kode-operator

Kode-Operator is a Kubernetes operator designed to manage the full lifecycle of various ephemeral and semi ephemeral development environments and tools. It automates the deployment, configuration, and management of Code-server, Webtop, DevContainers, Alnoda, and Jupyter environments within a Kubernetes cluster.

Additionally, it supports authentication and authorization using Envoy Proxy and a standard way for the user to add custom configuration and data.

## Description

Kode-Operator streamlines the setup and management of development environments on Kubernetes. By leveraging custom resource definitions (CRDs), it allows users to declaratively specify the desired state of their environments and automates the deployment process.

Whether you need a code server, a web-based development environment, or a data science notebook, Kode-Operator handles the orchestration, ensuring consistency and efficiency.

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
  user: myuser
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

### KodeTemplate & ClusterKodeTemplate

These are cluster scoped and namespace scoped templates. A template contains an image and some default configuration for that image. A user can also include an Envoy Proxy configuration that is then applied to the sidecar of the resulting Kode instance.

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
  port: 8443
  envs:
    - name: TZ
      value: UTC
  args:
    - --auth
    - none
  tz: UTC
  puid: 1000
  pgid: 1000
  defaultHome: /config
  defaultWorkspace: workspace
  envoyProxyTemplateRef:
    name: my-envoy-proxy-template

```

**Example for ClusterKodeTemplate:**

```yaml
apiVersion: kode.jacero.io/v1alpha1
kind: ClusterKodeTemplate
metadata:
  name: my-cluster-kode-template
spec:
  type: webtop
  image: linuxserver/webtop:ubuntu-mate
  port: 3000
  envs:
    - name: PUID
      value: "1000"
  args:
    - --init
  tz: UTC
  puid: 1000
  pgid: 1000
  defaultHome: /config
  defaultWorkspace: workspace
  envoyProxyTemplateRef:
    name: my-envoy-proxy-template

```

### EnvoyProxyTemplate & ClusterEnvoyProxyTemplate

These are cluster scoped and namespace scoped template for the Envoy Proxy sidecar. A way to define a standard Envoy Proxy configuration that a Kode template should use. This could be a HTTP filter to an Open Policy Agent (OPA) deployment within the cluster.

```yaml
```

### Features

- [ ] Full lifecycle of [Code-server](https://docs.linuxserver.io/images/docker-code-server/).
- [ ] Full lifecycle of [Webtop](https://docs.linuxserver.io/images/docker-webtop/).
- [ ] Full lifecycle of [DevContainers](https://containers.dev/).
- [ ] Full lifecycle of [Alnoda](https://docs.alnoda.org/about/intro/).
- [ ] Full lifecycle of [Jupyter](https://jupyter.org/).
- [ ] Authentication & Authorization support using Envoy Proxy sidecar.
  - [ ] [OAuth2](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/oauth2_filter)
  - [ ] [Basic Auth](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/basic_auth_filter.html#basic-auth)
  - [ ] [Ext_Authz](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter) HTTP and GRPC (Used by for example OPA)
- [ ] [Code-server] User defined settings and data (e.g. extensions, settings.json, etc).
- [ ] [Webtop] User defined settings and data (e.g. .profile, .bashrc, preinstalled software, etc).
- [ ] Backup & Restore of user data to S3.

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
  port: 8443
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
  user: devuser
  password: devpassword
  workspace: my-project
```

**3. Apply the Configuration:**

```yaml
kubectl apply -f code-server-template.yaml
kubectl apply -f my-code-server.yaml
```

### Scenario 2: Deploying Jupyter Notebooks

You need to provide Jupyter Notebooks for your data science team to perform data analysis and research.

**1. Create a KodeTemplate for Jupyter:**

```yaml
apiVersion: v1alpha1
kind: KodeTemplate
metadata:
  name: jupyter-template
spec:
  type: jupyter
  image: jupyter/base-notebook:latest
  port: 8888
  defaultHome: /home/jovyan
  defaultWorkspace: notebooks
```

**2. Create a Kode Instance Using the Template:**

```yaml
apiVersion: v1alpha1
kind: Kode
metadata:
  name: my-jupyter
spec:
  templateRef:
    kind: KodeTemplate
    name: jupyter-template
  user: datascientist
  password: securepassword
  workspace: data-analysis
```

**3. Apply the Configuration:**

```yaml
kubectl apply -f jupyter-template.yaml
kubectl apply -f my-jupyter.yaml
```

### Scenario 3: Using Envoy Proxy for Authentication

You want to secure your Code-server environment using Envoy Proxy with Basic Auth.

**1. Create an EnvoyProxyTemplate:**

```yaml
apiVersion: v1alpha1
kind: EnvoyProxyTemplate
metadata:
  name: basic-auth-proxy
spec:
  auth:
    type: basic
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
  port: 8443
  envoyProxyTemplateRef:
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
  user: devuser # Sent to the Envoy Proxy Basic Auth
  password: devpassword # Sent to the Envoy Proxy Basic Auth
  workspace: my-secure-project
```

**3. Apply the Configuration:**

```yaml
kubectl apply -f basic-auth-proxy.yaml
kubectl apply -f secure-code-server-template.yaml
kubectl apply -f secure-code-server.yaml
```

## Install using Kustomize

### Kustomize Prerequisites

- kubectl version v1.28.0+.
- Access to a Kubernetes v1.28.0+ cluster.

### To Deploy on the cluster

TBD

### To Uninstall

TBD

## Install using Helm

### Helm Prerequisites

- helm version 3.10+.

### Helm install

TBD

### Helm uninstall

TBD

## Contributing

We welcome contributions to the Kode-Operator project! Here are some guidelines to help you get started:

### How to Contribute

1. **Fork the Repository**: Click the "Fork" button at the top of this repository to create a copy of the repository in your own GitHub account.

2. **Clone Your Fork**: Use `git clone` to clone your forked repository to your local machine.

    ```sh
    git clone https://github.com/<your-username>/kode-operator.git
    cd kode-operator
    ```

3. **Create a Branch**: Create a new branch for your feature or bugfix.

    ```sh
    git checkout -b feature-or-bugfix-name
    ```

4. **Make Changes**: Make your changes to the code. Ensure your code follows the project's coding standards and includes appropriate tests.

5. **Commit Your Changes**: Commit your changes with a descriptive commit message.

    ```sh
    git add .
    git commit -m "Description of the feature or fix"
    ```

6. **Push to Your Fork**: Push your changes to your forked repository.

    ```sh
    git push origin feature-or-bugfix-name
    ```

7. **Create a Pull Request**: Go to the original repository and create a pull request from your fork. Provide a clear and detailed description of your changes and the problem they solve.

### Code of Conduct

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project, you agree to abide by its terms.

### Reporting Issues

If you find a bug or have a feature request, please create an issue in the [issue tracker](https://github.com/emil-jacero/kode-operator/issues) with as much detail as possible. Include steps to reproduce the issue and any relevant logs or screenshots.

### Development Setup

1. **Install Dependencies**: Ensure you have the required dependencies installed:
    - Go version v1.21.0+
    - Docker version 17.03+
    - kubectl version v1.29.1+
    - Access to a Kubernetes v1.29.1+ cluster

2. **Build the Project**: Use `make` to build the project.

    ```sh
    make build
    ```

3. **Run Tests**: Ensure all tests pass before submitting your pull request.

    ```sh
    make test
    ```

### Documentation

If you are adding a new feature, please include relevant documentation updates. Documentation is located in the `docs/` directory.

### Getting Help

If you need help with the project, feel free to join the [discussion forum](https://github.com/org/repo/discussions) or reach out on our [Slack channel](https://join.slack.com/t/yourworkspace/shared_invite/).

Thank you for your interest in contributing to Kode-Operator! Your contributions are greatly appreciated.

## License

```
Copyright 2024.

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
