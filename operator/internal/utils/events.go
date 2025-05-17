package utils

type EventReason string

const (
	ReasonServerInitialized        EventReason = "ServerInitialized"
	ReasonServerDeletionAllowed    EventReason = "ServerDeletionAllowed"
	ReasonServerDeletionNotAllowed EventReason = "ServerDeletionNotAllowed"
	ReasonServerPodDeleted         EventReason = "ServerPodDeleted"
	ReasonServerPodCreationFailed  EventReason = "ServerPodCreationFailed"
	ReasonServerUpdateFAiled       EventReason = "ServerUpdateFailed"

	ReasonFleetInitialized    EventReason = "FleetInitialized"
	ReasonFleetUpdateFailed   EventReason = "FleetUpdateFailed"
	ReasonFleetServersRemoved EventReason = "FleetServersRemoved"
	ReasonFleetScaleServers   EventReason = "FleetScaleServers"

	ReasonGametypeInitialized     EventReason = "GametypeInitialized"
	ReasonGameTypeDeleting        EventReason = "GameTypeDeleting"
	ReasonGametypeServersDeleted  EventReason = "GametypeServersDeleted"
	ReasonGametypeSpecUpdated     EventReason = "GametypeSpecUpdated"
	ReasonGametypeReplicasUpdated EventReason = "GametypeReplicasUpdated"

	ReasonGameTypeAutoscalerInvalidServer          EventReason = "GameAutoscalerInvalidServer"
	ReasonGameTypeAutoscalerInvalidAutoscalePolicy EventReason = "GameautoscalerInvalidAutoscalePolicy"
	ReasonGameTypeAutoscalerInvalidSyncType        EventReason = "GameautoscalerInvalidSyncType"
	ReasonGameTypeAutoscalerWebhook                EventReason = "GameautoscalerWebhook"
	ReasonGameTypeAutoscalerScale                  EventReason = "GameautoscalerScale"
)
