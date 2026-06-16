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
	"time"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/autoscaler"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"
)

// GameTypeAutoscalerReconciler reconciles a GameTypeAutoscaler object
type GameTypeAutoscalerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Webhook  autoscaler.Webhook
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypeautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *GameTypeAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("autoscaler", req.Name, "namespace", req.Namespace)

	autoscalerObj := &gameserverv1alpha1.GameTypeAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, autoscalerObj); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get autoscaler resource")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	gametype := &gameserverv1alpha1.GameType{}
	if err := r.Get(ctx, types.NamespacedName{Name: autoscalerObj.Spec.GameTypeName, Namespace: autoscalerObj.Namespace}, gametype); err != nil {
		r.emitEventf(autoscalerObj, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerInvalidTarget, "Failed to find GameType %q", autoscalerObj.Spec.GameTypeName)
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if autoscalerObj.Spec.AutoscalePolicy.Type != gameserverv1alpha1.Webhook {
		r.emitEventf(autoscalerObj, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerInvalidAutoscalePolicy,
			"%s is not a valid policy type", autoscalerObj.Spec.AutoscalePolicy.Type)
		return ctrl.Result{}, fmt.Errorf("%s is not a valid policy type", autoscalerObj.Spec.AutoscalePolicy.Type)
	}

	if autoscalerObj.Spec.Sync.Type != gameserverv1alpha1.FixedInterval {
		r.emitEventf(autoscalerObj, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerInvalidSyncType,
			"%s is not a valid sync type", autoscalerObj.Spec.Sync.Type)
		return ctrl.Result{}, fmt.Errorf("%s is not a valid sync type, currently only fixed interval is supported", autoscalerObj.Spec.Sync.Type)
	}

	result, err := r.Webhook.SendScaleWebhookRequest(autoscalerObj, gametype)
	if err != nil {
		r.emitEventf(autoscalerObj, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerWebhook, "failed to send the webhook request: %v", err)
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("failed to send scale webhook request: %w", err)
	}

	if !result.Scale {
		return ctrl.Result{RequeueAfter: autoscalerObj.Spec.Sync.Time.Duration}, nil
	}

	desired := int32(result.DesiredReplicas)
	scaling := gametype.Spec.FleetSpec.Scaling
	if scaling.MinReplicas != nil && desired < *scaling.MinReplicas {
		desired = *scaling.MinReplicas
	}
	if scaling.MaxReplicas != nil && desired > *scaling.MaxReplicas {
		desired = *scaling.MaxReplicas
	}
	gametype.Spec.FleetSpec.Scaling.Replicas = desired
	if err := r.Client.Update(ctx, gametype); err != nil {
		r.emitEvent(autoscalerObj, corev1.EventTypeWarning, utils.ReasonGameTypeAutoscalerScale, "failed to update the gametype")
		return ctrl.Result{}, fmt.Errorf("failed to update gametype with new replica count: %w", err)
	}
	r.emitEventf(autoscalerObj, corev1.EventTypeNormal, utils.ReasonGameTypeAutoscalerScale, "Scaling gametype to %d", result.DesiredReplicas)

	return ctrl.Result{RequeueAfter: autoscalerObj.Spec.Sync.Time.Duration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameTypeAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.GameTypeAutoscaler{}).
		Named("gametypeautoscaler").
		Complete(r)
}

func (r *GameTypeAutoscalerReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

func (r *GameTypeAutoscalerReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}
