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
)

func TestResolveDashboards(t *testing.T) {
	selected := []v1alpha1.IstioPersesDashboard{
		v1alpha1.IstioPersesDashboardMesh,
		v1alpha1.IstioPersesDashboardService,
	}
	got := ResolveDashboards(selected)
	if len(got) != 2 {
		t.Fatalf("expected 2 dashboards, got %d", len(got))
	}
	if got[0].Name != "istio-mesh-dashboard" || got[1].Name != "istio-service-dashboard" {
		t.Fatalf("unexpected dashboard names: %+v", got)
	}
}

func TestSelectedDashboardsDefault(t *testing.T) {
	spec := v1alpha1.MetricsIntegrationSpec{
		MetricsConfig: v1alpha1.MetricsConfig{
			Type: v1alpha1.MetricsTypeUserWorkloadMonitoring,
		},
	}
	if len(spec.SelectedDashboards()) != 6 {
		t.Fatalf("expected 6 default dashboards, got %d", len(spec.SelectedDashboards()))
	}
}
