# kode-crd

![Version: v0.0.0-latest](https://img.shields.io/badge/Version-v0.0.0--latest-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: latest](https://img.shields.io/badge/AppVersion-latest-informational?style=flat-square)

The CRD Helm chart for Kode.

**Homepage:** <https://kode.jacero.io/>

## Source Code

* <https://github.com/jacero-io/kode-operator/>

## Usage

[Helm](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs) to get started.

### Install from DockerHub

Once Helm has been set up correctly, install the chart from dockerhub:

``` shell
    helm install kode-crd oci://docker pull ghcr.io/jacero-io/charts/kode-crd --version v0.0.0-latest -n kode-system --create-namespace
```

You can find all helm chart release in [Dockerhub](https://github.com/jacero-io/kode-operator/pkgs/container/charts%2Fkode-crd)
