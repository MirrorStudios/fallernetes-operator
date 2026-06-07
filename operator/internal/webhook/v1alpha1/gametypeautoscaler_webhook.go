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

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var gametypeautoscalerlog = logf.Log.WithName("gametypeautoscaler-resource")

// SetupGameTypeAutoscalerWebhookWithManager registers the webhook for GameTypeAutoscaler in the manager.
func SetupGameTypeAutoscalerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &gameserverv1alpha1.GameTypeAutoscaler{}).
		WithValidator(&GameTypeAutoscalerCustomValidator{}).
		WithDefaulter(&GameTypeAutoscalerCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gameserver-falloria-com-v1alpha1-gametypeautoscaler,mutating=true,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=create;update,versions=v1alpha1,name=mgametypeautoscaler-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeAutoscalerCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind GameTypeAutoscaler when those are created or updated.
type GameTypeAutoscalerCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind GameTypeAutoscaler.
func (d *GameTypeAutoscalerCustomDefaulter) Default(_ context.Context, obj *gameserverv1alpha1.GameTypeAutoscaler) error {
	gametypeautoscalerlog.Info("Defaulting for GameTypeAutoscaler", "name", obj.GetName())

	return nil
}

// NOTE: If you want to customize the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-gametypeautoscaler,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypeautoscalers,verbs=create;update,versions=v1alpha1,name=vgametypeautoscaler-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeAutoscalerCustomValidator struct is responsible for validating the GameTypeAutoscaler resource
// when it is created, updated, or deleted.
type GameTypeAutoscalerCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GameTypeAutoscaler.
func (v *GameTypeAutoscalerCustomValidator) ValidateCreate(_ context.Context, obj *gameserverv1alpha1.GameTypeAutoscaler) (admission.Warnings, error) {
	gametypeautoscalerlog.Info("Validation for GameTypeAutoscaler upon creation", "name", obj.GetName())

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GameTypeAutoscaler.
func (v *GameTypeAutoscalerCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *gameserverv1alpha1.GameTypeAutoscaler) (admission.Warnings, error) {
	gametypeautoscalerlog.Info("Validation for GameTypeAutoscaler upon update", "name", newObj.GetName())

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GameTypeAutoscaler.
func (v *GameTypeAutoscalerCustomValidator) ValidateDelete(_ context.Context, obj *gameserverv1alpha1.GameTypeAutoscaler) (admission.Warnings, error) {
	gametypeautoscalerlog.Info("Validation for GameTypeAutoscaler upon deletion", "name", obj.GetName())

	return nil, nil
}
