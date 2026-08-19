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
	"errors"

	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
	"github.com/istio-ecosystem/sail-operator/api/v1alpha1"
	"github.com/istio-ecosystem/sail-operator/pkg/config"
	"github.com/istio-ecosystem/sail-operator/pkg/constants"
	"github.com/istio-ecosystem/sail-operator/pkg/perses"
	"github.com/istio-ecosystem/sail-operator/pkg/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconciler reconciles MetricsIntegration resources.
type Reconciler struct {
	client.Client
	Config config.ReconcilerConfig
	Scheme *runtime.Scheme
}

func NewReconciler(cfg config.ReconcilerConfig, cl client.Client, scheme *runtime.Scheme) *Reconciler {
	return &Reconciler{
		Client: cl,
		Config: cfg,
		Scheme: scheme,
	}
}

// +kubebuilder:rbac:groups=sailoperator.io,resources=metricsintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sailoperator.io,resources=metricsintegrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sailoperator.io,resources=metricsintegrations/finalizers,verbs=update
// +kubebuilder:rbac:groups=perses.dev,resources=persesdatasources;persesdashboards,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, mi *v1alpha1.MetricsIntegration) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	reconcileErr := r.doReconcile(ctx, mi)
	if err := r.updateStatus(ctx, mi, reconcileErr); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, errors.Join(reconcileErr, err)
	}
	return ctrl.Result{}, reconcileErr
}

func (r *Reconciler) doReconcile(ctx context.Context, mi *v1alpha1.MetricsIntegration) error {
	ref, ok := mi.Spec.PersesTarget()
	if !ok {
		return nil
	}

	namespace, err := perses.PersesProjectNamespace(ref)
	if err != nil {
		return reconciler.NewValidationError(err.Error())
	}

	result, err := (&perses.Reconciler{Client: r.Client, Scheme: r.Scheme}).Reconcile(ctx, perses.ReconcileInput{
		Integration: mi,
		Namespace:   namespace,
	})
	if err != nil {
		return err
	}
	if !result.CRDsAvailable {
		return nil
	}
	return nil
}

func (r *Reconciler) Finalize(ctx context.Context, mi *v1alpha1.MetricsIntegration) error {
	ref, ok := mi.Spec.PersesTarget()
	if !ok {
		return nil
	}
	namespace, err := perses.PersesProjectNamespace(ref)
	if err != nil {
		return err
	}
	return (&perses.Reconciler{Client: r.Client, Scheme: r.Scheme}).Finalize(ctx, mi, namespace)
}

func (r *Reconciler) updateStatus(ctx context.Context, mi *v1alpha1.MetricsIntegration, reconcileErr error) error {
	status := *mi.Status.DeepCopy()
	status.ObservedGeneration = mi.Generation
	status.SetCondition(r.reconciledCondition(reconcileErr))

	if ref, ok := mi.Spec.PersesTarget(); ok {
		status.SetCondition(r.persesAvailableCondition(ctx, ref, reconcileErr))
	}

	return reconciler.UpdateStatus(ctx, r.Client, mi, mi.Status, status, nil)
}

func (r *Reconciler) reconciledCondition(reconcileErr error) v1.StatusCondition {
	c := v1.StatusCondition{Type: v1.ConditionType(v1alpha1.MetricsIntegrationConditionReconciled)}
	if reconcileErr == nil {
		c.Status = metav1.ConditionTrue
		c.Reason = v1alpha1.MetricsIntegrationReasonHealthy
		return c
	}
	c.Status = metav1.ConditionFalse
	c.Reason = v1alpha1.MetricsIntegrationReasonReconcileError
	c.Message = reconcileErr.Error()
	return c
}

func (r *Reconciler) persesAvailableCondition(ctx context.Context, ref v1alpha1.TargetReference, reconcileErr error) v1.StatusCondition {
	c := v1.StatusCondition{Type: v1.ConditionType(v1alpha1.MetricsIntegrationConditionPersesAvailable)}
	available, err := perses.CRDsAvailable(ctx, r.Client)
	if err != nil {
		c.Status = metav1.ConditionFalse
		c.Reason = v1alpha1.MetricsIntegrationReasonReconcileError
		c.Message = err.Error()
		return c
	}
	if !available {
		c.Status = metav1.ConditionFalse
		c.Reason = v1alpha1.MetricsIntegrationReasonMissingCRDs
		c.Message = "perses.dev CRDs are not installed"
		return c
	}
	if reconcileErr != nil {
		c.Status = metav1.ConditionFalse
		c.Reason = v1alpha1.MetricsIntegrationReasonReconcileError
		c.Message = reconcileErr.Error()
		return c
	}
	if ref.Namespace == "" {
		c.Status = metav1.ConditionFalse
		c.Reason = v1alpha1.MetricsIntegrationReasonReconcileError
		c.Message = "Perses targetRef requires namespace"
		return c
	}
	c.Status = metav1.ConditionTrue
	c.Reason = v1alpha1.MetricsIntegrationReasonHealthy
	return c
}

// SetupWithManager registers the controller with the manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MetricsIntegration{}).
		Named("metricsintegration").
		Complete(reconciler.NewStandardReconcilerWithFinalizer[*v1alpha1.MetricsIntegration](
			r.Client, r.Reconcile, r.Finalize, constants.FinalizerName))
}
