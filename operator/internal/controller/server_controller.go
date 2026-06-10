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
	"errors"
	"fmt"

	"github.com/MirrorStudios/fallernetes-operator/internal/builders"
	"github.com/MirrorStudios/fallernetes-operator/internal/sidecar"
	"github.com/MirrorStudios/fallernetes-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
)

const ServerFinalizer = "server.falloria.com/finalizer"

// ServerReconciler reconciles a Server object
type ServerReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	ErrorOnNotAllowed bool
	Recorder          record.EventRecorder
	DeletionAllowed   sidecar.Deletion
}

// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=servers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=servers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gameserver.falloria.com,resources=servers/finalizers,verbs=update

func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	server := &gameserverv1alpha1.Server{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get Server: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Handle finalizer addition
	if server.DeletionTimestamp == nil && !controllerutil.ContainsFinalizer(server, ServerFinalizer) {
		controllerutil.AddFinalizer(server, ServerFinalizer)
		if err := r.Update(ctx, server); err != nil {
			r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerUpdateFailed, "failed to update server: %s", err)
			return ctrl.Result{}, fmt.Errorf("failed to update server for finalizer: %s", err)
		}
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerFinalizerAdded, "Finalizer added")
		return ctrl.Result{}, nil
	}

	// Handle resource deletion
	if server.DeletionTimestamp != nil || !server.GetDeletionTimestamp().IsZero() {
		if err := r.handleDeletion(ctx, server); err != nil {
			if err.Error() == "server deletion not allowed" && !r.ErrorOnNotAllowed {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("failed to handle server deletion: %s", err)
		}
		controllerutil.RemoveFinalizer(server, ServerFinalizer)
		if err := r.Update(ctx, server); err != nil {
			r.emitEvent(server, corev1.EventTypeWarning, utils.ReasonServerUpdateFailed, "Failed to update server object")
			return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
		}
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerFinalizerRemoved, "Finalizer removed")
		return ctrl.Result{}, nil
	}

	// Ensure Pod exists
	podExists, err := r.ensurePodExists(ctx, server)
	if err != nil {
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonPodNotFound,
			Message:            fmt.Sprintf("Failed to create pod: %s", err),
			ObservedGeneration: server.Generation,
		})
		_ = r.Status().Update(ctx, server)
		return ctrl.Result{}, fmt.Errorf("failed to ensure Pod exists for Server: %w", err)
	}
	if !podExists {
		// Pod was just created; set initial Pending state and wait for pod event.
		server.Status.Phase = gameserverv1alpha1.ServerPhasePending
		server.Status.ObservedGeneration = server.Generation
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionUnknown,
			Reason:             gameserverv1alpha1.ReasonPodNotRunning,
			Message:            "Pod has been created and is starting",
			ObservedGeneration: server.Generation,
		})
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionPodScheduled,
			Status:             metav1.ConditionUnknown,
			Reason:             gameserverv1alpha1.ReasonPodPending,
			Message:            "Waiting for pod to be scheduled",
			ObservedGeneration: server.Generation,
		})
		if err := r.Status().Update(ctx, server); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update Server status: %w", err)
		}
		return ctrl.Result{}, nil
	}

	// Ensure pod has the finalizer
	update, err := r.ensurePodFinalizer(ctx, server)
	if err != nil || update {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.syncServerStatus(ctx, server)
}

// syncServerStatus reads the pod's current state and writes Phase, PodPhase, and
// condition fields onto the server status, then persists the status subresource.
func (r *ServerReconciler) syncServerStatus(ctx context.Context, server *gameserverv1alpha1.Server) error {
	pod := &corev1.Pod{}
	namespacedName := types.NamespacedName{Namespace: server.Namespace, Name: server.Name + "-pod"}

	oldPhase := server.Status.Phase
	server.Status.ObservedGeneration = server.Generation

	if err := r.Get(ctx, namespacedName, pod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get pod for status sync: %w", err)
		}
		server.Status.Phase = gameserverv1alpha1.ServerPhasePending
		server.Status.PodPhase = ""
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             gameserverv1alpha1.ReasonPodNotFound,
			Message:            "Pod does not exist",
			ObservedGeneration: server.Generation,
		})
		meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
			Type:               gameserverv1alpha1.ConditionPodScheduled,
			Status:             metav1.ConditionUnknown,
			Reason:             gameserverv1alpha1.ReasonPodNotFound,
			Message:            "Pod does not exist",
			ObservedGeneration: server.Generation,
		})
		r.emitEvent(server, corev1.EventTypeWarning, utils.ReasonServerPodMissing, "Pod does not exist")
	} else {
		server.Status.PodPhase = pod.Status.Phase

		if pod.Spec.NodeName != "" {
			meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
				Type:               gameserverv1alpha1.ConditionPodScheduled,
				Status:             metav1.ConditionTrue,
				Reason:             gameserverv1alpha1.ReasonPodScheduled,
				Message:            fmt.Sprintf("Pod scheduled on node %s", pod.Spec.NodeName),
				ObservedGeneration: server.Generation,
			})
		} else {
			meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
				Type:               gameserverv1alpha1.ConditionPodScheduled,
				Status:             metav1.ConditionFalse,
				Reason:             gameserverv1alpha1.ReasonPodNotScheduled,
				Message:            "Pod has not been scheduled to a node yet",
				ObservedGeneration: server.Generation,
			})
		}

		if pod.Status.Phase == corev1.PodRunning {
			server.Status.Phase = gameserverv1alpha1.ServerPhaseReady
			meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
				Type:               gameserverv1alpha1.ConditionReady,
				Status:             metav1.ConditionTrue,
				Reason:             gameserverv1alpha1.ReasonPodRunning,
				Message:            "Pod is running",
				ObservedGeneration: server.Generation,
			})
			if oldPhase != gameserverv1alpha1.ServerPhaseReady {
				r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerReady, "Server is ready")
			}
		} else {
			server.Status.Phase = gameserverv1alpha1.ServerPhasePending
			meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
				Type:               gameserverv1alpha1.ConditionReady,
				Status:             metav1.ConditionFalse,
				Reason:             gameserverv1alpha1.ReasonPodNotRunning,
				Message:            fmt.Sprintf("Pod is in phase %s", pod.Status.Phase),
				ObservedGeneration: server.Generation,
			})
			if oldPhase == gameserverv1alpha1.ServerPhaseReady {
				r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerNotReady, "Server is no longer ready: pod is in phase %s", pod.Status.Phase)
			}
		}
	}

	if err := r.Status().Update(ctx, server); err != nil {
		return fmt.Errorf("failed to update Server status: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gameserverv1alpha1.Server{}).
		Owns(&corev1.Pod{}).
		Named("server").
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
		newPod := builders.GetNewPod(server, server.Namespace)
		err = controllerutil.SetControllerReference(server, newPod, r.Scheme)
		if err != nil {
			r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerPodCreationFailed, "failed to set pod owner reference: %s", err)
			return false, fmt.Errorf("failed to set controller reference on Pod: %w", err)
		}
		if err := r.Create(ctx, newPod); err != nil {
			r.emitEventf(server, corev1.EventTypeWarning, utils.ReasonServerPodCreationFailed, "Pod creation errored: %s", err)
			return false, err
		}
		r.emitEventf(server, corev1.EventTypeNormal, utils.ReasonServerPodCreated, "Pod created with sidecar image %s", server.Spec.SidecarSettings.SidecarImage)
		return false, nil
	}
	return true, nil
}

// handleDeletion handles the deletion process of the Server
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
		r.emitEvent(pod, corev1.EventTypeWarning, utils.ReasonServerDeletionNotAllowed, "Server did not respond with allowed")
		r.emitEvent(server, corev1.EventTypeWarning, utils.ReasonServerDeletionNotAllowed, "Server did not respond with allowed")
		return errors.New("server deletion not allowed")
	}

	if pod != nil && controllerutil.ContainsFinalizer(pod, ServerFinalizer) {
		controllerutil.RemoveFinalizer(pod, ServerFinalizer)
		r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerPodFinalizerRemoved, "Pod finalizer removed")
		r.emitEvent(pod, corev1.EventTypeNormal, utils.ReasonServerPodFinalizerRemoved, "Pod finalizer removed")
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
	if controllerutil.ContainsFinalizer(pod, ServerFinalizer) {
		return false, nil
	}
	controllerutil.AddFinalizer(pod, ServerFinalizer)
	r.emitEvent(server, corev1.EventTypeNormal, utils.ReasonServerPodFinalizerAdded, "Pod finalizer added")
	r.emitEvent(pod, corev1.EventTypeNormal, utils.ReasonServerPodFinalizerAdded, "Pod finalizer added")
	if err := r.Update(ctx, pod); err != nil {
		return false, fmt.Errorf("failed to add finalizer to pod: %s", err)
	}
	return true, nil
}

func (r *ServerReconciler) emitEvent(object runtime.Object, eventtype string, reason utils.EventReason, message string) {
	r.Recorder.Event(object, eventtype, string(reason), message)
}

func (r *ServerReconciler) emitEventf(object runtime.Object, eventtype string, reason utils.EventReason, message string, args ...interface{}) {
	r.Recorder.Eventf(object, eventtype, string(reason), message, args...)
}
