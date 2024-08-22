package validation

import (
	"context"
	"fmt"
	"strings"

	kodev1alpha2 "github.com/jacero-io/kode-operator/api/v1alpha2"
)

type validator struct{}

func NewDefaultValidator() Validator {
	return &validator{}
}

func (v *validator) ValidateKode(ctx context.Context, kode *kodev1alpha2.Kode) error {
	if err := v.validateKodeSpec(kode); err != nil {
		return err
	}
	return nil
}

func (v *validator) validateKodeSpec(kode *kodev1alpha2.Kode) error {
	var errors []string

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

func validateInitPlugin(plugin kodev1alpha2.InitPluginSpec, index int) error {
	if plugin.Image == "" {
		return fmt.Errorf("initPlugin[%d]: image is required", index)
	}

	// Add more specific validation for InitPluginSpec if needed
	// For example, validate that the image follows a valid format
	// or that the command and args are properly specified

	return nil
}
