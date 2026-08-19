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

// TargetReference identifies a resource that an Integration configures or a Perses
// project namespace where the controller provisions PersesDatasource and PersesDashboard resources.
type TargetReference struct {
	// Kind specifies the target kind: "Istio", "Kiali", or "Perses".
	// +kubebuilder:validation:Enum=Istio;Kiali;Perses
	Kind string `json:"kind"`

	// Name is the name of the target resource.
	Name string `json:"name"`

	// Namespace is the namespace of the target resource.
	// Required for namespace-scoped resources like Kiali and for Perses (project namespace).
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// NamespacedReference is a reference to a namespaced resource.
type NamespacedReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
