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

	var err error
	switch ref.Kind {
	case "KodeTemplate":
		m.log.V(1).Info("Fetching KodeTemplate", "Name", templates.KodeTemplateName, "Namespace", templates.KodeTemplateNamespace)
		err = m.fetchKodeTemplate(ctx, templates)
	case "KodeClusterTemplate":
		m.log.V(1).Info("Fetching KodeClusterTemplate", "Name", templates.KodeTemplateName, "Namespace", templates.KodeTemplateNamespace)
		err = m.fetchKodeClusterTemplate(ctx, templates)
	default:
		return nil, fmt.Errorf("invalid template kind: %s", ref.Kind)
	}

	if err != nil {
		return nil, err
	}

	// if templates.KodeTemplate.EnvoyConfigRef.Name != "" {
	// 	m.log.V(1).Info("Fetching EnvoyProxy config", "Name", templates.KodeTemplate.EnvoyConfigRef.Name, "Namespace", templates.KodeTemplate.EnvoyConfigRef.Namespace)
	// 	if err := m.fetchEnvoyProxyConfig(ctx, templates); err != nil {
	// 		return nil, err
	// 	}
	// }

	return templates, nil
}

func (m *defaultTemplateManager) fetchTemplate(ctx context.Context, templates *common.Templates, obj client.Object, kind string) error {
	namespacedName := types.NamespacedName{Name: templates.KodeTemplateName}
	if kind == "KodeTemplate" {
		namespacedName.Namespace = templates.KodeTemplateNamespace
	}

	if err := m.client.Get(ctx, namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return &common.TemplateNotFoundError{NamespacedName: namespacedName, Kind: kind}
		}
		return err
	}

	switch t := obj.(type) {
	case *kodev1alpha1.KodeTemplate:
		templates.KodeTemplate = &t.Spec.SharedKodeTemplateSpec
	case *kodev1alpha1.KodeClusterTemplate:
		templates.KodeTemplate = &t.Spec.SharedKodeTemplateSpec
	default:
		return fmt.Errorf("unsupported template type: %T", obj)
	}

	return nil
}

func (m *defaultTemplateManager) fetchKodeTemplate(ctx context.Context, templates *common.Templates) error {
	return m.fetchTemplate(ctx, templates, &kodev1alpha1.KodeTemplate{}, "KodeTemplate")
}

func (m *defaultTemplateManager) fetchKodeClusterTemplate(ctx context.Context, templates *common.Templates) error {
	return m.fetchTemplate(ctx, templates, &kodev1alpha1.KodeClusterTemplate{}, "KodeClusterTemplate")
}

// func (m *defaultTemplateManager) fetchEnvoyConfig(ctx context.Context, templates *common.Templates, obj client.Object, kind string) error {
// 	namespacedName := types.NamespacedName{Name: templates.EnvoyProxyConfigName}
// 	if kind == "EnvoyProxyConfig" {
// 		namespacedName.Namespace = templates.EnvoyProxyConfigNamespace
// 	}

// 	if err := m.client.Get(ctx, namespacedName, obj); err != nil {
// 		if errors.IsNotFound(err) {
// 			return &common.TemplateNotFoundError{NamespacedName: namespacedName, Kind: kind}
// 		}
// 		return err
// 	}

// 	switch t := obj.(type) {
// 	case *kodev1alpha1.EnvoyProxyConfig:
// 		templates.EnvoyProxyConfig = &t.Spec.SharedEnvoyProxyConfigSpec
// 	case *kodev1alpha1.EnvoyProxyClusterConfig:
// 		templates.EnvoyProxyConfig = &t.Spec.SharedEnvoyProxyConfigSpec
// 	default:
// 		return fmt.Errorf("unsupported EnvoyProxy config type: %T", obj)
// 	}

// 	return nil
// }

// func (m *defaultTemplateManager) fetchEnvoyProxyConfig(ctx context.Context, templates *common.Templates) error {
// 	EnvoyConfigRef := templates.KodeTemplate.EnvoyConfigRef
// 	templates.EnvoyProxyConfigName = EnvoyConfigRef.Name
// 	templates.EnvoyProxyConfigNamespace = EnvoyConfigRef.Namespace

// 	if templates.EnvoyProxyConfigNamespace == "" {
// 		templates.EnvoyProxyConfigNamespace = common.DefaultNamespace
// 	}

// 	switch EnvoyConfigRef.Kind {
// 	case "EnvoyProxyConfig":
// 		return m.fetchEnvoyConfig(ctx, templates, &kodev1alpha1.EnvoyProxyConfig{}, "EnvoyProxyConfig")
// 	case "EnvoyProxyClusterConfig":
// 		return m.fetchEnvoyConfig(ctx, templates, &kodev1alpha1.EnvoyProxyClusterConfig{}, "EnvoyProxyClusterConfig")
// 	default:
// 		return fmt.Errorf("invalid EnvoyProxy config kind: %s", EnvoyConfigRef.Kind)
// 	}
// }
