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
