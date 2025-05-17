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
var gametypelog = logf.Log.WithName("gametype-resource")

// SetupGameTypeWebhookWithManager registers the webhook for GameType in the manager.
func SetupGameTypeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gameserverv1alpha1.GameType{}).
		WithValidator(&GameTypeCustomValidator{}).
		WithDefaulter(&GameTypeCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-gameserver-falloria-com-v1alpha1-gametype,mutating=true,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypes,verbs=create;update,versions=v1alpha1,name=mgametype-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind GameType when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type GameTypeCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &GameTypeCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind GameType.
func (d *GameTypeCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	gametype, ok := obj.(*gameserverv1alpha1.GameType)

	if !ok {
		return fmt.Errorf("expected an GameType object but got %T", obj)
	}
	gametypelog.Info("Defaulting for GameType", "name", gametype.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-gametype,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=gametypes,verbs=create;update,versions=v1alpha1,name=vgametype-v1alpha1.kb.io,admissionReviewVersions=v1

// GameTypeCustomValidator struct is responsible for validating the GameType resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GameTypeCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &GameTypeCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GameType.
func (v *GameTypeCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gametype, ok := obj.(*gameserverv1alpha1.GameType)
	if !ok {
		return nil, fmt.Errorf("expected a GameType object but got %T", obj)
	}
	gametypelog.Info("Validation for GameType upon creation", "name", gametype.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GameType.
func (v *GameTypeCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	gametype, ok := newObj.(*gameserverv1alpha1.GameType)
	if !ok {
		return nil, fmt.Errorf("expected a GameType object for the newObj but got %T", newObj)
	}
	gametypelog.Info("Validation for GameType upon update", "name", gametype.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GameType.
func (v *GameTypeCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gametype, ok := obj.(*gameserverv1alpha1.GameType)
	if !ok {
		return nil, fmt.Errorf("expected a GameType object but got %T", obj)
	}
	gametypelog.Info("Validation for GameType upon deletion", "name", gametype.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
