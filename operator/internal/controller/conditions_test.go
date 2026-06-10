package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
)

var _ = Describe("countReadyServers", func() {
	It("returns 0 for a nil list", func() {
		Expect(countReadyServers(nil)).To(Equal(int32(0)))
	})

	It("returns 0 when no servers are in the Ready phase", func() {
		servers := []gameserverv1alpha1.Server{
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhasePending}},
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhaseDeleting}},
		}
		Expect(countReadyServers(servers)).To(Equal(int32(0)))
	})

	It("counts only servers in the Ready phase", func() {
		servers := []gameserverv1alpha1.Server{
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhaseReady}},
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhasePending}},
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhaseReady}},
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhaseDeleting}},
		}
		Expect(countReadyServers(servers)).To(Equal(int32(2)))
	})

	It("counts all servers when every server is Ready", func() {
		servers := []gameserverv1alpha1.Server{
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhaseReady}},
			{Status: gameserverv1alpha1.ServerStatus{Phase: gameserverv1alpha1.ServerPhaseReady}},
		}
		Expect(countReadyServers(servers)).To(Equal(int32(2)))
	})
})

var _ = Describe("syncFleetConditions", func() {
	r := &FleetReconciler{}

	makeFleetWithStatus := func(replicas, readyReplicas, desiredReplicas int32) *gameserverv1alpha1.Fleet {
		return &gameserverv1alpha1.Fleet{
			Status: gameserverv1alpha1.FleetStatus{
				Replicas:        replicas,
				ReadyReplicas:   readyReplicas,
				DesiredReplicas: desiredReplicas,
			},
		}
	}

	It("sets Ready=True when readyReplicas equals desiredReplicas and desired > 0", func() {
		fleet := makeFleetWithStatus(3, 3, 3)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonAtDesiredCount))
	})

	It("sets Ready=False when readyReplicas is less than desiredReplicas", func() {
		fleet := makeFleetWithStatus(2, 1, 2)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonReplicasMismatch))
	})

	It("sets Ready=False when desiredReplicas is zero", func() {
		fleet := makeFleetWithStatus(0, 0, 0)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
	})

	It("sets Available=True when readyReplicas > 0", func() {
		fleet := makeFleetWithStatus(3, 1, 3)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionAvailable)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonReplicasAvailable))
	})

	It("sets Available=False when readyReplicas is 0", func() {
		fleet := makeFleetWithStatus(2, 0, 2)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionAvailable)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonNoReplicas))
	})

	It("sets Scaling=True with ScalingUp when replicas < desired", func() {
		fleet := makeFleetWithStatus(1, 0, 3)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionScaling)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonScalingUp))
	})

	It("sets Scaling=True with ScalingDown when replicas > desired", func() {
		fleet := makeFleetWithStatus(5, 3, 2)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionScaling)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonScalingDown))
	})

	It("sets Scaling=False when replicas equals desired", func() {
		fleet := makeFleetWithStatus(2, 2, 2)
		r.syncFleetConditions(fleet)

		cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionScaling)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonAtDesiredCount))
	})

	It("always sets all three conditions", func() {
		fleet := makeFleetWithStatus(1, 1, 1)
		r.syncFleetConditions(fleet)

		Expect(findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady)).NotTo(BeNil())
		Expect(findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionAvailable)).NotTo(BeNil())
		Expect(findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionScaling)).NotTo(BeNil())
	})

	It("is idempotent: calling twice does not duplicate conditions", func() {
		fleet := makeFleetWithStatus(2, 1, 3)
		r.syncFleetConditions(fleet)
		r.syncFleetConditions(fleet)

		countByType := map[string]int{}
		for _, c := range fleet.Status.Conditions {
			countByType[c.Type]++
		}
		Expect(countByType[gameserverv1alpha1.ConditionReady]).To(Equal(1))
		Expect(countByType[gameserverv1alpha1.ConditionAvailable]).To(Equal(1))
		Expect(countByType[gameserverv1alpha1.ConditionScaling]).To(Equal(1))
	})
})

var _ = Describe("syncGameTypeConditions", func() {
	r := &GameTypeReconciler{}

	makeGTWithStatus := func(activeFleet string, totalFleets int32, readyReplicas, desired int32) *gameserverv1alpha1.GameType {
		return &gameserverv1alpha1.GameType{
			Spec: gameserverv1alpha1.GameTypeSpec{
				FleetSpec: gameserverv1alpha1.FleetSpec{
					Scaling: gameserverv1alpha1.FleetScaling{Replicas: desired},
				},
			},
			Status: gameserverv1alpha1.GameTypeStatus{
				ActiveFleetName: activeFleet,
				TotalFleets:     totalFleets,
				ReadyReplicas:   readyReplicas,
			},
		}
	}

	It("sets Ready=True when active fleet has all desired replicas ready", func() {
		gt := makeGTWithStatus("fleet-1", 1, 3, 3)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonAtDesiredCount))
	})

	It("sets Ready=False when there is no active fleet", func() {
		gt := makeGTWithStatus("", 0, 0, 2)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonReplicasMismatch))
	})

	It("sets Ready=False when readyReplicas is less than desired", func() {
		gt := makeGTWithStatus("fleet-1", 1, 1, 3)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
	})

	It("sets Ready=False when desired is 0", func() {
		gt := makeGTWithStatus("fleet-1", 1, 0, 0)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
	})

	It("sets RollingUpdate=True when totalFleets > 1", func() {
		gt := makeGTWithStatus("fleet-2", 2, 0, 2)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionRollingUpdate)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonRolloutInProgress))
	})

	It("sets RollingUpdate=False when totalFleets is 1", func() {
		gt := makeGTWithStatus("fleet-1", 1, 2, 2)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionRollingUpdate)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonNoRollout))
	})

	It("sets RollingUpdate=False when there are no fleets", func() {
		gt := makeGTWithStatus("", 0, 0, 2)
		r.syncGameTypeConditions(gt)

		cond := findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionRollingUpdate)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
	})

	It("always sets both Ready and RollingUpdate conditions", func() {
		gt := makeGTWithStatus("fleet-1", 1, 1, 1)
		r.syncGameTypeConditions(gt)

		Expect(findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionReady)).NotTo(BeNil())
		Expect(findCond(gt.Status.Conditions, gameserverv1alpha1.ConditionRollingUpdate)).NotTo(BeNil())
	})
})
