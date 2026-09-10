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
	"context"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	"github.com/istio-ecosystem/sail-operator/pkg/reconciler"
	"github.com/istio-ecosystem/sail-operator/pkg/scheme"
)

func TestReconcileAppliesDatasourceAndDashboards(t *testing.T) {
	ctx := context.Background()
	namespace := "monitoring"
	mi := testMetricsIntegration(namespace)
	datasource := testUserDatasource(namespace, DefaultDatasourceName)

	cl := newPersesTestClient(t, mi, datasource, testPersesCRDs()[0], testPersesCRDs()[1])
	r := &Reconciler{Client: cl, Scheme: scheme.Scheme}

	result, err := r.Reconcile(ctx, ReconcileInput{Integration: mi, Targets: mi.Spec.PersesDatasourceTargets()})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !result.CRDsAvailable {
		t.Fatal("expected CRDs to be available")
	}

	ds := &unstructured.Unstructured{}
	ds.SetGroupVersionKind(datasourceGVK)
	if err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: DefaultDatasourceName}, ds); err != nil {
		t.Fatalf("expected datasource to exist: %v", err)
	}
	if len(ds.GetOwnerReferences()) != 0 {
		t.Fatal("expected datasource to remain user-owned without owner references")
	}

	secret, found, err := unstructured.NestedString(ds.Object, "spec", "config", "plugin", "spec", "proxy", "spec", "secret")
	if err != nil || !found {
		t.Fatalf("expected datasource proxy secret: found=%v err=%v", found, err)
	}
	if secret != "prometheus-datasource-secret" {
		t.Fatalf("unexpected datasource secret: %q", secret)
	}

	tlsEnabled, found, err := unstructured.NestedBool(ds.Object, "spec", "client", "tls", "enable")
	if err != nil || !found || !tlsEnabled {
		t.Fatalf("expected datasource client TLS: found=%v enabled=%v err=%v", found, tlsEnabled, err)
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Group: persesGroup, Version: persesVersion, Kind: "PersesDashboardList"})
	if err := cl.List(ctx, list, client.InNamespace(namespace)); err != nil {
		t.Fatalf("list dashboards: %v", err)
	}
	if len(list.Items) != len(ProductDashboards) {
		t.Fatalf("expected %d dashboards, got %d", len(ProductDashboards), len(list.Items))
	}
}

func TestReconcileUpdatesExistingDashboard(t *testing.T) {
	ctx := context.Background()
	namespace := "monitoring"
	mi := testMetricsIntegration(namespace)
	datasource := testUserDatasource(namespace, DefaultDatasourceName)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(dashboardGVK)
	existing.SetName("istio-control-plane")
	existing.SetNamespace(namespace)
	existing.SetLabels(map[string]string{
		PartOfLabelKey: PartOfLabelValue,
		ManagedByLabel: ManagedByValue,
		"stale":        "true",
	})
	if err := unstructured.SetNestedMap(existing.Object, map[string]interface{}{
		"display": map[string]interface{}{"name": "stale"},
	}, "spec", "config"); err != nil {
		t.Fatalf("set nested map: %v", err)
	}

	cl := newPersesTestClient(t, mi, datasource, testPersesCRDs()[0], testPersesCRDs()[1], existing)
	r := &Reconciler{Client: cl, Scheme: scheme.Scheme}

	if _, err := r.Reconcile(ctx, ReconcileInput{Integration: mi, Targets: mi.Spec.PersesDatasourceTargets()}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(dashboardGVK)
	if err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: "istio-control-plane"}, got); err != nil {
		t.Fatalf("get dashboard: %v", err)
	}
	display, _, _ := unstructured.NestedString(got.Object, "spec", "config", "display", "name")
	if display != "Istio Control Plane Dashboard" {
		t.Fatalf("expected dashboard display name to be refreshed, got %q", display)
	}
}

func TestReconcilePrunesRemovedDashboards(t *testing.T) {
	ctx := context.Background()
	namespace := "monitoring"
	mi := testMetricsIntegration(namespace)
	datasource := testUserDatasource(namespace, DefaultDatasourceName)

	owned := &unstructured.Unstructured{}
	owned.SetGroupVersionKind(dashboardGVK)
	owned.SetName("istio-control-plane")
	owned.SetNamespace(namespace)
	owned.SetLabels(map[string]string{PartOfLabelKey: PartOfLabelValue, ManagedByLabel: ManagedByValue})
	_ = unstructured.SetNestedMap(owned.Object, map[string]interface{}{
		"display": map[string]interface{}{"name": "keep"},
	}, "spec", "config")
	_ = controllerutilSetOwnerRef(mi, owned)

	pruned := &unstructured.Unstructured{}
	pruned.SetGroupVersionKind(dashboardGVK)
	pruned.SetName("legacy-dashboard")
	pruned.SetNamespace(namespace)
	pruned.SetLabels(map[string]string{PartOfLabelKey: PartOfLabelValue, ManagedByLabel: ManagedByValue})
	_ = unstructured.SetNestedMap(pruned.Object, map[string]interface{}{
		"display": map[string]interface{}{"name": "remove"},
	}, "spec", "config")
	_ = controllerutilSetOwnerRef(mi, pruned)

	cl := newPersesTestClient(t, mi, datasource, testPersesCRDs()[0], testPersesCRDs()[1], owned, pruned)
	r := &Reconciler{Client: cl, Scheme: scheme.Scheme}

	if _, err := r.Reconcile(ctx, ReconcileInput{Integration: mi, Targets: mi.Spec.PersesDatasourceTargets()}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	remaining := &unstructured.UnstructuredList{}
	remaining.SetGroupVersionKind(schema.GroupVersionKind{Group: persesGroup, Version: persesVersion, Kind: "PersesDashboardList"})
	if err := cl.List(ctx, remaining, client.InNamespace(namespace)); err != nil {
		t.Fatalf("list dashboards: %v", err)
	}
	if len(remaining.Items) != len(ProductDashboards) {
		t.Fatalf("expected %d dashboards after prune, got %d", len(ProductDashboards), len(remaining.Items))
	}
	for _, item := range remaining.Items {
		if item.GetName() == "legacy-dashboard" {
			t.Fatal("expected legacy dashboard to be pruned")
		}
	}
}

func TestReconcileMissingDatasource(t *testing.T) {
	ctx := context.Background()
	mi := testMetricsIntegration("monitoring")

	cl := newPersesTestClient(t, mi, testPersesCRDs()[0], testPersesCRDs()[1])
	r := &Reconciler{Client: cl, Scheme: scheme.Scheme}

	_, err := r.Reconcile(ctx, ReconcileInput{Integration: mi, Targets: mi.Spec.PersesDatasourceTargets()})
	if err == nil {
		t.Fatal("expected error when datasource is missing")
	}
	if !reconciler.IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestReconcileMissingCRDs(t *testing.T) {
	ctx := context.Background()
	namespace := "monitoring"
	mi := testMetricsIntegration(namespace)
	datasource := testUserDatasource(namespace, DefaultDatasourceName)

	cl := newPersesTestClient(t, mi, datasource)
	r := &Reconciler{Client: cl, Scheme: scheme.Scheme}

	result, err := r.Reconcile(ctx, ReconcileInput{Integration: mi, Targets: mi.Spec.PersesDatasourceTargets()})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.CRDsAvailable {
		t.Fatal("expected CRDs to be unavailable")
	}
}

func testMetricsIntegration(namespace string) *v1alpha1.MetricsIntegration {
	return &v1alpha1.MetricsIntegration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-perses",
			UID:  "test-perses-uid",
		},
		Spec: v1alpha1.MetricsIntegrationSpec{
			TargetRefs: []v1alpha1.TargetReference{{
				Kind:      "PersesDatasource",
				Name:      DefaultDatasourceName,
				Namespace: namespace,
			}},
			MetricsConfig: v1alpha1.MetricsConfig{
				Type:                   v1alpha1.MetricsTypeUserWorkloadMonitoring,
				UserWorkloadMonitoring: &v1alpha1.UserWorkloadMonitoringConfig{},
			},
		},
	}
}

func testUserDatasource(namespace, name string) *unstructured.Unstructured {
	ds := &unstructured.Unstructured{}
	ds.SetGroupVersionKind(datasourceGVK)
	ds.SetName(name)
	ds.SetNamespace(namespace)
	ds.Object["spec"] = map[string]interface{}{}
	return ds
}

func testPersesCRDs() []*apiextensionsv1.CustomResourceDefinition {
	return []*apiextensionsv1.CustomResourceDefinition{
		testCRD(PersesDatasourceCRD, "PersesDatasource"),
		testCRD(PersesDashboardCRD, "PersesDashboard"),
	}
}

func testCRD(name, kind string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: persesGroup,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     kind,
				ListKind: kind + "List",
				Plural:   name[:len(name)-len(".perses.dev")],
				Singular: kind,
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    persesVersion,
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object", XPreserveUnknownFields: ptr(true)},
				},
			}},
		},
	}
}

func ptr(b bool) *bool { return &b }

func newPersesTestClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
	for _, obj := range objs {
		builder = builder.WithObjects(obj)
	}
	return builder.Build()
}

func controllerutilSetOwnerRef(owner *v1alpha1.MetricsIntegration, obj *unstructured.Unstructured) error {
	return unstructured.SetNestedSlice(obj.Object, []interface{}{
		map[string]interface{}{
			"apiVersion": "sailoperator.io/v1alpha1",
			"kind":       "MetricsIntegration",
			"name":       owner.Name,
			"uid":        string(owner.UID),
			"controller": true,
		},
	}, "metadata", "ownerReferences")
}
