# Helm

## Building the helm charts

```shell
go install sigs.k8s.io/kustomize/kustomize/v5@latest
go install github.com/arttor/helmify/cmd/helmify@latest

go run sigs.k8s.io/kustomize/kustomize/v5 build config/default | go run github.com/arttor/helmify/cmd/helmify -cert-manager-as-subchart -cert-manager-version 1.15.0 -generate-defaults -image-pull-secrets helm-charts/kode
go run sigs.k8s.io/kustomize/kustomize/v5 build config/crd | go run github.com/arttor/helmify/cmd/helmify helm-charts/kode-crd
```
