# kode

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/jacero-io/kode-operator)
![GitHub last commit](https://img.shields.io/github/last-commit/jacero-io/kode-operator)
![GitHub issues](https://img.shields.io/github/issues/jacero-io/kode-operator)
![GitHub pull requests](https://img.shields.io/github/issues-pr/jacero-io/kode-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/jacero-io/kode-operator)](https://goreportcard.com/report/github.com/jacero-io/kode-operator)
![Go Version](https://img.shields.io/badge/Go-v1.22%2B-blue)
![Kubernetes Version](https://img.shields.io/badge/Kubernetes-v1.29.1%2B-blue)
![Docker Version](https://img.shields.io/badge/Docker-v25.0.0%2B-blue)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kode-operator)](https://artifacthub.io/packages/helm/kode-operator/kode-operator)

**DISCLAIMER! THIS PROJECT IS UNDER HEAVY DEVELOPMENT AND NOT MEANT FOR PRODUCTION USE.**

Kode is a cloud-native development environment hosted inside a kubernetes cluster.

## Overview

Kode-Operator is a Kubernetes operator that manages the entire lifecycle of various ephemeral and semi-ephemeral development environments. It integrates a comprehensive suite of security tools (Falco, Envoy proxy) and observability standards (OpenTelemetry), ensuring robust security and transparency.

Currently, Kode-Operator supports Code-server, Webtop environments, with plans to support more in the future (eg. Jupyter). It is also relatively easily extendable to support other environments and tools beyond those listed. It supports containerd as kubernetes runtime with plans to include kata containers in the future.

The project has planned to manage in-cluster virtual machines using [Virtink](https://github.com/smartxworks/virtink) or [KubeVirt](https://kubevirt.io/) or external resources with Tofu/Terraform using [tofu-controller](https://github.com/flux-iac/tofu-controller).

## Description

Kode-Operator simplifies the setup and management of development environments on Kubernetes. Using custom resource definitions (CRDs), users can declaratively specify their desired environment state, and Kode-Operator automates the deployment process.

## Key Features

* Define your development environments using CRDs for consistent and repeatable setups.
* Integrated security tools like Falco and Envoy proxy protect your environments.
* OpenTelemetry standards provide deep insights.
* Manage a variety of development environments such as Code-server, Webtop, and Jupyter.
* Easily choose between different runtimes (e.g. containerd, kata/firecracker, or kubevirt)
* Easily extendable to support additional environments and tools beyond the current ones.
* Customize your development environment beforehand by building your own images or inject scripts into the Kode instance using [init plugins]().

## Key Concepts

### Kode

The Kode instance contains a reference to a Kode template. It can also contain other user specific options. For example, a reference to a git repository with vscode settings and extensions.

```yaml
apiVersion: kode.jacero.io/v1alpha2
kind: Kode
metadata:
  name: my-kode-instance
  namespace: default
spec:
  credentials:
    username: myuser
    password: mypassword
    enableBuiltinAuth: true
  templateRef:
    kind: PodTemplate
    name: my-kode-template
  home: /home/myuser
  workspace: my-workspace
  storage:
    accessModes:
      - ReadWriteOnce
    storageClassName: my-storage-class
    resources:
      requests:
        storage: 5Gi
```

### PodTemplate & ClusterPodTemplate

These are cluster scoped and namespace scoped templates. A template contains an image and some default configuration for that image. You can also include an Envoy Proxy configuration that is then applied to the sidecar of the resulting Kode instance.

**Example for PodTemplate:**

```yaml
apiVersion: kode.jacero.io/v1alpha2
kind: PodTemplate
metadata:
  name: my-kode-template
  namespace: default
spec:
  type: code-server
  image: linuxserver/code-server:latest
  tz: UTC
  defaultHome: /config
  defaultWorkspace: workspace
```

**Example for ClusterPodTemplate:**

```yaml
apiVersion: kode.jacero.io/v1alpha2
kind: ClusterPodTemplate
metadata:
  name: my-kode-cluster-template
spec:
  type: webtop
  image: linuxserver/webtop:debian-xfce
  tz: UTC
  defaultHome: /config
  defaultWorkspace: workspace
```

### EntryPoint & ClusterEntryPoint

This one expect you to have deployed Envoy Gateway and setup a gateway named `eg` in the `default` namespace.

```yaml
apiVersion: kode.jacero.io/v1alpha2
kind: EntryPoint
metadata:
  name: entrypoint-sample
  namespace: default
spec:
  routingType: "subdomain"
  baseDomain: "kode.example.com"
  gatewaySpec:
    existingGatewayRef:
      kind: Gateway
      name: eg
      namespace: default

```

### Features

* [x] PodTemplate - Deploying `code-server`, `webtop`, and `jupyter` directly into kubernetes accessing them through your browser.
* [ ] TofuTemplate - Deploying anything you can imagine in using Tofu.
* [ ] Authentication - Enforce `Basic auth`, `OIDC`, `JWT`, or `x509` authentication.
* [ ] Authorization - Make sure only you have access to your stuff!
* [ ] Observability - Know exactly what is going wrong and how well your development environments are doing.
* [x] Customizability - Add any extra configuration or run anything at startup or build your own base images.
* [ ] Resource Optimization - Remove unused deployments while keeping any persistent data.
* [ ] CLI - Create templates or Kode instances, right behind your keyboard.

## Usage Scenarios

### Scenario 1: Setting Up a Simple Code-Server Environment

You want to set up a VSCode-like development environment using Code-server for your team. This setup allows developers to access their development environment from any browser.

**1. Create a PodTemplate for code-server:**

```yaml
apiVersion: v1alpha2
kind: PodTemplate
metadata:
  name: code-server-template
spec:
  type: code-server
  image: linuxserver/code-server:latest
  defaultHome: /config
  defaultWorkspace: workspace
```

**2. Create a Kode instance using the template:**

```yaml
apiVersion: v1alpha2
kind: Kode
metadata:
  name: my-code-server
spec:
  credentials:
    username: devuser
  templateRef:
    kind: PodTemplate
    name: code-server-template
  workspace: my-project # Overrides the template workspace
```

**3. Apply the Configuration:**

```yaml
kubectl apply -f code-server-template.yaml
kubectl apply -f my-code-server.yaml
```

## Install using Timoni

### Timoni Prerequisites

* timoni version v0.22.0+.

### Install module

TBD

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

### Reporting Issues

If you find a bug or have a feature request, please create an issue in the [issue tracker](https://github.com/jacero-io/kode-operator/issues) with as much detail as possible. Include steps to reproduce the issue and any relevant logs or screenshots.

### Development Setup

1. Ensure you have the required dependencies installed:

    * Go version v1.22.0+
    * Docker version 25.0.0+
    * kubectl version v1.29.1+
    * Access to a Kubernetes v1.29.1+ cluster or kind

2. Launch the kind cluster:

    ```sh
    task setup-dev
    ```

3. Use `task` to run the controller.

    ```sh
    task run
    ```

4. **Run Tests**: Ensure all tests pass before submitting your pull request.

    ```sh
    task test-integration
    ```

5. **Teardown dev**: Remove everything.

    ```sh
    task teardown-dev
    ```

### Documentation

If you are adding a new feature, please include relevant documentation updates. Documentation is located in the `docs/` directory.
