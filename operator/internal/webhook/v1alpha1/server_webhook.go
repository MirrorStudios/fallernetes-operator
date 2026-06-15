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

	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var serverlog = logf.Log.WithName("server-resource")

// SetupServerWebhookWithManager registers the webhook for Server in the manager.
func SetupServerWebhookWithManager(mgr ctrl.Manager, defaultSidecarImage string) error {
	return ctrl.NewWebhookManagedBy(mgr, &gameserverv1alpha1.Server{}).
		WithValidator(&ServerCustomValidator{}).
		WithDefaulter(&ServerCustomDefaulter{DefaultSidecarImage: defaultSidecarImage}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gameserver-falloria-com-v1alpha1-server,mutating=true,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=servers,verbs=create;update,versions=v1alpha1,name=mserver-v1alpha1.kb.io,admissionReviewVersions=v1

// ServerCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Server when those are created or updated.
type ServerCustomDefaulter struct {
	DefaultSidecarImage string
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Server.
func (d *ServerCustomDefaulter) Default(_ context.Context, server *gameserverv1alpha1.Server) error {
	serverlog.Info("Defaulting for Server", "name", server.GetName())

	defaultSidecarSettings(server, d.DefaultSidecarImage)

	return nil
}

func defaultSidecarSettings(server *gameserverv1alpha1.Server, defaultImage string) {
	sidecarSettings := server.Spec.SidecarSettings
	if sidecarSettings == nil {
		sidecarSettings = &gameserverv1alpha1.SidecarSettings{}
	}

	if sidecarSettings.SidecarImage == nil {
		sidecarSettings.SidecarImage = &defaultImage
	}

	if sidecarSettings.Port == nil {
		defaultPort := 8080
		sidecarSettings.Port = &defaultPort
	}

	server.Spec.SidecarSettings = sidecarSettings
}

// +kubebuilder:webhook:path=/validate-gameserver-falloria-com-v1alpha1-server,mutating=false,failurePolicy=fail,sideEffects=None,groups=gameserver.falloria.com,resources=servers,verbs=create;update,versions=v1alpha1,name=vserver-v1alpha1.kb.io,admissionReviewVersions=v1

// ServerCustomValidator struct is responsible for validating the Server resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ServerCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Server.
func (v *ServerCustomValidator) ValidateCreate(_ context.Context, server *gameserverv1alpha1.Server) (admission.Warnings, error) {
	serverlog.Info("Validation for Server upon creation", "name", server.GetName())
	return nil, validateServer(server)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Server.
func (v *ServerCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *gameserverv1alpha1.Server) (admission.Warnings, error) {
	serverlog.Info("Validation for Server upon update", "name", newObj.GetName())
	if !equality.Semantic.DeepEqual(oldObj.Spec.Pod, newObj.Spec.Pod) {
		return nil, fmt.Errorf("spec.pod is immutable: the pod spec cannot be changed after creation")
	}
	return nil, validateServer(newObj)
}

func validateServer(server *gameserverv1alpha1.Server) error {
	if s := server.Spec.SidecarSettings; s != nil && s.Port != nil {
		if *s.Port < 1 || *s.Port > 65535 {
			return fmt.Errorf("spec.sidecarSettings.port must be in range 1–65535, got %d", *s.Port)
		}
	}
	return nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Server.
func (v *ServerCustomValidator) ValidateDelete(_ context.Context, obj *gameserverv1alpha1.Server) (admission.Warnings, error) {
	serverlog.Info("Validation for Server upon deletion", "name", obj.GetName())

	return nil, nil
}
