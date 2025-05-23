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

// GameTypeSpec defines the desired state of GameType
type GameTypeSpec struct {
	FleetSpec FleetSpec `json:"fleetSpec"`
}

// GameTypeStatus defines the observed state of GameType
type GameTypeStatus struct {
	Conditions       []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	CurrentFleetName string             `json:"fleetName"`
	// +kubebuilder:default=0
	CurrentFleetReplicas int32 `json:"fleetReplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GameType is the Schema for the gametypes API
type GameType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameTypeSpec   `json:"spec,omitempty"`
	Status GameTypeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameTypeList contains a list of GameType.
type GameTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameType `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameType{}, &GameTypeList{})
}
