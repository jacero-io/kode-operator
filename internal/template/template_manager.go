package template

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DefaultTemplateManager struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultTemplateManager(client client.Client, log logr.Logger) TemplateManager {
	return &DefaultTemplateManager{
		client: client,
		log:    log,
	}
}

func (m *DefaultTemplateManager) Fetch(ctx context.Context, ref kodev1alpha2.KodeTemplateReference) (*kodev1alpha2.Template, error) {
	template, err := m.fetchTemplate(ctx, ref)
	if err != nil {
		return nil, err
	}

	return template, nil
}

func (m *DefaultTemplateManager) fetchTemplate(ctx context.Context, ref kodev1alpha2.KodeTemplateReference) (*kodev1alpha2.Template, error) {
	template := &kodev1alpha2.Template{
		Kind:      ref.Kind,
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}

	switch ref.Kind {
	case "PodTemplate", "ClusterPodTemplate":
		var podTemplateSpec *kodev1alpha2.PodTemplateSharedSpec
		var port int32
		if ref.Kind == "ClusterPodTemplate" {
			clusterPodTemplate := &kodev1alpha2.ClusterPodTemplate{}
			if err := m.client.Get(ctx, types.NamespacedName{Name: ref.Name}, clusterPodTemplate); err != nil {
				return nil, handleNotFoundError(err, ref)
			}
			podTemplateSpec = &clusterPodTemplate.Spec.PodTemplateSharedSpec
			port = clusterPodTemplate.Spec.Port
		} else {
			podTemplate := &kodev1alpha2.PodTemplate{}
			if err := m.client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, podTemplate); err != nil {
				return nil, handleNotFoundError(err, ref)
			}
			podTemplateSpec = &podTemplate.Spec.PodTemplateSharedSpec
			port = podTemplate.Spec.Port
		}
		template.PodTemplateSpec = podTemplateSpec
		template.Port = port

	case "TofuTemplate", "ClusterTofuTemplate":
		var tofuTemplateSpec *kodev1alpha2.TofuSharedSpec
		var port int32
		if ref.Kind == "ClusterTofuTemplate" {
			clusterKodeTofu := &kodev1alpha2.ClusterTofuTemplate{}
			if err := m.client.Get(ctx, types.NamespacedName{Name: ref.Name}, clusterKodeTofu); err != nil {
				return nil, handleNotFoundError(err, ref)
			}
			tofuTemplateSpec = &clusterKodeTofu.Spec.TofuSharedSpec
			port = clusterKodeTofu.Spec.Port
		} else {
			kodeTofu := &kodev1alpha2.TofuTemplate{}
			if err := m.client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, kodeTofu); err != nil {
				return nil, handleNotFoundError(err, ref)
			}
			tofuTemplateSpec = &kodeTofu.Spec.TofuSharedSpec
			port = kodeTofu.Spec.Port
		}
		template.TofuTemplateSpec = tofuTemplateSpec
		template.Port = port

	default:
		return nil, fmt.Errorf("unknown template kind: %s", ref.Kind)
	}

	return template, nil
}

func handleNotFoundError(err error, ref kodev1alpha2.KodeTemplateReference) error {
	if errors.IsNotFound(err) {
		return &common.TemplateNotFoundError{
			NamespacedName: types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace},
			Kind:           ref.Kind,
		}
	}
	return err
}
