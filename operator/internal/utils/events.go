package utils

type EventReason string

const (
	// Server reasons
	ReasonServerFinalizerAdded      EventReason = "ServerFinalizerAdded"
	ReasonServerFinalizerRemoved    EventReason = "ServerFinalizerRemoved"
	ReasonServerPodCreated          EventReason = "ServerPodCreated"
	ReasonServerPodFinalizerAdded   EventReason = "ServerPodFinalizerAdded"
	ReasonServerPodFinalizerRemoved EventReason = "ServerPodFinalizerRemoved"
	ReasonServerReady               EventReason = "ServerReady"
	ReasonServerNotReady            EventReason = "ServerNotReady"
	ReasonServerPodMissing          EventReason = "ServerPodMissing"
	ReasonServerDeletionAllowed     EventReason = "ServerDeletionAllowed"
	ReasonServerDeletionNotAllowed  EventReason = "ServerDeletionNotAllowed"
	ReasonServerPodDeleted          EventReason = "ServerPodDeleted"
	ReasonServerPodCreationFailed   EventReason = "ServerPodCreationFailed"
	ReasonServerUpdateFailed        EventReason = "ServerUpdateFailed"

	// Fleet reasons
	ReasonFleetInitialized    EventReason = "FleetInitialized"
	ReasonFleetUpdateFailed   EventReason = "FleetUpdateFailed"
	ReasonFleetServersRemoved EventReason = "FleetServersRemoved"
	ReasonFleetScaledUp       EventReason = "FleetScaledUp"
	ReasonFleetScaledDown     EventReason = "FleetScaledDown"
	ReasonFleetReady          EventReason = "FleetReady"
	ReasonFleetNotReady       EventReason = "FleetNotReady"

	// GameType reasons
	ReasonGametypeFinalizerAdded        EventReason = "GametypeFinalizerAdded"
	ReasonGametypeFinalizerRemoved      EventReason = "GametypeFinalizerRemoved"
	ReasonGametypeFleetCreated          EventReason = "GametypeFleetCreated"
	ReasonGametypeFleetCreationFailed   EventReason = "GametypeFleetCreationFailed"
	ReasonGametypeFleetDeletionFailed   EventReason = "GametypeFleetDeletionFailed"
	ReasonGametypeReplicasUpdated       EventReason = "GametypeReplicasUpdated"
	ReasonGametypeRollingUpdateStarted  EventReason = "GametypeRollingUpdateStarted"
	ReasonGametypeRollingUpdateComplete EventReason = "GametypeRollingUpdateComplete"
	ReasonGametypeReady                 EventReason = "GametypeReady"
	ReasonGameTypeDeleting              EventReason = "GameTypeDeleting"
	ReasonGametypeFleetsDeleted         EventReason = "GametypeFleetsDeleted"

	// GameTypeAutoscaler reasons
	ReasonGameTypeAutoscalerInitialized            EventReason = "GameAutoscalerInitialized"
	ReasonGameTypeAutoscalerInvalidTarget          EventReason = "GameAutoscalerInvalidTarget"
	ReasonGameTypeAutoscalerInvalidAutoscalePolicy EventReason = "GameautoscalerInvalidAutoscalePolicy"
	ReasonGameTypeAutoscalerInvalidSyncType        EventReason = "GameautoscalerInvalidSyncType"
	ReasonGameTypeAutoscalerWebhook                EventReason = "GameautoscalerWebhook"
	ReasonGameTypeAutoscalerScale                  EventReason = "GameautoscalerScale"
)
