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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes/api/v1alpha1"
)

// GameTypeReconciler reconciles a GameType object
type GameTypeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const TypeFinalizer = "gametype.falloria.com/finalizer"

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=gametypes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GameTypeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("gametype", req.Name, "namespace", req.Namespace)

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
			r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeInitialized, "failed to add finalizers: %s", err)
			logger.Error(err, "Failed to add finalizer to gametype")
			return ctrl.Result{Requeue: true}, err
		}
		r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeInitialized, "Added finalizers to game")
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle resource deletion
	if gametype.DeletionTimestamp != nil || !gametype.GetDeletionTimestamp().IsZero() {
		logger.Info("Handling deletion of gametype")
		if err := r.handleDeletion(ctx, gametype, logger); err != nil {
			r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeInitialized, "failed to remove finalizers: %s", err)
			logger.Error(err, "Failed to handle gametype deletion")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	err := r.handleGametypeStatus(ctx, gametype, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updateReplicaCount(ctx, gametype)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	result, err, done := r.handleUpdating(ctx, gametype, logger)
	if done {
		return result, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// updateReplicaCount updates the replica count of the underlying fleet, based on the spec
func (r *GameTypeReconciler) updateReplicaCount(ctx context.Context, gametype *gameserverv1alpha1.GameType) error {
	if gametype.Status.CurrentFleetName == "" {
		return nil
	}

	fleet := &gameserverv1alpha1.Fleet{}
	name := types.NamespacedName{
		Namespace: gametype.Namespace,
		Name:      gametype.Status.CurrentFleetName,
	}
	err := r.Get(ctx, name, fleet)
	if err != nil {
		return fmt.Errorf("failed to get fleet to update: %s", err)
	}

	if fleet == nil {
		return fmt.Errorf("could not get fleet to update")
	}
	gametype.Status.CurrentFleetReplicas = fleet.Spec.Scaling.Replicas
	err = r.Status().Update(ctx, gametype)
	return err
}

// handleUpdating handles the updating process of the GameType
// Internally, this means creating a fleet, waiting for it to be done
// Then requesting the other fleet to be deleted
// And updating the latest fleets replica counts as needed
func (r *GameTypeReconciler) handleUpdating(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) (ctrl.Result, error, bool) {
	fleets, err := utils.GetFleetsForType(ctx, r.Client, gametype, logger)
	if err != nil {
		return ctrl.Result{}, err, true
	}
	if len(fleets.Items) == 0 {
		_, err := r.handleCreation(ctx, gametype, logger)
		if err != nil {
			return ctrl.Result{Requeue: true}, err, true
		}
		r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeInitialized, "Created initial fleet")
		return ctrl.Result{Requeue: true}, nil, true
	}
	if len(fleets.Items) == 1 {
		fleet := fleets.Items[0]
		gametype.Status.CurrentFleetName = fleet.Name
		if err := r.Status().Update(ctx, gametype); err != nil {
			return ctrl.Result{Requeue: true}, err, true
		}
		if !gameserverv1alpha1.AreFleetsPodsEqual(&fleet.Spec, &gametype.Spec.FleetSpec) {
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeSpecUpdated, "Creating new fleet")
			res, err := r.handleCreation(ctx, gametype, logger)
			return res, err, true
		} else if gametype.Spec.FleetSpec.Scaling.Replicas != gametype.Status.CurrentFleetReplicas {
			gametype.Status.CurrentFleetReplicas = gametype.Spec.FleetSpec.Scaling.Replicas
			fleet.Spec.Scaling.Replicas = gametype.Spec.FleetSpec.Scaling.Replicas
			if err := r.Update(ctx, &fleet); err != nil {
				return ctrl.Result{Requeue: true}, err, true
			}
			err := r.Status().Update(ctx, gametype)
			if err != nil {
				return ctrl.Result{Requeue: true}, err, true
			}
			r.emitEventf(gametype, corev1.EventTypeNormal, utils.ReasonGametypeReplicasUpdated, "Scaling gametype to %d", fleet.Spec.Scaling.Replicas)
		}
	}
	if len(fleets.Items) > 1 {
		var oldestFleet *gameserverv1alpha1.Fleet
		for _, fleet := range fleets.Items {
			if oldestFleet == nil {
				oldestFleet = &fleet
			} else if fleet.CreationTimestamp.Before(&oldestFleet.CreationTimestamp) {
				oldestFleet = &fleet
			}
		}

		if oldestFleet != nil && oldestFleet.GetDeletionTimestamp() == nil {
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeSpecUpdated, "Deleting extra fleet")
			if err := r.Delete(ctx, oldestFleet); err != nil {
				return ctrl.Result{}, err, true
			}
		}
	}
	return ctrl.Result{}, nil, false
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameTypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.GameType{}).
		Owns(&gameserverv1alpha1.Fleet{}).
		Complete(r)
}

// handleDeletion is used to trigger deletion of the GameType
// It first checks if we have the finalizer, then we can imagine we are still removing the fleets
// Once all fleets are removed, we remove the finalizer
func (r *GameTypeReconciler) handleDeletion(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) error {
	fmt.Printf("Triggered deletion for gametype\n")
	if controllerutil.ContainsFinalizer(gametype, TypeFinalizer) {
		fmt.Printf("Has finalizer in gametype\n")
		//Finalizer not yet removed, we can presume that fleet deletion in progress or starting
		fleets, err := utils.GetFleetsForType(ctx, r.Client, gametype, logger)
		if err != nil {
			return err
		}
		for _, fleet := range fleets.Items {
			r.emitEventf(gametype, corev1.EventTypeNormal, utils.ReasonGameTypeDeleting, "Deleting fleet %s", fleet.Name)
			if err := r.Delete(ctx, &fleet); err != nil {
				r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeServersDeleted, "Failed to delete fleet %s", fleet.Name)
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
			r.emitEvent(gametype, corev1.EventTypeNormal, utils.ReasonGametypeServersDeleted, "Removed finalizer")
		}
	}
	return nil
}

// handleCreation is used to initially create the underlying fleet
func (r *GameTypeReconciler) handleCreation(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) (ctrl.Result, error) {
	fleet := utils.GetFleetObjectForType(gametype)
	if err := r.Create(ctx, fleet); err != nil {
		r.emitEventf(gametype, corev1.EventTypeWarning, utils.ReasonGametypeReplicasUpdated, "Failed to create new fleet %s", err)
		logger.Error(err, "failed to create a new fleet for gametype")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// emitEvent is used by the GameTypeReconciler to quickly add new events to objects
func (r *GameTypeReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

// emitEventf is used by the GameTypeReconciler to add new events to objects with arguments
func (r *GameTypeReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}

// handleGametypeStatus is used by the GameTypeReconciler to make sure the fleet in gametype status is the newest one.
func (r *GameTypeReconciler) handleGametypeStatus(ctx context.Context, gametype *gameserverv1alpha1.GameType, logger logr.Logger) error {
	fleets, err := utils.GetFleetsForType(ctx, r.Client, gametype, logger)
	if err != nil {
		return err
	}
	var youngestFleet *gameserverv1alpha1.Fleet
	for _, fleet := range fleets.Items {
		if youngestFleet == nil || fleet.GetCreationTimestamp().After(youngestFleet.GetCreationTimestamp().Time) {
			youngestFleet = &fleet
		}
	}

	if youngestFleet != nil {
		gametype.Status.CurrentFleetName = youngestFleet.Name
		err = r.Status().Update(ctx, youngestFleet)
		if err != nil {
			return err
		}
	}
	return nil
}
