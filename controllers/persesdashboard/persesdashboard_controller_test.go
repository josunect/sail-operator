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

package persesdashboard

import (
	"context"
	"os"
	"path"
	"testing"

	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
	"github.com/istio-ecosystem/sail-operator/pkg/config"
	"github.com/istio-ecosystem/sail-operator/pkg/constants"
	"github.com/istio-ecosystem/sail-operator/pkg/perses"
	"github.com/istio-ecosystem/sail-operator/pkg/reconciler"
	"github.com/istio-ecosystem/sail-operator/pkg/scheme"
	"github.com/istio-ecosystem/sail-operator/pkg/test/project"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileSkipsWhenAnnotationDisabled(t *testing.T) {
	ctx := context.Background()
	istio := &v1.Istio{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec:       v1.IstioSpec{Version: "v1.30.3", Namespace: "istio-system"},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(istio).WithStatusSubresource(istio).Build()
	r := &Reconciler{
		Config: config.ReconcilerConfig{
			PersesDashboardFS: os.DirFS(path.Join(project.RootDir, "resources", "perses")),
		},
		Client: cl,
		Scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(ctx, istio)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
}

func TestReconcileValidationErrorMissingProject(t *testing.T) {
	ctx := context.Background()
	istio := &v1.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Annotations: map[string]string{
				constants.PersesDashboardsAnnotationKey: constants.PersesDashboardsEnabledValue,
			},
		},
		Spec: v1.IstioSpec{Version: "v1.30.3", Namespace: "istio-system"},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(istio).WithStatusSubresource(istio).Build()
	r := &Reconciler{
		Config: config.ReconcilerConfig{
			PersesDashboardFS: os.DirFS(path.Join(project.RootDir, "resources", "perses")),
		},
		Client: cl,
		Scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(ctx, istio)
	if !reconciler.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	got := &v1.Istio{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(istio), got); err != nil {
		t.Fatalf("get istio: %v", err)
	}
	cond := got.Status.GetCondition(v1.IstioConditionPersesDashboardsAvailable)
	if cond.Reason != v1.IstioReasonPersesDashboardsInvalidConfig {
		t.Fatalf("expected InvalidConfig, got %q", cond.Reason)
	}
}

func TestReconcileMissingCRDs(t *testing.T) {
	ctx := context.Background()
	istio := testAnnotatedIstio()
	cl := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(istio).WithStatusSubresource(istio).Build()
	r := &Reconciler{
		Config: config.ReconcilerConfig{
			PersesDashboardFS: os.DirFS(path.Join(project.RootDir, "resources", "perses")),
		},
		Client: cl,
		Scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(ctx, istio)
	if !reconciler.IsTransientError(err) {
		t.Fatalf("expected transient error, got %v", err)
	}

	got := &v1.Istio{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(istio), got); err != nil {
		t.Fatalf("get istio: %v", err)
	}
	cond := got.Status.GetCondition(v1.IstioConditionPersesDashboardsAvailable)
	if cond.Reason != v1.IstioReasonPersesDashboardsMissingCRDs {
		t.Fatalf("expected MissingCRDs, got %q", cond.Reason)
	}
}

func TestReconcileCreatesDashboards(t *testing.T) {
	ctx := context.Background()
	istio := testAnnotatedIstio()
	crd := testPersesDashboardCRD()
	cl := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(istio, crd).WithStatusSubresource(istio).Build()
	r := &Reconciler{
		Config: config.ReconcilerConfig{
			PersesDashboardFS: os.DirFS(path.Join(project.RootDir, "resources", "perses")),
		},
		Client: cl,
		Scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(ctx, istio)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	got := &v1.Istio{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(istio), got); err != nil {
		t.Fatalf("get istio: %v", err)
	}
	cond := got.Status.GetCondition(v1.IstioConditionPersesDashboardsAvailable)
	if cond.Status != metav1.ConditionTrue {
		t.Fatalf("expected PersesDashboardsAvailable=True, got %s (%s)", cond.Status, cond.Reason)
	}
}

func testAnnotatedIstio() *v1.Istio {
	return &v1.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Annotations: map[string]string{
				constants.PersesDashboardsAnnotationKey: constants.PersesDashboardsEnabledValue,
				constants.PersesProjectAnnotationKey:    "monitoring",
			},
		},
		Spec: v1.IstioSpec{Version: "v1.30.3", Namespace: "istio-system"},
	}
}

func testPersesDashboardCRD() *apiextensionsv1.CustomResourceDefinition {
	preserve := true
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: perses.PersesDashboardCRD},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "perses.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "PersesDashboard",
				ListKind: "PersesDashboardList",
				Plural:   "persesdashboards",
				Singular: "persesdashboard",
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
