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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var fleetlog = logf.Log.WithName("fleet-resource")

// SetupFleetWebhookWithManager registers the webhook for Fleet in the manager.
func SetupFleetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &gameserverv1alpha1.Fleet{}).
		WithValidator(&FleetCustomValidator{}).
		WithDefaulter(&FleetCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gameserver-falloria-com-v1alpha1-fleet,mutating=true,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=fleets,verbs=create;update,versions=v1alpha1,name=mfleet-v1alpha1.kb.io,admissionReviewVersions=v1

// FleetCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Fleet when those are created or updated.
type FleetCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Fleet.
func (d *FleetCustomDefaulter) Default(_ context.Context, fleet *gameserverv1alpha1.Fleet) error {
	fleetlog.Info("Defaulting for Fleet", "name", fleet.GetName())
	if fleet.Spec.ServerSpec.TimeOut == nil {
		fleet.Spec.ServerSpec.TimeOut = &metav1.Duration{Duration: time.Minute * 40}
	}
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-fleet,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=fleets,verbs=create;update,versions=v1alpha1,name=vfleet-v1alpha1.kb.io,admissionReviewVersions=v1

// FleetCustomValidator struct is responsible for validating the Fleet resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type FleetCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Fleet.
func (v *FleetCustomValidator) ValidateCreate(_ context.Context, obj *gameserverv1alpha1.Fleet) (admission.Warnings, error) {
	fleetlog.Info("Validation for Fleet upon creation", "name", obj.GetName())
	return nil, validateFleet(obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Fleet.
func (v *FleetCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *gameserverv1alpha1.Fleet) (admission.Warnings, error) {
	fleetlog.Info("Validation for Fleet upon update", "name", newObj.GetName())
	return nil, validateFleet(newObj)
}

func validateFleet(fleet *gameserverv1alpha1.Fleet) error {
	if fleet.Spec.Scaling.Replicas < 0 {
		return fmt.Errorf("spec.scaling.replicas must be >= 0, got %d", fleet.Spec.Scaling.Replicas)
	}
	validPriorities := map[gameserverv1alpha1.Priority]bool{
		gameserverv1alpha1.OldestFirst: true,
		gameserverv1alpha1.NewestFirst: true,
	}
	if !validPriorities[fleet.Spec.Scaling.AgePriority] {
		return fmt.Errorf("spec.scaling.agePriority must be one of oldest_first|newest_first, got %q", fleet.Spec.Scaling.AgePriority)
	}
	return nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Fleet.
func (v *FleetCustomValidator) ValidateDelete(_ context.Context, obj *gameserverv1alpha1.Fleet) (admission.Warnings, error) {
	fleetlog.Info("Validation for Fleet upon deletion", "name", obj.GetName())

	return nil, nil
}
