package validation

import (
	"context"
	"fmt"
	"strings"

	kodev1alpha1 "github.com/jacero-io/kode-operator/api/v1alpha1"
	"github.com/jacero-io/kode-operator/internal/common"
	"k8s.io/apimachinery/pkg/api/resource"
)

type validator struct{}

func NewDefaultValidator() Validator {
	return &validator{}
}

func (v *validator) ValidateKode(ctx context.Context, kode *kodev1alpha1.Kode, templates *common.Templates) error {
	if err := v.validateKodeSpec(kode); err != nil {
		return err
	}
	return nil
}

func (v *validator) validateKodeSpec(kode *kodev1alpha1.Kode) error {
	var errors []string

	// Validate TemplateRef
	if err := validateTemplateRef(kode.Spec.TemplateRef); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate User
	if err := validateUser(kode.Spec.Username); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate Password (if set)
	if kode.Spec.Password != "" {
		if err := validatePassword(kode.Spec.Password); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate ExistingSecret (if set)
	if kode.Spec.ExistingSecret != "" {
		// Secret must contain both username and password in Data
		if kode.Spec.Username == "" || kode.Spec.Password == "" {
			errors = append(errors, "existingSecret is specified, but username and password are not set")
		}
	}

	// Validate Home (if set)
	if kode.Spec.Home != "" {
		if err := validateHome(kode.Spec.Home); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate Workspace (if set)
	if kode.Spec.Workspace != "" {
		if err := validateWorkspace(kode.Spec.Workspace); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate Storage (if set)
	if !kode.Spec.Storage.IsEmpty() {
		if err := validateStorage(kode.Spec.Storage); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate InitPlugins (if any)
	for i, plugin := range kode.Spec.InitPlugins {
		if err := validateInitPlugin(plugin, i); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("kode spec validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate exisitingSecret
func validateExistingSecret(ctx context.Context, secretName string) error {
	if secretName == "" {
		return fmt.Errorf("existingSecret is required")
	}
	// Secret must contain both username and password in Data
	// secret := &corev1.Secret{}
	// if err := common.Get(ctx, client.ObjectKey{Name: secretName, Namespace: common.Namespace}, secret); err != nil {
	// 	return fmt.Errorf("failed to get Secret: %v", err)
	// }


	return nil

}

func validateTemplateRef(ref kodev1alpha1.KodeTemplateReference) error {
	if ref.Name == "" {
		return fmt.Errorf("templateRef.name is required")
	}
	if ref.Kind != "KodeTemplate" && ref.Kind != "KodeClusterTemplate" {
		return fmt.Errorf("invalid templateRef.kind: %s, must be either KodeTemplate or KodeClusterTemplate", ref.Kind)
	}
	return nil
}

func validateKodeTemplateSpec(spec kodev1alpha1.KodeTemplateSpec) error {
	if spec.Type != "code-server" && spec.Type != "webtop" {
		return fmt.Errorf("invalid template type: %s, must be either code-server or webtop", spec.Type)
	}
	if spec.DefaultHome == "" {
		return fmt.Errorf("defaultHome is required")
	}
	if spec.DefaultWorkspace == "" {
		return fmt.Errorf("defaultWorkspace is required")
	}

	return nil
}

func validateUser(user string) error {
	if user == "" {
		return fmt.Errorf("user is required")
	}
	if len(user) < 3 || len(user) > 32 {
		return fmt.Errorf("user must be between 3 and 32 characters long")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	return nil
}

func validateHome(home string) error {
	if !strings.HasPrefix(home, "/") {
		return fmt.Errorf("home must be an absolute path starting with '/'")
	}
	return nil
}

func validateWorkspace(workspace string) error {
	if strings.HasPrefix(workspace, "/") {
		return fmt.Errorf("workspace must be a relative path, not starting with '/'")
	}
	return nil
}

func validateStorage(storage kodev1alpha1.KodeStorageSpec) error {
	if len(storage.AccessModes) == 0 {
		return fmt.Errorf("at least one storage access mode is required")
	}

	if storage.Resources.Requests == nil || storage.Resources.Requests.Storage() == nil {
		return fmt.Errorf("storage resource request is required")
	}

	storageRequest := storage.Resources.Requests.Storage()
	if storageRequest.IsZero() {
		return fmt.Errorf("storage request quantity must be greater than zero")
	}

	minStorage := resource.MustParse("1Gi")
	if storageRequest.Cmp(minStorage) < 0 {
		return fmt.Errorf("storage request must be at least 1Gi")
	}

	return nil
}

func validateInitPlugin(plugin kodev1alpha1.InitPluginSpec, index int) error {
	if plugin.Image == "" {
		return fmt.Errorf("initPlugin[%d]: image is required", index)
	}

	// Add more specific validation for InitPluginSpec if needed
	// For example, validate that the image follows a valid format
	// or that the command and args are properly specified

	return nil
}
