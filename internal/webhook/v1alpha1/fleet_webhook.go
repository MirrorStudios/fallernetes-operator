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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var fleetlog = logf.Log.WithName("fleet-resource")

// SetupFleetWebhookWithManager registers the webhook for Fleet in the manager.
func SetupFleetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gameserverv1alpha1.Fleet{}).
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

var _ webhook.CustomDefaulter = &FleetCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Fleet.
func (d *FleetCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	fleet, ok := obj.(*gameserverv1alpha1.Fleet)

	if !ok {
		return fmt.Errorf("expected an Fleet object but got %T", obj)
	}
	fleetlog.Info("Defaulting for Fleet", "name", fleet.GetName())
	if fleet.Spec.ServerSpec.TimeOut == nil {
		fleet.Spec.ServerSpec.TimeOut = &metav1.Duration{Duration: time.Minute * 40}
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-fleet,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=fleets,verbs=create;update;delete,versions=v1alpha1,name=vfleet-v1alpha1.kb.io,admissionReviewVersions=v1

// FleetCustomValidator struct is responsible for validating the Fleet resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type FleetCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &FleetCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Fleet.
func (v *FleetCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, ok := obj.(*gameserverv1alpha1.Fleet)
	if !ok {
		return nil, fmt.Errorf("expected a Fleet object but got %T", obj)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Fleet.
func (v *FleetCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, ok := newObj.(*gameserverv1alpha1.Fleet)
	if !ok {
		return nil, fmt.Errorf("expected a Fleet object for the newObj but got %T", newObj)
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Fleet.
func (v *FleetCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, ok := obj.(*gameserverv1alpha1.Fleet)
	if !ok {
		return nil, fmt.Errorf("expected a Fleet object but got %T", obj)
	}
	return nil, nil
}
