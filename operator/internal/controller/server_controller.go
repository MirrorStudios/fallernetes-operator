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
	"errors"
	"fmt"
	"github.com/MirrorStudios/fallernetes/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const SERVER_FINALIZER = "server.falloria.com/finalizer"

// ServerReconciler reconciles a Server object
type ServerReconciler struct {
	client.Client
	ErrorOnNotAllowed bool
	Scheme            *runtime.Scheme
	Recorder          record.EventRecorder
	DeletionAllowed   utils.Deletion
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=servers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=servers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=servers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Server resource
	server := &gameserverv1alpha1.Server{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		if client.IgnoreNotFound(err) != nil { //If some other error
			return ctrl.Result{}, fmt.Errorf("failed to get Server: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Handle finalizer addition
	if server.DeletionTimestamp == nil && !controllerutil.ContainsFinalizer(server, SERVER_FINALIZER) {
		controllerutil.AddFinalizer(server, SERVER_FINALIZER)
		if err := r.Update(ctx, server); err != nil {
			r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerUpdateFAiled, "failed to update server: %s", err)
			return ctrl.Result{}, fmt.Errorf("failed to update server for finalizer: %s", err)
		}
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerInitialized, "Finalizer added")
		return ctrl.Result{}, nil
	}

	// Handle resource deletion
	if server.DeletionTimestamp != nil || !server.GetDeletionTimestamp().IsZero() {
		if err := r.handleDeletion(ctx, server); err != nil {
			if err.Error() == "server deletion not allowed" && !r.ErrorOnNotAllowed {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{Requeue: true}, fmt.Errorf("failed to handle server deletion: %s", err)
		}
		controllerutil.RemoveFinalizer(server, SERVER_FINALIZER)
		if err := r.Update(ctx, server); err != nil {
			r.emitEvent(server, corev1.EventTypeWarning, utils.ReasonServerDeletionAllowed, "Failed to update server object")
			return ctrl.Result{Requeue: true}, fmt.Errorf("failed to remove finalizer: %w", err)
		}
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerDeletionAllowed, "Finalizer removed")
		return ctrl.Result{Requeue: true}, nil // Return after finalizer removal
	}

	// Ensure Pod exists
	podExists, err := r.ensurePodExists(ctx, server)
	if err != nil {
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update server: %w", err)
		}
		return ctrl.Result{}, fmt.Errorf("failed to ensure Pod exists for Server: %w", err)
	}
	if !podExists {
		// If a Pod was created, exit early to requeue the reconciliation
		return ctrl.Result{Requeue: true}, nil
	}

	// Ensure pod has the finalizers
	update, err := r.ensurePodFinalizer(ctx, server)
	if err != nil || update {
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, server); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update Server resource: %w", err)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.Server{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}

// ensurePodExists makes sure that the pod with the matching name exists
func (r *ServerReconciler) ensurePodExists(ctx context.Context, server *gameserverv1alpha1.Server) (bool, error) {
	pod := &corev1.Pod{}
	namespacedName := types.NamespacedName{Namespace: server.Namespace, Name: server.Name + "-pod"}
	err := r.Get(ctx, namespacedName, pod)

	if client.IgnoreNotFound(err) != nil {
		return false, fmt.Errorf("failed to get Pod resource: %w", err)
	}

	if err != nil { // Pod does not exist
		newPod := utils.GetNewPod(server, server.Namespace)
		r.emitEventf(server, corev1.EventTypeNormal, utils.ReasonServerInitialized, "Setting up sidecar with image %s", server.Spec.SidecarSettings.SidecarImage)
		err = controllerutil.SetControllerReference(server, newPod, r.Scheme)
		if err != nil {
			r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerInitialized, "failed to set pod owner reference: %s", err)
			return false, fmt.Errorf("failed to set controller reference on Pod: %w", err)
		}
		if err := r.Create(ctx, newPod); err != nil {
			meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
				Type:               "PodFailed",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "PodCreationFailed",
				Message:            "Failed to create the Pod",
			})
			r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerPodCreationFailed, "Pod creation errored: %s", err)
			return false, err
		}

		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:               "PodCreated",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "PodCreatedSuccessfully",
			Message:            "Pod has been successfully created",
		})
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerInitialized, "Pod created successfully")
		return false, nil
	}
	return true, nil
}

// handleDeletion handles the deletion process of the Server, by checking with the sidecar if it is allowed to be deleted
func (r *ServerReconciler) handleDeletion(ctx context.Context, server *gameserverv1alpha1.Server) error {
	pod := &corev1.Pod{}
	namespacedName := types.NamespacedName{Namespace: server.Namespace, Name: server.Name + "-pod"}
	if err := r.Get(ctx, namespacedName, pod); err != nil {
		return err
	}
	allowed, err := r.DeletionAllowed.IsDeletionAllowed(server, pod)
	if err != nil {
		r.emitEvent(pod, corev1.EventTypeWarning, utils.ReasonServerDeletionNotAllowed, "Deletion request did not succeed")
		r.emitEvent(server, corev1.EventTypeWarning, utils.ReasonServerDeletionNotAllowed, "Deletion request did not succeed")
		return fmt.Errorf("failed to check for deletion for server: %s", err)
	}
	if !allowed {
		r.emitEvent(pod, corev1.EventTypeNormal, utils.ReasonServerDeletionAllowed, "Server did not respond with allowed")
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerDeletionAllowed, "Server did not respond with allowed")
		return errors.New("server deletion not allowed")
	}

	if pod != nil && controllerutil.ContainsFinalizer(pod, SERVER_FINALIZER) {
		controllerutil.RemoveFinalizer(pod, SERVER_FINALIZER)
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerDeletionAllowed, "Pod finalizer removed")
		r.emitEvent(pod, corev1.EventTypeNormal, utils.ReasonServerDeletionAllowed, "Pod finalizer removed")
		if err := r.Update(ctx, pod); err != nil {
			return err
		}
		if err := r.Get(ctx, namespacedName, pod); err != nil {
			return err
		}
	}

	if err := r.Delete(ctx, pod); err != nil {
		return err
	}

	meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
		Type:               "Finalizing",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "PodDeleted",
		Message:            "Pod successfully deleted during finalization",
	})
	r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerPodDeleted, "Pod successfully deleted during finalization")
	return nil
}

// ensurePodFinalizer makes sure the pod has the finalizer
func (r *ServerReconciler) ensurePodFinalizer(ctx context.Context, server *gameserverv1alpha1.Server) (bool, error) {
	pod := &corev1.Pod{}
	namespacedName := types.NamespacedName{Namespace: server.Namespace, Name: server.Name + "-pod"}
	if err := r.Get(ctx, namespacedName, pod); err != nil {
		return false, err
	}
	if controllerutil.ContainsFinalizer(pod, SERVER_FINALIZER) {
		return false, nil
	}
	controllerutil.AddFinalizer(pod, SERVER_FINALIZER)
	r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerInitialized, "Pod finalizer added")
	r.emitEvent(pod, corev1.EventTypeNormal, utils.ReasonServerInitialized, "Pod finalizer added")
	if err := r.Update(ctx, pod); err != nil {
		return false, fmt.Errorf("failed to add finalizer to pod: %s", err)
	}
	return true, nil
}

// emitEvent is used by the ServerReconciler to add events to an object easily
func (r *ServerReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

// emitEventf is used by the ServerReconciler to add events with arguments to an object easily
func (r *ServerReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}
