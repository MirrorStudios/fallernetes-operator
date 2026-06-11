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

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/builders"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/sidecar"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FLEET_FINALIZER = "fleets.falloria.com/finalizer"

// FleetReconciler reconciles a Fleet object
type FleetReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	DeletionChecker sidecar.FleetDeletionChecker
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=fleets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=fleets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=fleets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *FleetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fleet := &gameserverv1alpha1.Fleet{}
	if err := r.Get(ctx, req.NamespacedName, fleet); err != nil {
		return ctrl.Result{}, err
	}

	// Handle resource deletion
	if fleet.DeletionTimestamp != nil || !fleet.GetDeletionTimestamp().IsZero() {
		if err := r.handleDeletion(ctx, fleet); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to handle fleet deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Handle finalizer addition
	if fleet.DeletionTimestamp == nil && !controllerutil.ContainsFinalizer(fleet, FLEET_FINALIZER) {
		controllerutil.AddFinalizer(fleet, FLEET_FINALIZER)
		if err := r.Update(ctx, fleet); err != nil {
			r.emitEventf(fleet, corev1.EventTypeWarning, utils.ReasonFleetUpdateFailed, "Fleet finalizer update failed: %s", err)
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer to fleet: %w", err)
		}
		r.emitEvent(fleet, corev1.EventTypeNormal, utils.ReasonFleetInitialized, "Fleet finalizers added")
		return ctrl.Result{}, nil
	}

	servers, err := utils.GetServersForFleet(ctx, r.Client, fleet)
	if err != nil {
		return ctrl.Result{}, err
	}

	fleet.Status.Replicas = int32(len(servers.Items))
	fleet.Status.DesiredReplicas = fleet.Spec.Scaling.Replicas
	fleet.Status.ObservedGeneration = fleet.Generation
	fleet.Status.ReadyReplicas = countReadyServers(servers.Items)

	if fleet.Spec.Scaling.Replicas != fleet.Status.Replicas {
		if err := r.scaleServerCount(ctx, fleet, req.Namespace); err != nil {
			return ctrl.Result{}, err
		}
		servers, err = utils.GetServersForFleet(ctx, r.Client, fleet)
		if err != nil {
			return ctrl.Result{}, err
		}
		fleet.Status.Replicas = int32(len(servers.Items))
		fleet.Status.ReadyReplicas = countReadyServers(servers.Items)
	}

	var oldReadyStatus metav1.ConditionStatus
	if c := meta.FindStatusCondition(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady); c != nil {
		oldReadyStatus = c.Status
	}

	r.syncFleetConditions(fleet)

	if newReady := meta.FindStatusCondition(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady); newReady != nil && oldReadyStatus != newReady.Status {
		if newReady.Status == metav1.ConditionTrue {
			r.emitEventf(fleet, corev1.EventTypeNormal, utils.ReasonFleetReady, "Fleet ready: all %d replicas are healthy", fleet.Status.DesiredReplicas)
		} else {
			r.emitEventf(fleet, corev1.EventTypeWarning, utils.ReasonFleetNotReady, "Fleet not ready: %d/%d replicas ready", fleet.Status.ReadyReplicas, fleet.Status.DesiredReplicas)
		}
	}

	if err := r.Status().Update(ctx, fleet); err != nil {
		r.emitEventf(fleet, corev1.EventTypeWarning, utils.ReasonFleetUpdateFailed, "Failed to update fleet status: %s", err)
		return ctrl.Result{}, fmt.Errorf("failed to update Fleet status resource: %w", err)
	}
	return ctrl.Result{}, nil
}

// countReadyServers counts servers in the Ready phase.
func countReadyServers(servers []gameserverv1alpha1.Server) int32 {
	count := int32(0)
	for _, s := range servers {
		if s.Status.Phase == gameserverv1alpha1.ServerPhaseReady {
			count++
		}
	}
	return count
}

// syncFleetConditions sets Ready, Available, and Scaling conditions based on current status fields.
func (r *FleetReconciler) syncFleetConditions(fleet *gameserverv1alpha1.Fleet) {
	gen := fleet.Generation

	if fleet.Status.ReadyReplicas == fleet.Status.DesiredReplicas && fleet.Status.DesiredReplicas > 0 {
		meta.SetStatusCondition(&fleet.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             gameserverv1alpha1.ReasonAtDesiredCount,
			Message:            fmt.Sprintf("All %d replicas are ready", fleet.Status.DesiredReplicas),
			ObservedGeneration: gen,
		})
	} else {
		meta.SetStatusCondition(&fleet.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonReplicasMismatch,
			Message:            fmt.Sprintf("Ready replicas (%d) do not match desired (%d)", fleet.Status.ReadyReplicas, fleet.Status.DesiredReplicas),
			ObservedGeneration: gen,
		})
	}

	if fleet.Status.ReadyReplicas > 0 {
		meta.SetStatusCondition(&fleet.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionAvailable,
			Status:             metav1.ConditionTrue,
			Reason:             gameserverv1alpha1.ReasonReplicasAvailable,
			Message:            fmt.Sprintf("%d replica(s) available", fleet.Status.ReadyReplicas),
			ObservedGeneration: gen,
		})
	} else {
		meta.SetStatusCondition(&fleet.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionAvailable,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonNoReplicas,
			Message:            "No replicas are available",
			ObservedGeneration: gen,
		})
	}

	if fleet.Status.Replicas != fleet.Status.DesiredReplicas {
		reason := gameserverv1alpha1.ReasonScalingUp
		if fleet.Status.Replicas > fleet.Status.DesiredReplicas {
			reason = gameserverv1alpha1.ReasonScalingDown
		}
		meta.SetStatusCondition(&fleet.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionScaling,
			Status:             metav1.ConditionTrue,
			Reason:             reason,
			Message:            fmt.Sprintf("Scaling from %d to %d replicas", fleet.Status.Replicas, fleet.Status.DesiredReplicas),
			ObservedGeneration: gen,
		})
	} else {
		meta.SetStatusCondition(&fleet.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionScaling,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonAtDesiredCount,
			Message:            "Fleet is at desired replica count",
			ObservedGeneration: gen,
		})
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *FleetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.Fleet{}).
		Owns(&gameserverv1alpha1.Server{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Named("fleet").
		Complete(r)
}

func (r *FleetReconciler) scaleServerCount(ctx context.Context, fleet *gameserverv1alpha1.Fleet, namespace string) error {
	if fleet.Status.Replicas < fleet.Spec.Scaling.Replicas {
		serversNeeded := fleet.Spec.Scaling.Replicas - fleet.Status.Replicas
		for range serversNeeded {
			server := builders.CreateServerForFleet(*fleet, namespace)
			err := r.Create(ctx, server)
			if err != nil {
				r.emitEventf(fleet, corev1.EventTypeWarning, utils.ReasonFleetUpdateFailed, "Failed to create a server: %s", err)
				return err
			}
		}
		r.emitEventf(fleet, corev1.EventTypeNormal, utils.ReasonFleetScaledUp, "Scaled servers up to %d", fleet.Spec.Scaling.Replicas)
	}
	if fleet.Status.Replicas > fleet.Spec.Scaling.Replicas {
		servers, err := utils.GetServersForFleet(ctx, r.Client, fleet)
		if err != nil {
			return err
		}
		server, err := utils.FindDeleteServer(ctx, fleet, servers, r.Client, r.DeletionChecker)
		if err != nil {
			return err
		}
		if err := r.Client.Delete(ctx, server); err != nil {
			r.emitEventf(fleet, corev1.EventTypeWarning, utils.ReasonFleetUpdateFailed, "Failed to delete a server: %s", err)
			return err
		}
		r.emitEventf(fleet, corev1.EventTypeNormal, utils.ReasonFleetScaledDown, "Scaled servers down to %d", fleet.Spec.Scaling.Replicas)
	}
	return nil
}

func (r *FleetReconciler) handleDeletion(ctx context.Context, fleet *gameserverv1alpha1.Fleet) error {
	servers, err := utils.GetServersForFleet(ctx, r.Client, fleet)
	if err != nil {
		return err
	}
	for _, server := range servers.Items {
		if err := r.Delete(ctx, &server); err != nil {
			return err
		}
	}
	servers, err = utils.GetServersForFleet(ctx, r.Client, fleet)
	if err != nil {
		return err
	}
	if len(servers.Items) == 0 {
		controllerutil.RemoveFinalizer(fleet, FLEET_FINALIZER)
		if err := r.Update(ctx, fleet); err != nil {
			r.emitEventf(fleet, corev1.EventTypeWarning, utils.ReasonFleetUpdateFailed, "Failed to remove finalizer: %s", err)
			return err
		}
		r.emitEvent(fleet, corev1.EventTypeNormal, utils.ReasonFleetServersRemoved, "Fleet finalizers removed")
	}
	return nil
}

func (r *FleetReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

func (r *FleetReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}
