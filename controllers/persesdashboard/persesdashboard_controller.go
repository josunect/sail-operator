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
	"errors"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/istio-ecosystem/sail-operator/api/v1"
	"github.com/istio-ecosystem/sail-operator/pkg/config"
	"github.com/istio-ecosystem/sail-operator/pkg/enqueuelogger"
	"github.com/istio-ecosystem/sail-operator/pkg/perses"
	"github.com/istio-ecosystem/sail-operator/pkg/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler provisions PersesDashboard resources when enabled via Istio annotations.
type Reconciler struct {
	Config config.ReconcilerConfig
	client.Client
	Scheme *runtime.Scheme
}

func NewReconciler(cfg config.ReconcilerConfig, cl client.Client, scheme *runtime.Scheme) *Reconciler {
	return &Reconciler{
		Config: cfg,
		Client: cl,
		Scheme: scheme,
	}
}

// +kubebuilder:rbac:groups=sailoperator.io,resources=istios,verbs=get;list;watch
// +kubebuilder:rbac:groups=sailoperator.io,resources=istios/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=perses.dev,resources=persesdashboards,verbs=get;list;create
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// Reconcile provisions Perses dashboards for Istio resources with the perses-dashboards annotation.
func (r *Reconciler) Reconcile(ctx context.Context, istio *v1.Istio) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling Perses dashboards")

	result, reconcileErr := r.doReconcile(ctx, istio)
	statusErr := r.updateStatus(ctx, istio, reconcileErr)
	return result, errors.Join(reconcileErr, statusErr)
}

func (r *Reconciler) doReconcile(ctx context.Context, istio *v1.Istio) (ctrl.Result, error) {
	if !perses.DashboardsEnabled(istio) {
		return ctrl.Result{}, nil
	}

	project, err := perses.ProjectNamespace(istio)
	if err != nil {
		return ctrl.Result{}, reconciler.NewValidationError(err.Error())
	}

	result, err := perses.ReconcileDashboards(ctx, r.Client, r.Config.PersesDashboardFS, project)
	if err != nil {
		if len(result.FailedNames) > 0 {
			return ctrl.Result{}, reconciler.NewTransientError(err.Error())
		}
		return ctrl.Result{}, err
	}
	if !result.CRDsAvailable {
		return ctrl.Result{}, reconciler.NewTransientError("PersesDashboard CRD not found")
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) updateStatus(ctx context.Context, istio *v1.Istio, reconcileErr error) error {
	if !perses.DashboardsEnabled(istio) {
		return nil
	}

	status := *istio.Status.DeepCopy()
	status.SetCondition(r.persesDashboardsCondition(ctx, istio, reconcileErr))
	return reconciler.UpdateStatus(ctx, r.Client, istio, istio.Status, status, nil)
}

func (r *Reconciler) persesDashboardsCondition(ctx context.Context, istio *v1.Istio, reconcileErr error) v1.StatusCondition {
	c := v1.StatusCondition{Type: v1.IstioConditionPersesDashboardsAvailable}

	if _, err := perses.ProjectNamespace(istio); err != nil {
		c.Status = metav1.ConditionFalse
		c.Reason = v1.IstioReasonPersesDashboardsInvalidConfig
		c.Message = err.Error()
		return c
	}

	available, err := perses.DashboardCRDAvailable(ctx, r.Client)
	if err != nil {
		c.Status = metav1.ConditionFalse
		c.Reason = v1.IstioReasonPersesDashboardsReconcileError
		c.Message = err.Error()
		return c
	}
	if !available {
		c.Status = metav1.ConditionFalse
		c.Reason = v1.IstioReasonPersesDashboardsMissingCRDs
		c.Message = "PersesDashboard CRD not found"
		return c
	}

	if reconcileErr != nil {
		c.Status = metav1.ConditionFalse
		if reconciler.IsValidationError(reconcileErr) {
			c.Reason = v1.IstioReasonPersesDashboardsInvalidConfig
		} else if strings.Contains(reconcileErr.Error(), "create dashboard") {
			c.Reason = v1.IstioReasonPersesDashboardsPartiallyApplied
		} else {
			c.Reason = v1.IstioReasonPersesDashboardsReconcileError
		}
		c.Message = reconcileErr.Error()
		return c
	}

	project, err := perses.ProjectNamespace(istio)
	if err != nil {
		c.Status = metav1.ConditionFalse
		c.Reason = v1.IstioReasonPersesDashboardsInvalidConfig
		c.Message = err.Error()
		return c
	}

	allExist, err := perses.DashboardsExist(ctx, r.Client, project)
	if err != nil {
		c.Status = metav1.ConditionFalse
		c.Reason = v1.IstioReasonPersesDashboardsReconcileError
		c.Message = err.Error()
		return c
	}
	if !allExist {
		c.Status = metav1.ConditionFalse
		c.Reason = v1.IstioReasonPersesDashboardsPartiallyApplied
		c.Message = "not all Perses dashboards exist in the target namespace"
		return c
	}

	c.Status = metav1.ConditionTrue
	c.Reason = v1.IstioReasonPersesDashboardsAvailable
	return c
}

// SetupWithManager registers the controller with the manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("ctrlr").WithName("persesdashboard")
	mainObjectHandler := wrapEventHandler(logger, &handler.EnqueueRequestForObject{})

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			LogConstructor: func(req *reconcile.Request) logr.Logger {
				log := logger
				if req != nil {
					log = log.WithValues("Istio", req.Name)
				}
				return log
			},
			MaxConcurrentReconciles: r.Config.MaxConcurrentReconciles,
		}).
		Watches(&v1.Istio{}, mainObjectHandler).
		Named("persesdashboard").
		Complete(reconciler.NewStandardReconciler(r.Client, r.Reconcile))
}

func wrapEventHandler(logger logr.Logger, h handler.EventHandler) handler.EventHandler {
	return enqueuelogger.WrapIfNecessary(v1.IstioKind, logger, h)
}
