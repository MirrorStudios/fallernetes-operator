/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// GameTypeAutoscalerReconciler reconciles a GameTypeAutoscaler object
type GameTypeAutoscalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers/finalizers,verbs=update

func (r *GameTypeAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	autoscaler := &gameserverv1alpha1.GameTypeAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, autoscaler); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get GameTypeAutoscaler: %w", err)
		}
		return ctrl.Result{}, nil
	}

	autoscaler.Status.ObservedGeneration = autoscaler.Generation

	// Initialise conditions to Unknown if they have not been set yet.
	if meta.FindStatusCondition(autoscaler.Status.Conditions, gameserverv1alpha1.ConditionReady) == nil {
		meta.SetStatusCondition(&autoscaler.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionUnknown,
			Reason:             gameserverv1alpha1.ReasonInitializing,
			Message:            "Autoscaler has not yet been evaluated",
			ObservedGeneration: autoscaler.Generation,
		})
	}
	if meta.FindStatusCondition(autoscaler.Status.Conditions, gameserverv1alpha1.ConditionScaling) == nil {
		meta.SetStatusCondition(&autoscaler.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionScaling,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonNotScaling,
			Message:            "No scale action has been taken",
			ObservedGeneration: autoscaler.Generation,
		})
	}

	if err := r.Status().Update(ctx, autoscaler); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update GameTypeAutoscaler status: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameTypeAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.GameTypeAutoscaler{}).
		Named("gametypeautoscaler").
		Complete(r)
}
