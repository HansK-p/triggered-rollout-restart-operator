/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	reloadrestarttriggerv1alpha1 "reload-restart-trigger/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	ctrlpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ResourceReloadRestartTriggerReconciler reconciles a ResourceReloadRestartTrigger object
type ResourceReloadRestartTriggerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=reload-restart-trigger.k8s.faith,resources=resourcereloadrestarttriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=reload-restart-trigger.k8s.faith,resources=resourcereloadrestarttriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=reload-restart-trigger.k8s.faith,resources=resourcereloadrestarttriggers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ResourceReloadRestartTrigger object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ResourceReloadRestartTriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx).WithValues("resourcereloadrestarttrigger", req.NamespacedName)

	reqLogger = reqLogger.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.V(2).Info("Reconciling ResourceReloadRestartTrigger")

	// Fetch the ResourceReloadRestartTrigger instance
	instance := &reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.V(3).Info("Request crd object with namespaced name in request not found. It must have been deleted. No actions needed.")
			return ctrl.Result{}, nil
		}
		reqLogger.V(0).Error(err, "Requeue request after error reading the CR object with namespaced name %s")
		return ctrl.Result{}, err
	}
	err = reconcileCrd(reqLogger, r, req, instance)
	if err != nil {
		reqLogger.Error(err, "Got an error during reconcile of crd")
	} else {
		reqLogger.V(2).Info("No issues found during reconcile")
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceReloadRestartTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	reqLogger := log.FromContext(context.TODO())
	return ctrl.NewControllerManagedBy(mgr).
		For(&reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}, builder.WithPredicates(ctrlpredicate.GenerationChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(eventHandler(reqLogger, r.Client)),
			builder.WithPredicates(eventFilter(reqLogger, r.Client)),
		).
		Watches(&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(eventHandler(reqLogger, r.Client)),
			builder.WithPredicates(eventFilter(reqLogger, r.Client)),
		).
		Complete(r)
}
