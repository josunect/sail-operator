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
	"testing"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewDatasourceOpenShiftUWM(t *testing.T) {
	mi := &v1alpha1.MetricsIntegration{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v1alpha1.MetricsIntegrationSpec{
			Type: v1alpha1.MetricsTypeUserWorkloadMonitoring,
		},
	}

	ds, err := NewDatasource(mi, "monitoring", v1alpha1.DefaultPersesDatasourceName)
	if err != nil {
		t.Fatalf("NewDatasource() error = %v", err)
	}

	url, found, err := unstructured.NestedString(ds.Object, "spec", "config", "plugin", "spec", "proxy", "spec", "url")
	if err != nil || !found {
		t.Fatalf("expected datasource url: found=%v err=%v", found, err)
	}
	if url != UWMPrometheusURL {
		t.Fatalf("unexpected url: %q", url)
	}

	secret, found, err := unstructured.NestedString(ds.Object, "spec", "config", "plugin", "spec", "proxy", "spec", "secret")
	if err != nil || !found {
		t.Fatalf("expected datasource secret: found=%v err=%v", found, err)
	}
	if secret != "prometheus-datasource-secret" {
		t.Fatalf("unexpected secret: %q", secret)
	}

	tlsEnabled, found, err := unstructured.NestedBool(ds.Object, "spec", "client", "tls", "enable")
	if err != nil || !found || !tlsEnabled {
		t.Fatalf("expected client TLS to be enabled: found=%v enabled=%v err=%v", found, tlsEnabled, err)
	}

	caType, found, err := unstructured.NestedString(ds.Object, "spec", "client", "tls", "caCert", "type")
	if err != nil || !found || caType != "file" {
		t.Fatalf("unexpected CA type: found=%v type=%q err=%v", found, caType, err)
	}

	caPath, found, err := unstructured.NestedString(ds.Object, "spec", "client", "tls", "caCert", "certPath")
	if err != nil || !found || caPath != OpenShiftServiceCAPath {
		t.Fatalf("unexpected CA path: found=%v path=%q err=%v", found, caPath, err)
	}
}
