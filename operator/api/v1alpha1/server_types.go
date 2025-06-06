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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerSpec defines the desired state of Server
type ServerSpec struct {
	Pod v1.PodSpec `json:"pod,omitempty"`
	// +kubebuilder:validation:Optional
	TimeOut *metav1.Duration `json:"timeout,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AllowForceDelete bool `json:"allowForceDelete,omitempty"`
	// +kubebuilder:validation:Optional
	SidecarSettings *SidecarSettings `json:"sidecar,omitempty"`
	// +kubebuilder:validation:Optional
	GameInfo *GameInfo `json:"gameInfo,omitempty"`
}

type GameInfo struct {
	// +kubebuilder:validation:Optional
	Capacity *int `json:"capacity,omitempty"`
}

type SidecarSettings struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=8080
	Port *int `json:"port"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="unfamousthomas/fallernetes-sidecar:main"
	SidecarImage *string `json:"image,omitempty"`
	// +kubebuilder:validation:Optional
	LogDebug bool `json:"logDebug,omitempty"`
}

// ServerStatus defines the observed state of Server
type ServerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Server is the Schema for the servers API
type Server struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerSpec   `json:"spec,omitempty"`
	Status ServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerList contains a list of Server.
type ServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Server `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Server{}, &ServerList{})
}
