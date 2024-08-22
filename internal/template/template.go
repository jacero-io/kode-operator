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

// TemplateManager defines the interface for managing Kode templates
type TemplateManager interface {
	Fetch(ctx context.Context, ref kodev1alpha2.CrossNamespaceObjectReference) (*kodev1alpha2.Template, error)
}

type DefaultTemplateManager struct {
	Client client.Client
	Log    logr.Logger
}

func NewDefaultTemplateManager(client client.Client, log logr.Logger) TemplateManager {
	return &DefaultTemplateManager{
		Client: client,
		Log:    log,
	}
}

func (m *DefaultTemplateManager) Fetch(ctx context.Context, ref kodev1alpha2.CrossNamespaceObjectReference) (*kodev1alpha2.Template, error) {
	template, err := m.fetchTemplate(ctx, ref)
	if err != nil {
		return nil, err
	}

	return template, nil
}

func (m *DefaultTemplateManager) fetchTemplate(ctx context.Context, ref kodev1alpha2.CrossNamespaceObjectReference) (*kodev1alpha2.Template, error) {
	log := m.Log.WithValues("templateRef", ref)
	log.V(1).Info("Fetching template", "kind", ref.Kind, "name", ref.Name, "namespace", ref.Namespace)

	template := &kodev1alpha2.Template{
		Kind: ref.Kind,
		Name: ref.Name,
	}

	// Only set Namespace if it's not nil
	if ref.Namespace != nil {
		template.Namespace = *ref.Namespace
	}

	var podTemplateSpec *kodev1alpha2.PodTemplateSharedSpec
	var tofuTemplateSpec *kodev1alpha2.TofuSharedSpec
	var port kodev1alpha2.Port

	switch ref.Kind {
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindPodTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterPodTemplate):

		log.V(1).Info("Fetching PodTemplate", "kind", ref.Kind)

		if ref.Kind == kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterPodTemplate) {
			clusterPodTemplate := &kodev1alpha2.ClusterPodTemplate{}

			if err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name)}, clusterPodTemplate); err != nil {
				log.Error(err, "Failed to get ClusterPodTemplate")
				return nil, handleNotFoundError(err, ref)
			}

			log.V(1).Info("ClusterPodTemplate fetched successfully")
			podTemplateSpec = &clusterPodTemplate.Spec.PodTemplateSharedSpec
			port = *clusterPodTemplate.Spec.Port

		} else {

			podTemplate := &kodev1alpha2.PodTemplate{}

			if err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: string(template.Namespace)}, podTemplate); err != nil {
				log.Error(err, "Failed to get PodTemplate")
				return nil, handleNotFoundError(err, ref)
			}

			log.V(1).Info("PodTemplate fetched successfully")
			podTemplateSpec = &podTemplate.Spec.PodTemplateSharedSpec
			port = *podTemplate.Spec.Port

		}

		template.PodTemplateSpec = podTemplateSpec
		template.Port = port

	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindTofuTemplate), kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterTofuTemplate):

		log.V(1).Info("Fetching TofuTemplate", "kind", ref.Kind)

		if ref.Kind == kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterTofuTemplate) {

			clusterKodeTofu := &kodev1alpha2.ClusterTofuTemplate{}

			if err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name)}, clusterKodeTofu); err != nil {
				log.Error(err, "Failed to get ClusterTofuTemplate")
				return nil, handleNotFoundError(err, ref)
			}

			log.V(1).Info("ClusterTofuTemplate fetched successfully")
			tofuTemplateSpec = &clusterKodeTofu.Spec.TofuSharedSpec
			port = *clusterKodeTofu.Spec.Port

		} else {

			kodeTofu := &kodev1alpha2.TofuTemplate{}

			if err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: string(template.Namespace)}, kodeTofu); err != nil {
				log.Error(err, "Failed to get TofuTemplate")
				return nil, handleNotFoundError(err, ref)
			}

			log.V(1).Info("TofuTemplate fetched successfully")
			tofuTemplateSpec = &kodeTofu.Spec.TofuSharedSpec
			port = *kodeTofu.Spec.Port

		}
		template.TofuTemplateSpec = tofuTemplateSpec
		template.Port = port

	default:
		err := fmt.Errorf("unknown template kind: %s", ref.Kind)
		log.Error(err, "Unknown template kind")
		return nil, err
	}

	log.V(1).Info("Template fetched successfully", "kind", template.Kind, "port", template.Port)
	return template, nil
}

func handleNotFoundError(err error, ref kodev1alpha2.CrossNamespaceObjectReference) error {
    if errors.IsNotFound(err) {
        namespacedName := types.NamespacedName{Name: string(ref.Name)}
        if ref.Namespace != nil {
            namespacedName.Namespace = string(*ref.Namespace)
        }
        return &common.TemplateNotFoundError{
            NamespacedName: namespacedName,
            Kind:           string(ref.Kind),
        }
    }
    return err
}
