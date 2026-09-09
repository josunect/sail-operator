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

package metricsintegration

import (
	"context"
	"testing"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	"github.com/istio-ecosystem/sail-operator/pkg/config"
	"github.com/istio-ecosystem/sail-operator/pkg/perses"
	"github.com/istio-ecosystem/sail-operator/pkg/scheme"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileNoPersesTarget(t *testing.T) {
	ctx := context.Background()
	mi := &v1alpha1.MetricsIntegration{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v1alpha1.MetricsIntegrationSpec{
			TargetRefs: []v1alpha1.TargetReference{{
				Kind: "Istio",
				Name: "default",
			}},
			MetricsConfig: v1alpha1.MetricsConfig{
				Type:                   v1alpha1.MetricsTypeUserWorkloadMonitoring,
				UserWorkloadMonitoring: &v1alpha1.UserWorkloadMonitoringConfig{},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithStatusSubresource(&v1alpha1.MetricsIntegration{}).
		WithObjects(mi).
		Build()
	r := NewReconciler(config.ReconcilerConfig{}, cl, scheme.Scheme)

	if _, err := r.Reconcile(ctx, mi); err == nil {
		t.Fatal("expected validation error when Perses target is missing")
	}

	got := &v1alpha1.MetricsIntegration{}
	if err := cl.Get(ctx, types.NamespacedName{Name: mi.Name}, got); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	reconciled := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionReconciled)
	if reconciled.Status != metav1.ConditionFalse {
		t.Fatalf("expected Reconciled=False, got %s", reconciled.Status)
	}
	if reconciled.Reason != v1alpha1.MetricsIntegrationReasonNoPersesTarget {
		t.Fatalf("expected reason %q, got %q", v1alpha1.MetricsIntegrationReasonNoPersesTarget, reconciled.Reason)
	}
}

func TestReconcileMissingPersesCRDs(t *testing.T) {
	ctx := context.Background()
	mi := metricsIntegrationWithPerses("monitoring")

	cl := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithStatusSubresource(&v1alpha1.MetricsIntegration{}).
		WithObjects(mi, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "monitoring"}}).
		Build()
	r := NewReconciler(config.ReconcilerConfig{}, cl, scheme.Scheme)

	if _, err := r.Reconcile(ctx, mi); err == nil {
		t.Fatal("expected transient error when Perses CRDs are missing")
	}

	got := &v1alpha1.MetricsIntegration{}
	if err := cl.Get(ctx, types.NamespacedName{Name: mi.Name}, got); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	reconciled := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionReconciled)
	if reconciled.Status != metav1.ConditionFalse {
		t.Fatalf("expected Reconciled=False, got %s", reconciled.Status)
	}
	if reconciled.Reason != v1alpha1.MetricsIntegrationReasonMissingCRDs {
		t.Fatalf("expected reason %q, got %q", v1alpha1.MetricsIntegrationReasonMissingCRDs, reconciled.Reason)
	}

	persesCond := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionPersesAvailable)
	if persesCond.Status != metav1.ConditionFalse || persesCond.Reason != v1alpha1.MetricsIntegrationReasonMissingCRDs {
		t.Fatalf("unexpected PersesAvailable condition: %+v", persesCond)
	}
}

func TestReconcilePersesProvisioning(t *testing.T) {
	ctx := context.Background()
	mi := metricsIntegrationWithPerses("monitoring")
	mi.Spec.Perses = &v1alpha1.PersesProvisioningConfig{
		Dashboards: []v1alpha1.IstioPersesDashboard{v1alpha1.IstioPersesDashboardControlPlane},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithStatusSubresource(&v1alpha1.MetricsIntegration{}).
		WithObjects(
			mi,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "monitoring"}},
			persesCRD(perses.PersesDatasourceCRD, "PersesDatasource"),
			persesCRD(perses.PersesDashboardCRD, "PersesDashboard"),
		).
		Build()
	r := NewReconciler(config.ReconcilerConfig{}, cl, scheme.Scheme)

	_, err := r.Reconcile(ctx, mi)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	got := &v1alpha1.MetricsIntegration{}
	if err := cl.Get(ctx, types.NamespacedName{Name: mi.Name}, got); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	reconciled := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionReconciled)
	if reconciled.Status != metav1.ConditionTrue {
		t.Fatalf("expected Reconciled=True, got %+v", reconciled)
	}
	persesCond := got.Status.GetCondition(v1alpha1.MetricsIntegrationConditionPersesAvailable)
	if persesCond.Status != metav1.ConditionTrue {
		t.Fatalf("expected PersesAvailable=True, got %+v", persesCond)
	}
}

func metricsIntegrationWithPerses(namespace string) *v1alpha1.MetricsIntegration {
	return &v1alpha1.MetricsIntegration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-perses",
			UID:  "test-perses-uid",
		},
		Spec: v1alpha1.MetricsIntegrationSpec{
			TargetRefs: []v1alpha1.TargetReference{{
				Kind:      "Perses",
				Name:      "perses",
				Namespace: namespace,
			}},
			MetricsConfig: v1alpha1.MetricsConfig{
				Type:                   v1alpha1.MetricsTypeUserWorkloadMonitoring,
				UserWorkloadMonitoring: &v1alpha1.UserWorkloadMonitoringConfig{},
			},
		},
	}
}

func persesCRD(name, kind string) *apiextensionsv1.CustomResourceDefinition {
	preserve := true
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "perses.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     kind,
				ListKind: kind + "List",
				Plural:   name[:len(name)-len(".perses.dev")],
				Singular: kind,
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha2",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type:                   "object",
						XPreserveUnknownFields: &preserve,
					},
				},
			}},
		},
	}
}
