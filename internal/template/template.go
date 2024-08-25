package template

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
	koderrs "github.com/jacero-io/kode-operator/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TemplateManager defines the interface for managing Kode templates
type TemplateManager interface {
	Fetch(ctx context.Context, ref kodev1alpha2.CrossNamespaceObjectReference) (*kodev1alpha2.Template, error)
}

type DefaultTemplateManager struct {
	Client client.Client
	Log    logr.Logger
	Cache  map[string]*kodev1alpha2.Template
}

func NewDefaultTemplateManager(client client.Client, log logr.Logger) TemplateManager {
	return &DefaultTemplateManager{
		Client: client,
		Log:    log,
		Cache:  make(map[string]*kodev1alpha2.Template),
	}
}

func (m *DefaultTemplateManager) Fetch(ctx context.Context, ref kodev1alpha2.CrossNamespaceObjectReference) (*kodev1alpha2.Template, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}

	cacheKey := m.getCacheKey(ref)
	if cachedTemplate, ok := m.Cache[cacheKey]; ok {
		return cachedTemplate, nil
	}

	var template *kodev1alpha2.Template
	var err error

	backoff := wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}

	err = wait.ExponentialBackoff(backoff, func() (bool, error) {
		template, err = m.fetchTemplate(ctx, ref)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, err // Don't retry for NotFound errors
			}
			m.Log.Error(err, "Failed to fetch template, retrying", "ref", ref)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch template after retries: %w", err)
	}

	m.Cache[cacheKey] = template
	return template, nil
}

func (m *DefaultTemplateManager) getCacheKey(ref kodev1alpha2.CrossNamespaceObjectReference) string {
	namespace := ""
	if ref.Namespace != nil {
		namespace = string(*ref.Namespace)
	}
	return fmt.Sprintf("%s/%s/%s", ref.Kind, namespace, ref.Name)
}

func (m *DefaultTemplateManager) fetchTemplate(ctx context.Context, ref kodev1alpha2.CrossNamespaceObjectReference) (*kodev1alpha2.Template, error) {
	log := m.Log.WithValues("templateRef", ref)
	log.V(1).Info("Fetching template", "kind", ref.Kind, "name", ref.Name, "namespace", ref.Namespace)

	template := &kodev1alpha2.Template{
		Kind: ref.Kind,
		Name: ref.Name,
	}
	if ref.Namespace != nil {
		template.Namespace = *ref.Namespace
	}

	var err error
	switch ref.Kind {
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindPodTemplate):
		err = m.fetchPodTemplate(ctx, template, ref)
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterPodTemplate):
		err = m.fetchClusterPodTemplate(ctx, template, ref)
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindTofuTemplate):
		err = m.fetchTofuTemplate(ctx, template, ref)
	case kodev1alpha2.Kind(kodev1alpha2.TemplateKindClusterTofuTemplate):
		err = m.fetchClusterTofuTemplate(ctx, template, ref)
	default:
		return nil, fmt.Errorf("unknown template kind: %s", ref.Kind)
	}

	if err != nil {
		return nil, err
	}

	log.V(1).Info("Template fetched successfully", "kind", template.Kind, "port", template.Port)
	return template, nil
}

func (m *DefaultTemplateManager) fetchPodTemplate(ctx context.Context, template *kodev1alpha2.Template, ref kodev1alpha2.CrossNamespaceObjectReference) error {
	podTemplate := &kodev1alpha2.PodTemplate{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: string(template.Namespace)}, podTemplate)
	if err != nil {
		return handleNotFoundError(err, ref)
	}
	template.PodTemplateSpec = &podTemplate.Spec.PodTemplateSharedSpec
	template.Port = *podTemplate.Spec.Port
	return nil
}

func (m *DefaultTemplateManager) fetchClusterPodTemplate(ctx context.Context, template *kodev1alpha2.Template, ref kodev1alpha2.CrossNamespaceObjectReference) error {
	clusterPodTemplate := &kodev1alpha2.ClusterPodTemplate{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name)}, clusterPodTemplate)
	if err != nil {
		return handleNotFoundError(err, ref)
	}
	template.PodTemplateSpec = &clusterPodTemplate.Spec.PodTemplateSharedSpec
	template.Port = *clusterPodTemplate.Spec.Port
	return nil
}

func (m *DefaultTemplateManager) fetchTofuTemplate(ctx context.Context, template *kodev1alpha2.Template, ref kodev1alpha2.CrossNamespaceObjectReference) error {
	tofuTemplate := &kodev1alpha2.TofuTemplate{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: string(template.Namespace)}, tofuTemplate)
	if err != nil {
		return handleNotFoundError(err, ref)
	}
	template.TofuTemplateSpec = &tofuTemplate.Spec.TofuSharedSpec
	template.Port = *tofuTemplate.Spec.Port
	return nil
}

func (m *DefaultTemplateManager) fetchClusterTofuTemplate(ctx context.Context, template *kodev1alpha2.Template, ref kodev1alpha2.CrossNamespaceObjectReference) error {
	clusterTofuTemplate := &kodev1alpha2.ClusterTofuTemplate{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: string(ref.Name)}, clusterTofuTemplate)
	if err != nil {
		return handleNotFoundError(err, ref)
	}
	template.TofuTemplateSpec = &clusterTofuTemplate.Spec.TofuSharedSpec
	template.Port = *clusterTofuTemplate.Spec.Port
	return nil
}

func handleNotFoundError(err error, ref kodev1alpha2.CrossNamespaceObjectReference) error {
	if errors.IsNotFound(err) {
		namespacedName := types.NamespacedName{Name: string(ref.Name)}
		if ref.Namespace != nil {
			namespacedName.Namespace = string(*ref.Namespace)
		}
		return &koderrs.TemplateNotFoundError{
			NamespacedName: namespacedName,
			Kind:           string(ref.Kind),
		}
	}
	return err
}

func validateRef(ref kodev1alpha2.CrossNamespaceObjectReference) error {
	if ref.Kind == "" {
		return fmt.Errorf("template reference kind is empty")
	}
	if ref.Name == "" {
		return fmt.Errorf("template reference name is empty")
	}
	return nil
}
