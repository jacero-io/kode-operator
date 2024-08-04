// internal/envoy/errors.go

/*
Copyright 2024 Emil Larsson.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package envoy

import "fmt"

type EnvoyErrorType string

const (
	EnvoyErrorTypeConfiguration EnvoyErrorType = "Configuration"
	EnvoyErrorTypeCreation      EnvoyErrorType = "Creation"
)

// EnvoyError represents an error related to Envoy operations
type EnvoyError struct {
	Type    EnvoyErrorType
	Message string
	Err     error
}

// Error returns the error message
func (e *EnvoyError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Envoy %s error: %s - %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("Envoy %s error: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *EnvoyError) Unwrap() error {
	return e.Err
}

// NewEnvoyError creates a new EnvoyError
func NewEnvoyError(errorType EnvoyErrorType, message string, err error) *EnvoyError {
	return &EnvoyError{
		Type:    errorType,
		Message: message,
		Err:     err,
	}
}

// IsEnvoyError checks if an error is an EnvoyError
func IsEnvoyError(err error) bool {
	_, ok := err.(*EnvoyError)
	return ok
}

// GetEnvoyErrorType returns the type of an EnvoyError, or an empty string if it's not an EnvoyError
func GetEnvoyErrorType(err error) EnvoyErrorType {
	if envoyErr, ok := err.(*EnvoyError); ok {
		return envoyErr.Type
	}
	return ""
}
