// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
)

const MetricsIntegrationKind = "MetricsIntegration"

// MetricsIntegrationSpec defines the desired state of MetricsIntegration.
type MetricsIntegrationSpec struct {
	// TargetRefs specifies the resources that this integration configures.
	// +kubebuilder:validation:MinItems=1
	TargetRefs []TargetReference `json:"targetRefs"`

	MetricsConfig `json:",inline"`
}

// MetricsType identifies the type of metrics integration.
// +kubebuilder:validation:Enum=UserWorkloadMonitoring;ClusterObservabilityOperator
type MetricsType string

const (
	MetricsTypeUserWorkloadMonitoring       MetricsType = "UserWorkloadMonitoring"
	MetricsTypeClusterObservabilityOperator MetricsType = "ClusterObservabilityOperator"
)

// MetricsConfig configures a metrics backend.
type MetricsConfig struct {
	// Type specifies the metrics integration type.
	Type MetricsType `json:"type"`

	// UserWorkloadMonitoring configures integration with OpenShift User Workload Monitoring.
	// +optional
	UserWorkloadMonitoring *UserWorkloadMonitoringConfig `json:"userWorkloadMonitoring,omitempty"`

	// ClusterObservabilityOperator configures integration with the Cluster Observability
	// Operator's MonitoringStack resource for metrics collection.
	// +optional
	ClusterObservabilityOperator *ClusterObservabilityOperatorConfig `json:"clusterObservabilityOperator,omitempty"`
}

// UserWorkloadMonitoringConfig configures the User Workload Monitoring integration.
type UserWorkloadMonitoringConfig struct{}

// ClusterObservabilityOperatorConfig configures the Cluster Observability Operator integration.
type ClusterObservabilityOperatorConfig struct {
	// MonitoringStackRef is a reference to a MonitoringStack resource that defines
	// the Prometheus stack used for scraping Istio metrics.
	MonitoringStackRef NamespacedReference `json:"monitoringStackRef"`
}

// MetricsIntegrationStatus defines the observed state of MetricsIntegration.
type MetricsIntegrationStatus struct {
	ObservedGeneration int64                `json:"observedGeneration,omitempty"`
	Conditions         []v1.StatusCondition `json:"conditions,omitempty"`
}

func (s *MetricsIntegrationStatus) GetCondition(conditionType MetricsIntegrationConditionType) v1.StatusCondition {
	if s != nil {
		return v1.GetCondition(s.Conditions, v1.ConditionType(conditionType))
	}
	return v1.StatusCondition{Type: v1.ConditionType(conditionType), Status: metav1.ConditionUnknown}
}

func (s *MetricsIntegrationStatus) SetCondition(condition v1.StatusCondition) {
	v1.SetCondition(&s.Conditions, condition)
}

// MetricsIntegrationConditionType represents the type of a MetricsIntegration condition.
type MetricsIntegrationConditionType string

// MetricsIntegrationConditionReason is an alias for ConditionReason.
type MetricsIntegrationConditionReason = v1.ConditionReason

const (
	MetricsIntegrationConditionReconciled      MetricsIntegrationConditionType = "Reconciled"
	MetricsIntegrationConditionPersesAvailable MetricsIntegrationConditionType = "PersesAvailable"

	MetricsIntegrationReasonReconcileError     MetricsIntegrationConditionReason = "ReconcileError"
	MetricsIntegrationReasonMissingCRDs        MetricsIntegrationConditionReason = "MissingCRDs"
	MetricsIntegrationReasonDatasourceNotFound MetricsIntegrationConditionReason = "DatasourceNotFound"
	MetricsIntegrationReasonNamespaceNotFound  MetricsIntegrationConditionReason = "NamespaceNotFound"
	MetricsIntegrationReasonHealthy            MetricsIntegrationConditionReason = "Healthy"
)

// PersesDatasourceTargets returns all PersesDatasource target references.
func (s *MetricsIntegrationSpec) PersesDatasourceTargets() []TargetReference {
	var refs []TargetReference
	for _, ref := range s.TargetRefs {
		if ref.Kind == "PersesDatasource" {
			refs = append(refs, ref)
		}
	}
	return refs
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=istio-io
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="Metrics integration type."
// +kubebuilder:printcolumn:name="Reconciled",type="string",JSONPath=".status.conditions[?(@.type==\"Reconciled\")].status",description="Whether reconciliation succeeded."
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="The age of the object."

// MetricsIntegration configures metrics observability integrations for Istio.
type MetricsIntegration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricsIntegrationSpec   `json:"spec,omitempty"`
	Status MetricsIntegrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MetricsIntegrationList contains a list of MetricsIntegration.
type MetricsIntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsIntegration `json:"items"`
}
