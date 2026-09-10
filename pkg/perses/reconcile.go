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
	"github.com/istio-ecosystem/sail-operator/pkg/reconciler"
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
	Targets     []v1alpha1.TargetReference
}

// ReconcileResult captures Perses reconciliation outcome for status updates.
type ReconcileResult struct {
	CRDsAvailable bool
}

// Reconciler provisions Perses dashboards and server-side applies datasource fields.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile server-side applies mesh fields onto user-created PersesDatasource resources
// and creates productized PersesDashboard resources.
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

	for _, target := range in.Targets {
		if err := r.reconcileTarget(ctx, in.Integration, target); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (r *Reconciler) reconcileTarget(ctx context.Context, mi *v1alpha1.MetricsIntegration, target v1alpha1.TargetReference) error {
	namespace, err := DatasourceNamespace(target)
	if err != nil {
		return reconciler.NewValidationError(err.Error())
	}

	if err := r.ensureDatasourceExists(ctx, namespace, target.Name); err != nil {
		return err
	}

	if err := r.reconcileDatasource(ctx, mi, namespace, target.Name); err != nil {
		return err
	}

	for _, def := range ProductDashboards {
		if err := r.reconcileDashboard(ctx, mi, namespace, target.Name, def); err != nil {
			return err
		}
	}

	return r.pruneDashboards(ctx, mi, namespace, ProductDashboards)
}

func (r *Reconciler) ensureDatasourceExists(ctx context.Context, namespace, name string) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(datasourceGVK)
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			return reconciler.NewValidationError(fmt.Sprintf("PersesDatasource %q not found in namespace %q", name, namespace))
		}
		return err
	}
	return nil
}

func (r *Reconciler) reconcileDatasource(ctx context.Context, mi *v1alpha1.MetricsIntegration, namespace, name string) error {
	desired, err := NewDatasource(mi, namespace, name)
	if err != nil {
		return err
	}
	return r.serverSideApply(ctx, desired)
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

	if err := controllerutil.SetControllerReference(mi, desired, r.Scheme); err != nil {
		return fmt.Errorf("set owner reference on %s/%s: %w", desired.GetKind(), desired.GetName(), err)
	}
	return r.serverSideApply(ctx, desired)
}

func (r *Reconciler) serverSideApply(ctx context.Context, desired *unstructured.Unstructured) error {
	//nolint:staticcheck // client.Apply via Patch is the supported SSA path in controller-runtime v0.24.
	return r.Patch(ctx, desired, client.Apply, client.FieldOwner(FieldOwner), client.ForceOwnership)
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

// Finalize removes owned PersesDashboard resources when the integration is deleted.
func (r *Reconciler) Finalize(ctx context.Context, mi *v1alpha1.MetricsIntegration, targets []v1alpha1.TargetReference) error {
	for _, target := range targets {
		namespace, err := DatasourceNamespace(target)
		if err != nil {
			continue
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

// DatasourceNamespace returns the namespace for a PersesDatasource targetRef.
func DatasourceNamespace(ref v1alpha1.TargetReference) (string, error) {
	if ref.Namespace == "" {
		return "", fmt.Errorf("PersesDatasource targetRef requires namespace")
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
