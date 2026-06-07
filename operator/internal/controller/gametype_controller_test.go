package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
)

var _ = Describe("GameType Controller", func() {
	const ns = "default"

	newGameTypeReconciler := func() *GameTypeReconciler {
		return &GameTypeReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: NewFakeRecorder(),
		}
	}

	reconcileGameType := func(name string) error {
		_, err := newGameTypeReconciler().Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
		})
		return err
	}

	fleetsForGameType := func(gameTypeName string) []gameserverv1alpha1.Fleet {
		list := &gameserverv1alpha1.FleetList{}
		_ = k8sClient.List(context.Background(), list,
			client.InNamespace(ns),
			client.MatchingLabels{"gametype": gameTypeName},
		)
		return list.Items
	}

	cleanupGameType := func(name string) {
		// Remove fleets first
		for _, f := range fleetsForGameType(name) {
			f := f
			clearFinalizers(&f)
			_ = k8sClient.Delete(context.Background(), &f)
		}
		gt := &gameserverv1alpha1.GameType{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, gt); err != nil {
			return
		}
		clearFinalizers(gt)
		_ = k8sClient.Delete(context.Background(), gt)
	}

	Context("Finalizer management", func() {
		const gtName = "gt-fin-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gtName, ns, 0))).To(Succeed())
		})

		AfterEach(func() { cleanupGameType(gtName) })

		It("adds the gametype finalizer on first reconcile", func() {
			Expect(reconcileGameType(gtName)).To(Succeed())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gtName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Finalizers).To(ContainElement(TypeFinalizer))
		})
	})

	Context("Fleet creation", func() {
		const gtName = "gt-fleet-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gtName, ns, 1))).To(Succeed())
			Expect(reconcileGameType(gtName)).To(Succeed()) // adds finalizer
		})

		AfterEach(func() { cleanupGameType(gtName) })

		It("creates a fleet for the gametype after reconcile", func() {
			Expect(reconcileGameType(gtName)).To(Succeed())

			Eventually(func() int {
				return len(fleetsForGameType(gtName))
			}).Should(BeNumerically(">=", 1))
		})

		It("labels the fleet with the gametype name", func() {
			Expect(reconcileGameType(gtName)).To(Succeed())

			fleets := fleetsForGameType(gtName)
			Expect(fleets).NotTo(BeEmpty())
			Expect(fleets[0].Labels).To(HaveKeyWithValue("gametype", gtName))
		})

		It("sets the gametype as the fleet owner", func() {
			Expect(reconcileGameType(gtName)).To(Succeed())

			fleets := fleetsForGameType(gtName)
			Expect(fleets).NotTo(BeEmpty())
			Expect(fleets[0].OwnerReferences).NotTo(BeEmpty())
			Expect(fleets[0].OwnerReferences[0].Name).To(Equal(gtName))
		})

		It("records the fleet name in the gametype status", func() {
			Expect(reconcileGameType(gtName)).To(Succeed()) // create fleet
			Expect(reconcileGameType(gtName)).To(Succeed()) // update status

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gtName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Status.CurrentFleetName).NotTo(BeEmpty())
		})
	})

	Context("Replica updates", func() {
		const gtName = "gt-replica-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gtName, ns, 1))).To(Succeed())
			Expect(reconcileGameType(gtName)).To(Succeed()) // finalizer
			Expect(reconcileGameType(gtName)).To(Succeed()) // create fleet
			Expect(reconcileGameType(gtName)).To(Succeed()) // set status
		})

		AfterEach(func() { cleanupGameType(gtName) })

		It("updates the underlying fleet replicas when spec changes", func() {
			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gtName, Namespace: ns}, gt)).To(Succeed())
			gt.Spec.FleetSpec.Scaling.Replicas = 3
			Expect(k8sClient.Update(context.Background(), gt)).To(Succeed())

			Expect(reconcileGameType(gtName)).To(Succeed())

			fleets := fleetsForGameType(gtName)
			Expect(fleets).NotTo(BeEmpty())
			Expect(fleets[0].Spec.Scaling.Replicas).To(Equal(int32(3)))
		})
	})

	Context("Deletion handling", func() {
		const gtName = "gt-del-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gtName, ns, 1))).To(Succeed())
			Expect(reconcileGameType(gtName)).To(Succeed()) // finalizer
			Expect(reconcileGameType(gtName)).To(Succeed()) // create fleet
		})

		AfterEach(func() { cleanupGameType(gtName) })

		It("deletes fleets and removes finalizer when gametype is deleted", func() {
			Expect(fleetsForGameType(gtName)).NotTo(BeEmpty())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gtName, Namespace: ns}, gt)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), gt)).To(Succeed())

			Expect(reconcileGameType(gtName)).To(Succeed())

			Expect(fleetsForGameType(gtName)).To(BeEmpty())

			updated := &gameserverv1alpha1.GameType{}
			if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: gtName, Namespace: ns}, updated); err == nil {
				Expect(updated.Finalizers).NotTo(ContainElement(TypeFinalizer))
			}
		})
	})
})
