# TASKS.md - Comprehensive Task Documentation

This document provides an overview of all available tasks in our project's Taskfiles. Tasks are organized by their respective Taskfiles and correctly namespaced.

## Table of Contents

1. [Build Tasks](#build-tasks)
2. [Cluster Management Tasks](#cluster-management-tasks)
3. [Development Tasks](#development-tasks)
4. [Linting Tasks](#linting-tasks)
5. [Sample Management Tasks](#sample-management-tasks)
6. [Testing Tasks](#testing-tasks)
7. [Tool Management Tasks](#tool-management-tasks)
8. [Helm Chart Tasks](#helm-chart-tasks)
9. [Envoy Tasks](#envoy-tasks)
10. [Flux Tasks](#flux-tasks)

## Build Tasks

Located in `build.yaml`

### build:manifests

- **Description**: Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
- **Usage**: `task build:manifests`
- **Output**: Generated manifests in `config/crd/bases` directory.

### build:generate

- **Description**: Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
- **Usage**: `task build:generate`

### build:build

- **Description**: Build manager binary.
- **Usage**: `task build:build`
- **Output**: Compiled binary in `bin/manager`.

### build:build-installer

- **Description**: Generate a consolidated YAML with CRDs and deployment, excluding Prometheus components.
- **Usage**: `task build:build-installer`
- **Output**: Installation YAML in `dist/install.yaml`.

### build:docker-build

- **Description**: Build docker image locally.
- **Usage**: `task build:docker-build [PUSH=true|false]`
- **Variables**:
  - `PUSH`: Whether to push the image (default: "true")

### build:docker-build-multi

- **Description**: Build docker image for the manager with cross-platform support.
- **Usage**: `task build:docker-build-multi [PUSH=true|false] [PLATFORMS=<platform-list>]`
- **Variables**:
  - `PUSH`: Whether to push the image (default: "false")
  - `PLATFORMS`: Platforms to build for (default: "linux/amd64,linux/arm64")

## Cluster Management Tasks

Located in `clusters.yaml`

### clusters:kind-ensure

- **Description**: Ensure a Kind cluster exists, creating it if necessary.
- **Usage**: `task clusters:kind-ensure`

### clusters:kind-create

- **Description**: Create a Kind cluster.
- **Usage**: `task clusters:kind-create`

### clusters:kind-destroy

- **Description**: Delete the Kind cluster.
- **Usage**: `task clusters:kind-destroy`

### clusters:kind-start

- **Description**: Start the Kind cluster.
- **Usage**: `task clusters:kind-start`

### clusters:kind-stop

- **Description**: Stop the Kind cluster.
- **Usage**: `task clusters:kind-stop`

### clusters:kind-load-image

- **Description**: Load a Docker image into the Kind cluster.
- **Usage**: `task clusters:kind-load-image`

### clusters:kind-load-images

- **Description**: Load internal and external Docker images into Kind cluster.
- **Usage**: `task clusters:kind-load-images`

## Development Tasks

Located in `dev.yaml`

### dev:run

- **Description**: Run a controller from your host.
- **Usage**: `task dev:run [LOG_LEVEL=<level>] [CLI_ARGS="<additional args>"]`
- **Variables**:
  - `LOG_LEVEL`: Set the logging level (default: debug)
  - `CLI_ARGS`: Additional command-line arguments
  - `ENV`: Environment variables for the controller process

### dev:install

- **Description**: Install CRDs into the cluster.
- **Usage**: `task dev:install`

### dev:uninstall

- **Description**: Uninstall CRDs from the cluster.
- **Usage**: `task dev:uninstall`

### dev:deploy

- **Description**: Deploy controller to the specified cluster (Kind or remote).
- **Usage**: `task dev:deploy`

### dev:undeploy

- **Description**: Undeploy controller from the specified cluster.
- **Usage**: `task dev:undeploy`

### dev:setup

- **Description**: Set up the complete development environment.
- **Usage**: `task dev:setup`

### dev:teardown

- **Description**: Tear down the complete development environment.
- **Usage**: `task dev:teardown`

## Linting Tasks

Located in `lint.yaml`

### lint:fmt

- **Description**: Run go fmt against code.
- **Usage**: `task lint:fmt`

### lint:vet

- **Description**: Run go vet against code.
- **Usage**: `task lint:vet`

### lint:lint

- **Description**: Run golangci-lint linter.
- **Usage**: `task lint:lint`

### lint:lint-fix

- **Description**: Run golangci-lint linter and perform fixes.
- **Usage**: `task lint:lint-fix`

## Sample Management Tasks

Located in `samples.yaml`

### samples:get

- **Description**: Get Kubernetes resources defined in config/samples/ from the cluster.
- **Usage**: `task samples:get`

### samples:apply

- **Description**: Apply Kubernetes resources defined in config/samples/ to the cluster.
- **Usage**: `task samples:apply`

### samples:delete

- **Description**: Delete Kubernetes resources defined in config/samples/ from the cluster.
- **Usage**: `task samples:delete`

## Testing Tasks

Located in `test.yaml`

### test:unit

- **Description**: Run unit tests with coverage.
- **Usage**: `task test:unit`

### test:integration

- **Description**: Run integration tests with coverage.
- **Usage**: `task test:integration`

### test:e2e

- **Description**: Run end-to-end tests with Kind cluster.
- **Usage**: `task test:e2e`

### test:all

- **Description**: Run all tests (unit, integration, e2e).
- **Usage**: `task test:all`

### test:coverage-report

- **Description**: Generate a combined coverage report.
- **Usage**: `task test:coverage-report`
- **Output**: Combined coverage report in `coverage/coverage.html`

## Tool Management Tasks

Located in `tools.yaml`

### tools:kustomize

- **Description**: Download kustomize locally if necessary.
- **Usage**: `task tools:kustomize`

### tools:controller-gen

- **Description**: Download controller-gen locally if necessary and create/update symlink.
- **Usage**: `task tools:controller-gen`

### tools:envtest

- **Description**: Download setup-envtest locally if necessary.
- **Usage**: `task tools:envtest`

### tools:golangci-lint

- **Description**: Download golangci-lint locally if necessary.
- **Usage**: `task tools:golangci-lint`

### tools:helmify

- **Description**: Download helmify locally if necessary.
- **Usage**: `task tools:helmify`

## Helm Chart Tasks

Located in `helm.yaml`

### helm:package-kode-crd

- **Description**: Build and package kode-crd Helm chart.
- **Usage**: `task helm:package-kode-crd [VERSION=<version>]`

### helm:package-kode

- **Description**: Build and package kode Helm chart.
- **Usage**: `task helm:package-kode [VERSION=<version>]`

### helm:package

- **Description**: Build and package all Helm charts.
- **Usage**: `task helm:package [VERSION=<version>]`

### helm:manage-kode-crd

- **Description**: Install or uninstall kode-crd Helm chart on the cluster.
- **Usage**: `task helm:manage-kode-crd [ACTION=install|uninstall] [VERSION=<version>]`

### helm:manage-kode

- **Description**: Install or uninstall kode Helm chart on the cluster.
- **Usage**: `task helm:manage-kode [ACTION=install|uninstall] [VERSION=<version>]`

### helm:manage

- **Description**: Install or uninstall all kode Helm charts on the cluster.
- **Usage**: `task helm:manage [ACTION=install|uninstall] [VERSION=<version>]`

## Envoy Tasks

Located in `envoy.yaml`

### envoy:create-dev-cert

- **Description**: Create a development certificate for Envoy Gateway if it doesn't exist.
- **Usage**: `task envoy:create-dev-cert`

### envoy:apply-dev-cert

- **Description**: Apply the development certificate secret to the Kubernetes cluster.
- **Usage**: `task envoy:apply-dev-cert`

### envoy:deploy

- **Description**: Deploy Envoy Gateway.
- **Usage**: `task envoy:deploy`

### envoy:undeploy

- **Description**: Undeploy Envoy Gateway.
- **Usage**: `task envoy:undeploy`

### envoy:install

- **Description**: Install Envoy Gateway manifests.
- **Usage**: `task envoy:install`

### envoy:uninstall

- **Description**: Uninstall Envoy Gateway manifests.
- **Usage**: `task envoy:uninstall`

## Flux Tasks

Located in `flux.yaml`

### flux:deploy

- **Description**: Deploy Flux.
- **Usage**: `task flux:deploy`

### flux:undeploy

- **Description**: Undeploy Flux.
- **Usage**: `task flux:undeploy`

---

For more detailed information on each task, including specific options and behaviors, please refer to the individual Taskfiles or run `task -l` to list all available tasks with their descriptions.
