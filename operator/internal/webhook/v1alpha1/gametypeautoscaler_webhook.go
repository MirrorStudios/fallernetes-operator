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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var gametypeautoscalerlog = logf.Log.WithName("gametypeautoscaler-resource")

// SetupGameTypeAutoscalerWebhookWithManager registers the webhook for GameTypeAutoscaler in the manager.
func SetupGameTypeAutoscalerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gameserverv1alpha1.GameTypeAutoscaler{}).
		WithValidator(&GameTypeAutoscalerCustomValidator{}).
		WithDefaulter(&GameTypeAutoscalerCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-gameserver-falloria-com-v1alpha1-gametypeautoscaler,mutating=true,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=create;update,versions=v1alpha1,name=mgametypeautoscaler-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeAutoscalerCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind GameTypeAutoscaler when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type GameTypeAutoscalerCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &GameTypeAutoscalerCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind GameTypeAutoscaler.
func (d *GameTypeAutoscalerCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	gametypeautoscaler, ok := obj.(*gameserverv1alpha1.GameTypeAutoscaler)

	if !ok {
		return fmt.Errorf("expected an GameTypeAutoscaler object but got %T", obj)
	}
	gametypeautoscalerlog.Info("Defaulting for GameTypeAutoscaler", "name", gametypeautoscaler.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-gametypeautoscaler,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=create;update,versions=v1alpha1,name=vgametypeautoscaler-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeAutoscalerCustomValidator struct is responsible for validating the GameTypeAutoscaler resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GameTypeAutoscalerCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &GameTypeAutoscalerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GameTypeAutoscaler.
func (v *GameTypeAutoscalerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gametypeautoscaler, ok := obj.(*gameserverv1alpha1.GameTypeAutoscaler)
	if !ok {
		return nil, fmt.Errorf("expected a GameTypeAutoscaler object but got %T", obj)
	}
	gametypeautoscalerlog.Info("Validation for GameTypeAutoscaler upon creation", "name", gametypeautoscaler.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GameTypeAutoscaler.
func (v *GameTypeAutoscalerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	gametypeautoscaler, ok := newObj.(*gameserverv1alpha1.GameTypeAutoscaler)
	if !ok {
		return nil, fmt.Errorf("expected a GameTypeAutoscaler object for the newObj but got %T", newObj)
	}
	gametypeautoscalerlog.Info("Validation for GameTypeAutoscaler upon update", "name", gametypeautoscaler.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GameTypeAutoscaler.
func (v *GameTypeAutoscalerCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gametypeautoscaler, ok := obj.(*gameserverv1alpha1.GameTypeAutoscaler)
	if !ok {
		return nil, fmt.Errorf("expected a GameTypeAutoscaler object but got %T", obj)
	}
	gametypeautoscalerlog.Info("Validation for GameTypeAutoscaler upon deletion", "name", gametypeautoscaler.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
