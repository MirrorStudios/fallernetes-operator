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
