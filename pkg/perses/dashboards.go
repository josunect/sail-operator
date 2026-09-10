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

const (
	PartOfLabelKey   = "app.kubernetes.io/part-of"
	PartOfLabelValue = "sail-operator"
	ManagedByLabel   = "app.kubernetes.io/managed-by"
	ManagedByValue   = "sail-operator"

	PersesDatasourceCRD = "persesdatasources.perses.dev"
	PersesDashboardCRD  = "persesdashboards.perses.dev"

	// DefaultDatasourceName is the default PersesDatasource name in community-mixins dashboards.
	DefaultDatasourceName = "prometheus-datasource"

	// UWMPrometheusURL is the OpenShift User Workload Monitoring Thanos querier endpoint.
	UWMPrometheusURL = "https://thanos-querier.openshift-monitoring.svc.cluster.local:9091"

	// OpenShiftServiceCAPath is mounted by the COO/Perses operand for in-cluster TLS.
	OpenShiftServiceCAPath = "/ca/service-ca.crt"

	// DatasourceSecretSuffix is appended to the PersesDatasource name for HTTPProxy auth.
	// The perses-operator provisions the corresponding Perses secret automatically.
	DatasourceSecretSuffix = "-secret"

	// FieldOwner is the server-side apply field manager for Perses resources.
	FieldOwner = "sail-operator"
)

// DashboardDefinition maps a product dashboard to its PersesDashboard CR metadata.name.
type DashboardDefinition struct {
	Name     string
	Filename string
}

// ProductDashboards is the supported GA dashboard set (community-mixins operator YAML).
var ProductDashboards = []DashboardDefinition{
	{Name: "istio-control-plane", Filename: "istio-control-plane.yaml"},
	{Name: "istio-mesh-dashboard", Filename: "istio-mesh-dashboard.yaml"},
	{Name: "istio-performance", Filename: "istio-performance.yaml"},
	{Name: "istio-service-dashboard", Filename: "istio-service-dashboard.yaml"},
	{Name: "istio-workload-dashboard", Filename: "istio-workload-dashboard.yaml"},
	{Name: "istio-ztunnel-dashboard", Filename: "istio-ztunnel-dashboard.yaml"},
}
