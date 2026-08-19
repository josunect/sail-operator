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
	"fmt"

	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	datasourceGVK = schema.GroupVersionKind{Group: persesGroup, Version: persesVersion, Kind: "PersesDatasource"}
	dashboardGVK  = schema.GroupVersionKind{Group: persesGroup, Version: persesVersion, Kind: "PersesDashboard"}
)

// ReconcileInput drives Perses provisioning for a MetricsIntegration.
type ReconcileInput struct {
	Integration *v1alpha1.MetricsIntegration
	Namespace   string
}

// ReconcileResult captures Perses reconciliation outcome for status updates.
type ReconcileResult struct {
	CRDsAvailable bool
}

// Reconciler provisions PersesDatasource and PersesDashboard resources.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile creates or updates Perses resources in the target project namespace.
func (r *Reconciler) Reconcile(ctx context.Context, in ReconcileInput) (ReconcileResult, error) {
	result := ReconcileResult{}

	available, err := CRDsAvailable(ctx, r.Client)
	if err != nil {
		return result, err
	}
	result.CRDsAvailable = available
	if !available {
		return result, nil
	}

	mi := in.Integration
	datasourceName := mi.Spec.DatasourceNameOrDefault()

	if err := r.reconcileDatasource(ctx, mi, in.Namespace, datasourceName); err != nil {
		return result, err
	}

	dashboards := ResolveDashboards(mi.Spec.SelectedDashboards())
	for _, def := range dashboards {
		if err := r.reconcileDashboard(ctx, mi, in.Namespace, datasourceName, def); err != nil {
			return result, err
		}
	}

	if err := r.pruneDashboards(ctx, mi, in.Namespace, dashboards); err != nil {
		return result, err
	}

	return result, nil
}

func (r *Reconciler) reconcileDatasource(ctx context.Context, mi *v1alpha1.MetricsIntegration, namespace, name string) error {
	desired, err := NewDatasource(mi, namespace, name)
	if err != nil {
		return err
	}
	return r.applyOwnedResource(ctx, mi, desired)
}

func (r *Reconciler) reconcileDashboard(ctx context.Context, mi *v1alpha1.MetricsIntegration, namespace, datasourceName string, def DashboardDefinition) error {
	raw, err := LoadDashboardYAML(def)
	if err != nil {
		return err
	}
	raw, err = RewriteDatasourceName(raw, datasourceName)
	if err != nil {
		return err
	}
	desired, err := ParseDashboard(raw)
	if err != nil {
		return err
	}
	desired.SetName(def.Name)
	desired.SetNamespace(namespace)
	labels := desired.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[PartOfLabelKey] = PartOfLabelValue
	labels[ManagedByLabel] = ManagedByValue
	desired.SetLabels(labels)
	return r.applyOwnedResource(ctx, mi, desired)
}

func (r *Reconciler) applyOwnedResource(ctx context.Context, owner *v1alpha1.MetricsIntegration, desired *unstructured.Unstructured) error {
	if err := controllerutil.SetControllerReference(owner, desired, r.Scheme); err != nil {
		return fmt.Errorf("set owner reference on %s/%s: %w", desired.GetKind(), desired.GetName(), err)
	}

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(desired.GroupVersionKind())
	key := client.ObjectKeyFromObject(desired)
	if err := r.Get(ctx, key, existing); err != nil {
		if apierrors.IsNotFound(err) {
			return r.Create(ctx, desired)
		}
		return err
	}

	desired.SetResourceVersion(existing.GetResourceVersion())
	return r.Update(ctx, desired)
}

func (r *Reconciler) pruneDashboards(ctx context.Context, mi *v1alpha1.MetricsIntegration, namespace string, keep []DashboardDefinition) error {
	keepNames := make(map[string]struct{}, len(keep))
	for _, def := range keep {
		keepNames[def.Name] = struct{}{}
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Group: persesGroup, Version: persesVersion, Kind: "PersesDashboardList"})
	if err := r.List(ctx, list, client.InNamespace(namespace), client.MatchingLabels{
		PartOfLabelKey: PartOfLabelValue,
		ManagedByLabel: ManagedByValue,
	}); err != nil {
		return err
	}

	for _, item := range list.Items {
		if _, ok := keepNames[item.GetName()]; ok {
			continue
		}
		for _, ownerRef := range item.GetOwnerReferences() {
			if ownerRef.UID == mi.UID {
				if err := r.Delete(ctx, &item); err != nil && !apierrors.IsNotFound(err) {
					return err
				}
				break
			}
		}
	}
	return nil
}

// Finalize removes owned Perses resources when the integration is deleted.
// Owner references normally handle garbage collection; this is a safety net for
// resources created before owner refs were set.
func (r *Reconciler) Finalize(ctx context.Context, mi *v1alpha1.MetricsIntegration, namespace string) error {
	for _, gvk := range []schema.GroupVersionKind{datasourceGVK, dashboardGVK} {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
		if err := r.List(ctx, list, client.InNamespace(namespace), client.MatchingLabels{
			PartOfLabelKey: PartOfLabelValue,
			ManagedByLabel: ManagedByValue,
		}); err != nil {
			return err
		}
		for _, item := range list.Items {
			for _, ownerRef := range item.GetOwnerReferences() {
				if ownerRef.UID == mi.UID {
					if err := r.Delete(ctx, &item); err != nil && !apierrors.IsNotFound(err) {
						return err
					}
					break
				}
			}
		}
	}
	return nil
}

// PersesProjectNamespace returns the namespace for Perses CRs from the target ref.
func PersesProjectNamespace(ref v1alpha1.TargetReference) (string, error) {
	if ref.Namespace == "" {
		return "", fmt.Errorf("Perses targetRef requires namespace (Perses project)")
	}
	return ref.Namespace, nil
}

// OwnerLabelSelector returns labels used on operator-managed Perses resources.
func OwnerLabelSelector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			PartOfLabelKey: PartOfLabelValue,
			ManagedByLabel: ManagedByValue,
		},
	}
}
