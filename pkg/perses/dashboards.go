// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package perses

import (
	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
)

const (
	PartOfLabelKey   = "app.kubernetes.io/part-of"
	PartOfLabelValue = "sail-operator"
	ManagedByLabel   = "app.kubernetes.io/managed-by"
	ManagedByValue   = "sail-operator"

	PersesDatasourceCRD = "persesdatasources.perses.dev"
	PersesDashboardCRD  = "persesdashboards.perses.dev"

	// UWMPrometheusURL is the OpenShift User Workload Monitoring Thanos querier endpoint.
	UWMPrometheusURL = "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091"
)

// DashboardDefinition maps a product dashboard name to its PersesDashboard CR metadata.name.
type DashboardDefinition struct {
	Product  v1alpha1.IstioPersesDashboard
	Name     string
	Filename string
}

// ProductDashboards is the supported GA dashboard set (community-mixins operator YAML).
var ProductDashboards = []DashboardDefinition{
	{Product: v1alpha1.IstioPersesDashboardControlPlane, Name: "istio-control-plane", Filename: "istio-control-plane.yaml"},
	{Product: v1alpha1.IstioPersesDashboardMesh, Name: "istio-mesh-dashboard", Filename: "istio-mesh-dashboard.yaml"},
	{Product: v1alpha1.IstioPersesDashboardPerformance, Name: "istio-performance", Filename: "istio-performance.yaml"},
	{Product: v1alpha1.IstioPersesDashboardService, Name: "istio-service-dashboard", Filename: "istio-service-dashboard.yaml"},
	{Product: v1alpha1.IstioPersesDashboardWorkload, Name: "istio-workload-dashboard", Filename: "istio-workload-dashboard.yaml"},
	{Product: v1alpha1.IstioPersesDashboardZtunnel, Name: "istio-ztunnel-dashboard", Filename: "istio-ztunnel-dashboard.yaml"},
}

// ResolveDashboards returns dashboard definitions for the requested product names.
func ResolveDashboards(selected []v1alpha1.IstioPersesDashboard) []DashboardDefinition {
	want := make(map[v1alpha1.IstioPersesDashboard]struct{}, len(selected))
	for _, d := range selected {
		want[d] = struct{}{}
	}
	var out []DashboardDefinition
	for _, def := range ProductDashboards {
		if _, ok := want[def.Product]; ok {
			out = append(out, def)
		}
	}
	return out
}
