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

	"github.com/MirrorStudios/fallernetes-operator/operator/internal/builders"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
)

const TypeFinalizer = "gametype.falloria.com/finalizer"

// GameTypeReconciler reconciles a GameType object
type GameTypeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *GameTypeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrllog.FromContext(ctx).WithValues("gametype", req.Name, "namespace", req.Namespace)

	logger.Info("Reconciling GameType")
	gametype := &gameserverv1alpha1.GameType{}
	if err := r.Get(ctx, req.NamespacedName, gametype); err != nil {
		logger.Error(err, "Failed to get gametype resource")
		return ctrl.Result{}, err
	}

	// Handle finalizer addition
	if gametype.DeletionTimestamp == nil && !controllerutil.ContainsFinalizer(gametype, TypeFinalizer) {
		logger.Info("Adding finalizer to gametype")
		controllerutil.AddFinalizer(gametype, TypeFinalizer)
		if err := r.Update(ctx, gametype); err != nil {
			r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeFinalizerAdded, "failed to add finalizers: %s", err)
			logger.Error(err, "Failed to add finalizer to gametype")
			return ctrl.Result{}, err
		}
		r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeFinalizerAdded, "Added finalizers to game")
		return ctrl.Result{}, nil
	}

	// Handle resource deletion
	if gametype.DeletionTimestamp != nil || !gametype.GetDeletionTimestamp().IsZero() {
		logger.Info("Handling deletion of gametype")
		if err := r.handleDeletion(ctx, gametype, logger); err != nil {
			r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeFinalizerRemoved, "failed to remove finalizers: %s", err)
			logger.Error(err, "Failed to handle gametype deletion")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.syncGameTypeStatus(ctx, gametype, logger); err != nil {
		return ctrl.Result{}, err
	}

	return r.handleUpdating(ctx, gametype, logger)
}

// syncGameTypeStatus updates ActiveFleetName, TotalFleets, Replicas, ReadyReplicas,
// ObservedGeneration, and conditions based on the current set of owned fleets.
func (r *GameTypeReconciler) syncGameTypeStatus(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) error {
	fleets, err := utils.GetFleetsForType(ctx, r.Client, gametype, logger)
	if err != nil {
		return err
	}

	gametype.Status.TotalFleets = int32(len(fleets.Items))
	gametype.Status.ObservedGeneration = gametype.Generation

	newestFleet := utils.GetNewestFleet(fleets.Items)
	if newestFleet != nil {
		gametype.Status.ActiveFleetName = newestFleet.Name
		gametype.Status.Replicas = newestFleet.Status.Replicas
		gametype.Status.ReadyReplicas = newestFleet.Status.ReadyReplicas
	} else {
		gametype.Status.ActiveFleetName = ""
		gametype.Status.Replicas = 0
		gametype.Status.ReadyReplicas = 0
	}

	var oldReadyStatus metav1.ConditionStatus
	if c := meta.FindStatusCondition(gametype.Status.Conditions, gameserverv1alpha1.ConditionReady); c != nil {
		oldReadyStatus = c.Status
	}
	var oldRollingUpdateStatus metav1.ConditionStatus
	if c := meta.FindStatusCondition(gametype.Status.Conditions, gameserverv1alpha1.ConditionRollingUpdate); c != nil {
		oldRollingUpdateStatus = c.Status
	}

	r.syncGameTypeConditions(gametype)

	if newReady := meta.FindStatusCondition(gametype.Status.Conditions, gameserverv1alpha1.ConditionReady); newReady != nil {
		if oldReadyStatus != metav1.ConditionTrue && newReady.Status == metav1.ConditionTrue {
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeReady, "GameType is ready")
		}
	}
	if newRU := meta.FindStatusCondition(gametype.Status.Conditions, gameserverv1alpha1.ConditionRollingUpdate); newRU != nil {
		if oldRollingUpdateStatus == metav1.ConditionTrue && newRU.Status == metav1.ConditionFalse {
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeRollingUpdateComplete, "Rolling update complete")
		}
	}


	return r.Status().Update(ctx, gametype)
}

// syncGameTypeConditions sets the Ready and RollingUpdate conditions.
func (r *GameTypeReconciler) syncGameTypeConditions(gametype *gameserverv1alpha1.GameType) {
	gen := gametype.Generation
	desired := gametype.Spec.FleetSpec.Scaling.Replicas

	if gametype.Status.ActiveFleetName != "" &&
		gametype.Status.ReadyReplicas == desired && desired > 0 {
		meta.SetStatusCondition(&gametype.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             gameserverv1alpha1.ReasonAtDesiredCount,
			Message:            fmt.Sprintf("Active fleet %s is ready at %d replicas", gametype.Status.ActiveFleetName, gametype.Status.ReadyReplicas),
			ObservedGeneration: gen,
		})
	} else {
		msg := "Fleet not ready or no active fleet"
		if gametype.Status.ActiveFleetName != "" {
			msg = fmt.Sprintf("Fleet %s has %d/%d ready replicas", gametype.Status.ActiveFleetName, gametype.Status.ReadyReplicas, desired)
		}
		meta.SetStatusCondition(&gametype.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonReplicasMismatch,
			Message:            msg,
			ObservedGeneration: gen,
		})
	}

	if gametype.Status.TotalFleets > 1 {
		meta.SetStatusCondition(&gametype.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionRollingUpdate,
			Status:             metav1.ConditionTrue,
			Reason:             gameserverv1alpha1.ReasonRolloutInProgress,
			Message:            fmt.Sprintf("Rolling update in progress: %d fleets exist", gametype.Status.TotalFleets),
			ObservedGeneration: gen,
		})
	} else {
		meta.SetStatusCondition(&gametype.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionRollingUpdate,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonNoRollout,
			Message:            "No rolling update in progress",
			ObservedGeneration: gen,
		})
	}
}

// handleUpdating handles fleet creation, replica scaling, and rolling updates.
func (r *GameTypeReconciler) handleUpdating(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) (ctrl.Result, error) {
	fleets, err := utils.GetFleetsForType(ctx, r.Client, gametype, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(fleets.Items) == 0 {
		if _, err := r.handleCreation(ctx, gametype, logger); err != nil {
			return ctrl.Result{}, err
		}
		r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeFleetCreated, "Created initial fleet")
		return ctrl.Result{}, nil
	}
	if len(fleets.Items) == 1 {
		fleet := fleets.Items[0]
		if !gameserverv1alpha1.AreFleetsPodsEqual(&fleet.Spec, &gametype.Spec.FleetSpec) {
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeRollingUpdateStarted, "Creating new fleet for rolling update")
			return r.handleCreation(ctx, gametype, logger)
		} else if gametype.Spec.FleetSpec.Scaling.Replicas != fleet.Spec.Scaling.Replicas {
			fleet.Spec.Scaling.Replicas = gametype.Spec.FleetSpec.Scaling.Replicas
			if err := r.Update(ctx, &fleet); err != nil {
				return ctrl.Result{}, err
			}
			r.emitEventf(gametype, corev1.EventTypeNormal, utils.ReasonGametypeReplicasUpdated, "Scaling gametype to %d", fleet.Spec.Scaling.Replicas)
		}
	}
	if len(fleets.Items) > 1 {
		for i := range fleets.Items {
			fleet := &fleets.Items[i]
			if !gameserverv1alpha1.AreFleetsPodsEqual(&fleet.Spec, &gametype.Spec.FleetSpec) &&
				fleet.GetDeletionTimestamp() == nil {
				if err := r.Delete(ctx, fleet); err != nil {
					return ctrl.Result{}, err
				}
				r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeRollingUpdateComplete, "Old fleet pruned, rolling update complete")
				break
			}
		}
	}
	return ctrl.Result{}, nil
}

// handleDeletion triggers deletion of all fleets owned by the GameType.
func (r *GameTypeReconciler) handleDeletion(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) error {
	logger.Info("Handling gametype deletion")
	if controllerutil.ContainsFinalizer(gametype, TypeFinalizer) {
		logger.Info("Finalizer present, deleting associated fleets")
		fleets, err := utils.GetFleetsForType(ctx, r.Client, gametype, logger)
		if err != nil {
			return err
		}
		for _, fleet := range fleets.Items {
			r.emitEventf(gametype, corev1.EventTypeNormal, utils.ReasonGameTypeDeleting, "Deleting fleet %s", fleet.Name)
			if err := r.Delete(ctx, &fleet); err != nil {
				r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeFleetDeletionFailed, "Failed to delete fleet %s", fleet.Name)
				return err
			}
		}
		fleets, err = utils.GetFleetsForType(ctx, r.Client, gametype, logger)
		if err != nil {
			return err
		}
		if len(fleets.Items) == 0 {
			controllerutil.RemoveFinalizer(gametype, TypeFinalizer)
			if err := r.Update(ctx, gametype); err != nil {
				return err
			}
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeFinalizerRemoved, "Finalizer removed, all fleets deleted")
		}
	}
	return nil
}

// handleCreation creates the underlying fleet for the GameType.
func (r *GameTypeReconciler) handleCreation(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) (ctrl.Result, error) {
	fleet := builders.GetFleetObjectForType(gametype)
	if err := r.Create(ctx, fleet); err != nil {
		r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeFleetCreationFailed, "Failed to create new fleet: %s", err)
		logger.Error(err, "failed to create a new fleet for gametype")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *GameTypeReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

func (r *GameTypeReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameTypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.GameType{}).
		Named("gametype").
		Owns(&gameserverv1alpha1.Fleet{}).
		Complete(r)
}
