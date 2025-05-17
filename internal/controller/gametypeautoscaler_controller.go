/*
Copyright 2025.

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
	"github.com/MirrorStudios/fallernetes/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes/api/v1alpha1"
)

// GameTypeAutoscalerReconciler reconciles a GameTypeAutoscaler object
type GameTypeAutoscalerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Webhook  utils.Webhook
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GameTypeAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("autoscaler", req.Name, "namespace", req.Namespace)

	autoscaler := &gameserverv1alpha1.GameTypeAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, autoscaler); err != nil {
		logger.Error(err, "Failed to get autoscaler resource")
		return ctrl.Result{Requeue: true}, err
	}

	gametype := &gameserverv1alpha1.GameType{}
	namespacedGametype := types.NamespacedName{
		Name:      autoscaler.Spec.GameTypeName,
		Namespace: autoscaler.Namespace,
	}
	if err := r.Get(ctx, namespacedGametype, gametype); err != nil {
		r.emitEvent(autoscaler, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerInvalidServer, "Failed to find the gametype")
		return ctrl.Result{Requeue: true}, err
	}

	//Make sure the type is fine
	if autoscaler.Spec.AutoscalePolicy.Type != gameserverv1alpha1.Webhook {
		r.emitEvent(autoscaler, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerInvalidAutoscalePolicy,
			"invalid game autoscaler policy type")
		return ctrl.Result{}, fmt.Errorf("%s is not a valid policy type", autoscaler.Spec.AutoscalePolicy.Type)
	}

	//Send request to defined webhook
	result, err := r.Webhook.SendScaleWebhookRequest(autoscaler, gametype)
	if err != nil {
		r.emitEventf(autoscaler, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerWebhook, "failed to send the webhook request: %v", err)
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("failed to send scale webhook request: %w", err)
	}

	//Check that the sync type is fine
	if autoscaler.Spec.Sync.Type != gameserverv1alpha1.FixedInterval {
		r.emitEventf(autoscaler, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerInvalidSyncType, "%s is not a valid sync type", autoscaler.Spec.Sync.Type)
		return ctrl.Result{}, fmt.Errorf("%s is not a valid sync type, currently only fixed interval is supported", autoscaler.Spec.Sync.Type)
	}

	//If scaleing not requested, requeue
	if !result.Scale {
		return ctrl.Result{
			RequeueAfter: autoscaler.Spec.Sync.Time.Duration,
		}, nil
	}

	//Otherwise, scale to new replica count
	gametype.Spec.FleetSpec.Scaling.Replicas = int32(result.DesiredReplicas)
	if err := r.Client.Update(ctx, gametype); err != nil {
		r.emitEvent(autoscaler, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerScale, "failed to update the gametype")
		return ctrl.Result{}, fmt.Errorf("failed to update gametype with new replica count: %w", err)
	}
	r.emitEventf(autoscaler, corev1.EventTypeNormal, utils.ReasonGameTypeAutoscalerScale, "Scaling game to %d", result.DesiredReplicas)

	//Requeue after the defined time
	return ctrl.Result{
		RequeueAfter: autoscaler.Spec.Sync.Time.Duration,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameTypeAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.GameTypeAutoscaler{}).
		Complete(r)
}

// emitEvent is used by the GameTypeAutoscalerReconciler to easily add events to objects
func (r *GameTypeAutoscalerReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

// emitEventf is used by the GameTypeAutoscalerReconciler to easily add events to objects with arguments
func (r *GameTypeAutoscalerReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}
