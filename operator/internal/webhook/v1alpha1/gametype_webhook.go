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

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var gametypelog = logf.Log.WithName("gametype-resource")

// SetupGameTypeWebhookWithManager registers the webhook for GameType in the manager.
func SetupGameTypeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &gameserverv1alpha1.GameType{}).
		WithValidator(&GameTypeCustomValidator{}).
		WithDefaulter(&GameTypeCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gameserver-falloria-com-v1alpha1-gametype,mutating=true,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypes,verbs=create;update,versions=v1alpha1,name=mgametype-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind GameType when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type GameTypeCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind GameType.
func (d *GameTypeCustomDefaulter) Default(_ context.Context, obj *gameserverv1alpha1.GameType) error {
	gametypelog.Info("Defaulting for GameType", "name", obj.GetName())

	return nil
}

// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-gametype,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypes,verbs=create;update,versions=v1alpha1,name=vgametype-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeCustomValidator struct is responsible for validating the GameType resource
// when it is created, updated, or deleted.
type GameTypeCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GameType.
func (v *GameTypeCustomValidator) ValidateCreate(_ context.Context, obj *gameserverv1alpha1.GameType) (admission.Warnings, error) {
	gametypelog.Info("Validation for GameType upon creation", "name", obj.GetName())
	return nil, validateFleetScaling(obj.Spec.FleetSpec.Scaling)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GameType.
func (v *GameTypeCustomValidator) ValidateUpdate(_ context.Context, _, newObj *gameserverv1alpha1.GameType) (admission.Warnings, error) {
	gametypelog.Info("Validation for GameType upon update", "name", newObj.GetName())
	return nil, validateFleetScaling(newObj.Spec.FleetSpec.Scaling)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GameType.
func (v *GameTypeCustomValidator) ValidateDelete(_ context.Context, obj *gameserverv1alpha1.GameType) (admission.Warnings, error) {
	gametypelog.Info("Validation for GameType upon deletion", "name", obj.GetName())

	return nil, nil
}
