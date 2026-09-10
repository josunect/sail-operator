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

package perses

import (
	"fmt"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	persesGroup   = "perses.dev"
	persesVersion = "v1alpha2"
)

// NewDatasourceSpec builds the mesh-related spec fields for a PersesDatasource.
func NewDatasourceSpec(mi *v1alpha1.MetricsIntegration, name string) (map[string]interface{}, error) {
	url, err := prometheusURL(mi)
	if err != nil {
		return nil, err
	}

	spec := map[string]interface{}{
		"config": map[string]interface{}{
			"display": map[string]interface{}{
				"name": "Prometheus",
			},
			"default": true,
			"plugin": map[string]interface{}{
				"kind": "PrometheusDatasource",
				"spec": map[string]interface{}{
					"proxy": map[string]interface{}{
						"kind": "HTTPProxy",
						"spec": map[string]interface{}{
							"url": url,
						},
					},
				},
			},
		},
	}

	if err := unstructured.SetNestedField(spec, datasourceSecretName(name), "config", "plugin", "spec", "proxy", "spec", "secret"); err != nil {
		return nil, fmt.Errorf("set datasource secret: %w", err)
	}

	if client := openShiftDatasourceClient(mi.Spec.Type); client != nil {
		spec["client"] = client
	}

	return spec, nil
}

// NewDatasource builds a PersesDatasource object with mesh-related fields for server-side apply.
func NewDatasource(mi *v1alpha1.MetricsIntegration, namespace, name string) (*unstructured.Unstructured, error) {
	spec, err := NewDatasourceSpec(mi, name)
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   persesGroup,
		Version: persesVersion,
		Kind:    "PersesDatasource",
	})
	obj.SetName(name)
	obj.SetNamespace(namespace)
	if err := unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
		return nil, fmt.Errorf("set datasource spec: %w", err)
	}
	return obj, nil
}

func datasourceSecretName(name string) string {
	return name + DatasourceSecretSuffix
}

func openShiftDatasourceClient(metricsType v1alpha1.MetricsType) map[string]interface{} {
	switch metricsType {
	case v1alpha1.MetricsTypeUserWorkloadMonitoring:
		return map[string]interface{}{
			"tls": map[string]interface{}{
				"enable": true,
				"caCert": map[string]interface{}{
					"type":     "file",
					"certPath": OpenShiftServiceCAPath,
				},
			},
		}
	default:
		return nil
	}
}

func prometheusURL(mi *v1alpha1.MetricsIntegration) (string, error) {
	switch mi.Spec.Type {
	case v1alpha1.MetricsTypeUserWorkloadMonitoring:
		return UWMPrometheusURL, nil
	case v1alpha1.MetricsTypeClusterObservabilityOperator:
		return "", fmt.Errorf("ClusterObservabilityOperator datasource URL resolution is not implemented yet")
	default:
		return "", fmt.Errorf("unsupported metrics type %q", mi.Spec.Type)
	}
}
