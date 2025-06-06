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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PolicyStrategy string
type SyncStrategy string

var validPolicyStrategies = map[PolicyStrategy]struct{}{
	Webhook: {},
	// Add new strategies here as needed
}
var validSyncStrategy = map[SyncStrategy]struct{}{
	FixedInterval: {},
	// Add new strategies here as needed
}

var (
	Webhook PolicyStrategy = "webhook"

	FixedInterval SyncStrategy = "fixedinterval"
)

//The following structs handle the policy of how to sync

type AutoscalePolicy struct {
	// +kubebuilder:validation:Enum=webhook
	Type                  PolicyStrategy        `json:"type"`
	WebhookAutoscalerSpec WebhookAutoscalerSpec `json:"webhook"`
}

type WebhookAutoscalerSpec struct {
	// +kubebuilder:validation:Optional
	Url  *string `json:"url,omitempty"`
	Path *string `json:"path"`
	// +kubebuilder:validation:Optional
	Service *Service `json:"service"`
}

type Service struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Port      int    `json:"port"`
}

// The following sync structs handle when to sync
type Sync struct {
	// +kubebuilder:validation:Enum=fixedinterval
	Type SyncStrategy     `json:"type"`
	Time *metav1.Duration `json:"interval"`
}

// GameTypeAutoscalerSpec defines the desired state of GameTypeAutoscaler.
type GameTypeAutoscalerSpec struct {
	GameTypeName    string          `json:"gameTypeName"`
	AutoscalePolicy AutoscalePolicy `json:"policy"`
	Sync            Sync            `json:"sync"`
}

// GameTypeAutoscalerStatus defines the observed state of GameTypeAutoscaler.
type GameTypeAutoscalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GameTypeAutoscaler is the Schema for the gametypeautoscalers API.
type GameTypeAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameTypeAutoscalerSpec   `json:"spec,omitempty"`
	Status GameTypeAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameTypeAutoscalerList contains a list of GameTypeAutoscaler.
type GameTypeAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameTypeAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameTypeAutoscaler{}, &GameTypeAutoscalerList{})
}
