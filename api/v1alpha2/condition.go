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

package v1alpha2

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jacero-io/kode-operator/pkg/constant"
)

type ConditionedStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (s *ConditionedStatus) SetCondition(conditionType constant.ConditionType, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}

	// Iterate through the conditions and update the condition if it already exists
	for i, condition := range s.Conditions {
		if condition.Type == string(conditionType) {
			if condition.Status != status {
				s.Conditions[i] = newCondition
			}
			return
		}
	}

	s.Conditions = append(s.Conditions, newCondition)
}

func (s *ConditionedStatus) GetCondition(conditionType constant.ConditionType) *metav1.Condition {
	for _, condition := range s.Conditions {
		if condition.Type == string(conditionType) {
			return &condition
		}
	}
	return nil
}

func (s *ConditionedStatus) DeleteCondition(conditionType constant.ConditionType) {
	conditions := []metav1.Condition{}
	for _, condition := range s.Conditions {
		if condition.Type != string(conditionType) {
			conditions = append(conditions, condition)
		}
	}
	s.Conditions = conditions
}
