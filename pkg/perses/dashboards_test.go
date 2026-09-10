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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestProductDashboards(t *testing.T) {
	if len(ProductDashboards) != 6 {
		t.Fatalf("expected 6 dashboards, got %d", len(ProductDashboards))
	}
}

func TestBundledDashboardsHaveSpecConfig(t *testing.T) {
	for _, def := range ProductDashboards {
		raw, err := LoadDashboardYAML(def)
		if err != nil {
			t.Fatalf("LoadDashboardYAML(%s): %v", def.Filename, err)
		}
		obj, err := ParseDashboard(raw)
		if err != nil {
			t.Fatalf("ParseDashboard(%s): %v", def.Filename, err)
		}
		config, found, err := unstructured.NestedMap(obj.Object, "spec", "config")
		if err != nil {
			t.Fatalf("NestedMap(%s): %v", def.Filename, err)
		}
		if !found || len(config) == 0 {
			t.Fatalf("dashboard %s must define spec.config", def.Filename)
		}
	}
}
