package template

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultTemplateManager struct {
	client client.Client
	log    logr.Logger
}

func NewDefaultTemplateManager(client client.Client, log logr.Logger) TemplateManager {
	return &defaultTemplateManager{
		client: client,
		log:    log,
	}
}

func (m *defaultTemplateManager) Fetch(ctx context.Context, ref kodev1alpha1.KodeTemplateReference) (*common.Templates, error) {
	templates := &common.Templates{
		KodeTemplateName:      ref.Name,
		KodeTemplateNamespace: ref.Namespace,
	}

	// If no namespace is provided, use the default namespace
	if templates.KodeTemplateNamespace == "" {
		templates.KodeTemplateNamespace = common.DefaultNamespace
	}

	switch ref.Kind {
	case "KodeTemplate":
		if err := m.fetchKodeTemplate(ctx, templates); err != nil {
			return nil, err
		}
	case "KodeClusterTemplate":
		if err := m.fetchKodeClusterTemplate(ctx, templates); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid template kind: %s", ref.Kind)
	}

	if templates.KodeTemplate.EnvoyConfigRef.Name != "" {
		if err := m.fetchEnvoyProxyConfig(ctx, templates); err != nil {
			return nil, err
		}
	}

	return templates, nil
}

func (m *defaultTemplateManager) fetchKodeTemplate(ctx context.Context, templates *common.Templates) error {
	template := &kodev1alpha1.KodeTemplate{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: templates.KodeTemplateName, Namespace: templates.KodeTemplateNamespace}, template); err != nil {
		if errors.IsNotFound(err) {
			return &common.TemplateNotFoundError{NamespacedName: types.NamespacedName{Name: templates.KodeTemplateName, Namespace: templates.KodeTemplateNamespace}, Kind: "KodeTemplate"}
		}
		return err
	}
	templates.KodeTemplate = &template.Spec.SharedKodeTemplateSpec
	return nil
}

func (m *defaultTemplateManager) fetchKodeClusterTemplate(ctx context.Context, templates *common.Templates) error {
	template := &kodev1alpha1.KodeClusterTemplate{}
	if err := m.client.Get(ctx, types.NamespacedName{Name: templates.KodeTemplateName}, template); err != nil {
		if errors.IsNotFound(err) {
			return &common.TemplateNotFoundError{NamespacedName: types.NamespacedName{Name: templates.KodeTemplateName}, Kind: "KodeClusterTemplate"}
		}
		return err
	}
	templates.KodeTemplate = &template.Spec.SharedKodeTemplateSpec
	return nil
}

func (m *defaultTemplateManager) fetchEnvoyProxyConfig(ctx context.Context, templates *common.Templates) error {
	EnvoyConfigRef := templates.KodeTemplate.EnvoyConfigRef
	templates.EnvoyProxyConfigName = EnvoyConfigRef.Name
	templates.EnvoyProxyConfigNamespace = EnvoyConfigRef.Namespace

	// If no namespace is provided, use the default namespace
	if templates.EnvoyProxyConfigNamespace == "" {
		templates.EnvoyProxyConfigNamespace = common.DefaultNamespace
	}

	switch EnvoyConfigRef.Kind {
	case "EnvoyProxyConfig":
		config := &kodev1alpha1.EnvoyProxyConfig{}
		if err := m.client.Get(ctx, types.NamespacedName{Name: templates.EnvoyProxyConfigName, Namespace: templates.EnvoyProxyConfigNamespace}, config); err != nil {
			if errors.IsNotFound(err) {
				return &common.TemplateNotFoundError{NamespacedName: types.NamespacedName{Name: templates.EnvoyProxyConfigName, Namespace: templates.EnvoyProxyConfigNamespace}, Kind: "EnvoyProxyConfig"}
			}
			return err
		}
		templates.EnvoyProxyConfig = &config.Spec.SharedEnvoyProxyConfigSpec
	case "EnvoyProxyClusterConfig":
		config := &kodev1alpha1.EnvoyProxyClusterConfig{}
		if err := m.client.Get(ctx, types.NamespacedName{Name: templates.EnvoyProxyConfigName}, config); err != nil {
			if errors.IsNotFound(err) {
				return &common.TemplateNotFoundError{NamespacedName: types.NamespacedName{Name: templates.EnvoyProxyConfigName}, Kind: "EnvoyProxyClusterConfig"}
			}
			return err
		}
		templates.EnvoyProxyConfig = &config.Spec.SharedEnvoyProxyConfigSpec
	default:
		return fmt.Errorf("invalid EnvoyProxy config kind: %s", EnvoyConfigRef.Kind)
	}

	return nil
}
