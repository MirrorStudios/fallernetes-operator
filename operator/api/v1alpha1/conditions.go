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

// Condition type constants for all CRDs.
const (
	ConditionReady        = "Ready"
	ConditionPodScheduled = "PodScheduled"
	ConditionAvailable    = "Available"
	ConditionScaling      = "Scaling"
	ConditionRollingUpdate = "RollingUpdate"
)

// Reason constants used in condition Reason fields.
const (
	ReasonPodNotFound        = "PodNotFound"
	ReasonPodNotRunning      = "PodNotRunning"
	ReasonPodRunning         = "PodRunning"
	ReasonPodPending         = "PodPending"
	ReasonPodScheduled       = "PodScheduled"
	ReasonPodNotScheduled    = "PodNotScheduled"
	ReasonReplicasMismatch   = "ReplicasMismatch"
	ReasonAtDesiredCount     = "AtDesiredCount"
	ReasonScalingUp          = "ScalingUp"
	ReasonScalingDown        = "ScalingDown"
	ReasonNoReplicas         = "NoReplicas"
	ReasonReplicasAvailable  = "ReplicasAvailable"
	ReasonRolloutInProgress  = "RolloutInProgress"
	ReasonNoRollout          = "NoRollout"
	ReasonNotScaling         = "NotScaling"
	ReasonAutoscalerReady    = "AutoscalerReady"
	ReasonInitializing       = "Initializing"
)
