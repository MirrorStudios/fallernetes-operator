package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/utils"
)

var _ = Describe("Fleet Controller", func() {
	const ns = "default"

	newReconciler := func() *FleetReconciler {
		return &FleetReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: NewFakeRecorder(),
		}
	}

	reconcileFleet := func(name string) error {
		_, err := newReconciler().Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
		})
		return err
	}

	serversForFleet := func(fleetName string) []gameserverv1alpha1.Server {
		list := &gameserverv1alpha1.ServerList{}
		_ = k8sClient.List(context.Background(), list,
			client.InNamespace(ns),
			client.MatchingLabels{"fleet": fleetName},
		)
		return list.Items
	}

	cleanupFleet := func(name string) {
		fleet := &gameserverv1alpha1.Fleet{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, fleet); err != nil {
			return
		}
		clearFinalizers(fleet)
		_ = k8sClient.Delete(context.Background(), fleet)
		for _, s := range serversForFleet(name) {
			s := s
			clearFinalizers(&s)
			_ = k8sClient.Delete(context.Background(), &s)
		}
	}

	Context("Finalizer management", func() {
		const fleetName = "fleet-fin-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 0))).To(Succeed())
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("adds the fleet finalizer on first reconcile", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(fleet.Finalizers).To(ContainElement(FLEET_FINALIZER))
		})

		It("does not add the finalizer twice", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			count := 0
			for _, f := range fleet.Finalizers {
				if f == FLEET_FINALIZER {
					count++
				}
			}
			Expect(count).To(Equal(1))
		})
	})

	Context("Scale up", func() {
		const fleetName = "fleet-scaleup-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 2))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // adds finalizer
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("creates the requested number of servers", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(serversForFleet(fleetName)).To(HaveLen(2))
		})

		It("updates status to reflect current replica count", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(fleet.Status.Replicas).To(Equal(int32(2)))
		})

		It("each created server has the fleet label", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())

			for _, s := range serversForFleet(fleetName) {
				Expect(s.Labels).To(HaveKeyWithValue("fleet", fleetName))
			}
		})
	})

	Context("Scale-up idempotency", func() {
		const fleetName = "fleet-idempotent-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 2))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer
			Expect(reconcileFleet(fleetName)).To(Succeed()) // scale to 2
			Expect(serversForFleet(fleetName)).To(HaveLen(2))
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("does not create extra Servers when reconciled repeatedly at the same replica count", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())

			Expect(serversForFleet(fleetName)).To(HaveLen(2))
		})
	})

	Context("Scale down", func() {
		const fleetName = "fleet-scaledown-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 2))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer
			Expect(reconcileFleet(fleetName)).To(Succeed()) // scale to 2
			Expect(serversForFleet(fleetName)).To(HaveLen(2))
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("deletes one server when replicas decrease by one", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			fleet.Spec.Scaling.Replicas = 1
			Expect(k8sClient.Update(context.Background(), fleet)).To(Succeed())

			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(serversForFleet(fleetName)).To(HaveLen(1))
		})

		It("deletes all servers when scaled to zero", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			fleet.Spec.Scaling.Replicas = 0
			Expect(k8sClient.Update(context.Background(), fleet)).To(Succeed())

			// Each reconcile removes one server at a time (FindDeleteServer picks one)
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(serversForFleet(fleetName)).To(BeEmpty())
		})
	})

	Context("Deletion handling", func() {
		const fleetName = "fleet-del-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 1))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer
			Expect(reconcileFleet(fleetName)).To(Succeed()) // scale to 1
			Expect(serversForFleet(fleetName)).To(HaveLen(1))
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("removes servers and finalizer when fleet is deleted", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), fleet)).To(Succeed())

			Expect(reconcileFleet(fleetName)).To(Succeed())

			Expect(serversForFleet(fleetName)).To(BeEmpty())

			// Fleet finalizer should be gone (fleet may be GC'd or finalizer stripped)
			updated := &gameserverv1alpha1.Fleet{}
			if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, updated); err == nil {
				Expect(updated.Finalizers).NotTo(ContainElement(FLEET_FINALIZER))
			}
		})
	})

	Context("Oldest-first deletion strategy", func() {
		const fleetName = "fleet-oldest-test"

		BeforeEach(func() {
			fleet := makeFleet(fleetName, ns, 0)
			fleet.Spec.Scaling.AgePriority = gameserverv1alpha1.OldestFirst
			Expect(k8sClient.Create(context.Background(), fleet)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer

			for _, name := range []string{"alpha", "beta", "gamma"} {
				s := makeServer(fleetName+"-"+name, ns)
				s.Labels = map[string]string{"fleet": fleetName}
				Expect(k8sClient.Create(context.Background(), s)).To(Succeed())
			}

			fleet2 := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet2)).To(Succeed())
			fleet2.Spec.Scaling.Replicas = 3
			Expect(k8sClient.Update(context.Background(), fleet2)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // status sync
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("deletes the server with the earliest CreationTimestamp", func() {
			servers := serversForFleet(fleetName)
			Expect(servers).To(HaveLen(3))
			oldest := servers[0]
			for _, s := range servers[1:] {
				if s.CreationTimestamp.Before(&oldest.CreationTimestamp) {
					oldest = s
				}
			}

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			fleet.Spec.Scaling.Replicas = 2
			Expect(k8sClient.Update(context.Background(), fleet)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())

			remaining := serversForFleet(fleetName)
			Expect(remaining).To(HaveLen(2))
			remainingNames := make([]string, len(remaining))
			for i, s := range remaining {
				remainingNames[i] = s.Name
			}
			Expect(remainingNames).NotTo(ContainElement(oldest.Name))
		})
	})

	Context("Full scale-up to three replicas", func() {
		const fleetName = "fleet-scale-up-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 3))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("creates three Server CRs", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(serversForFleet(fleetName)).To(HaveLen(3))
		})

		It("results in three Pods after server reconciles", func() {
			Expect(reconcileFleet(fleetName)).To(Succeed()) // scale to 3

			serverList := &gameserverv1alpha1.ServerList{}
			Expect(k8sClient.List(context.Background(), serverList,
				client.InNamespace(ns),
				client.MatchingLabels{"fleet": fleetName},
			)).To(Succeed())

			serverReconciler := &ServerReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				Recorder:        NewFakeRecorder(),
				DeletionAllowed: FakeDeletion{Allow: true},
			}
			for _, s := range serverList.Items {
				req := reconcile.Request{NamespacedName: types.NamespacedName{Name: s.Name, Namespace: ns}}
				_, _ = serverReconciler.Reconcile(context.Background(), req) // finalizer
				_, _ = serverReconciler.Reconcile(context.Background(), req) // pod
			}

			podList := &corev1.PodList{}
			Expect(k8sClient.List(context.Background(), podList,
				client.InNamespace(ns),
				client.MatchingLabels{"fleet": fleetName},
			)).To(Succeed())
			Expect(podList.Items).To(HaveLen(3))
		})
	})

	Context("Scale down from 3 to 1", func() {
		const fleetName = "fleet-scale-down-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 3))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer
			Expect(reconcileFleet(fleetName)).To(Succeed()) // scale to 3
			Expect(serversForFleet(fleetName)).To(HaveLen(3))
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("leaves one Server after patching replicas to 1", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			patch := client.MergeFrom(fleet.DeepCopy())
			fleet.Spec.Scaling.Replicas = 1
			Expect(k8sClient.Patch(context.Background(), fleet, patch)).To(Succeed())

			// Each reconcile removes one server; two deletions needed
			Expect(reconcileFleet(fleetName)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())

			serverList := &gameserverv1alpha1.ServerList{}
			Expect(k8sClient.List(context.Background(), serverList,
				client.InNamespace(ns),
				client.MatchingLabels{"fleet": fleetName},
			)).To(Succeed())
			Expect(serverList.Items).To(HaveLen(1))
		})
	})

	Context("Newest-first deletion strategy", func() {
		const fleetName = "fleet-newest-test"

		BeforeEach(func() {
			fleet := makeFleet(fleetName, ns, 0)
			fleet.Spec.Scaling.AgePriority = gameserverv1alpha1.NewestFirst
			Expect(k8sClient.Create(context.Background(), fleet)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer

			for _, name := range []string{"alpha", "beta", "gamma"} {
				s := makeServer(fleetName+"-"+name, ns)
				s.Labels = map[string]string{"fleet": fleetName}
				Expect(k8sClient.Create(context.Background(), s)).To(Succeed())
			}

			fleet2 := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet2)).To(Succeed())
			fleet2.Spec.Scaling.Replicas = 3
			Expect(k8sClient.Update(context.Background(), fleet2)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("deletes the server with the latest CreationTimestamp", func() {
			servers := serversForFleet(fleetName)
			Expect(servers).To(HaveLen(3))
			newest := servers[0]
			for _, s := range servers[1:] {
				if newest.CreationTimestamp.Before(&s.CreationTimestamp) {
					newest = s
				}
			}

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			fleet.Spec.Scaling.Replicas = 2
			Expect(k8sClient.Update(context.Background(), fleet)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())

			remaining := serversForFleet(fleetName)
			Expect(remaining).To(HaveLen(2))
			remainingNames := make([]string, len(remaining))
			for i, s := range remaining {
				remainingNames[i] = s.Name
			}
			Expect(remainingNames).NotTo(ContainElement(newest.Name))
		})
	})

	Context("Status fields and conditions", func() {
		const fleetName = "fleet-status-cond-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 2))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // adds finalizer
			Expect(reconcileFleet(fleetName)).To(Succeed()) // scales to 2 and syncs status
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("sets DesiredReplicas to the spec value", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(fleet.Status.DesiredReplicas).To(Equal(int32(2)))
		})

		It("sets ObservedGeneration", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(fleet.Status.ObservedGeneration).To(Equal(fleet.Generation))
		})

		It("sets all three conditions after reconcile", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady)).NotTo(BeNil())
			Expect(findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionAvailable)).NotTo(BeNil())
			Expect(findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionScaling)).NotTo(BeNil())
		})

		It("sets Scaling=False when replica count matches desired", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionScaling)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonAtDesiredCount))
		})

		It("sets Ready=False when no servers are in the Ready phase", func() {
			// In envtest, pods never transition to Running on their own, so ReadyReplicas=0.
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		})

		It("sets Available=False when no servers are in the Ready phase", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			cond := findCond(fleet.Status.Conditions, gameserverv1alpha1.ConditionAvailable)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal(gameserverv1alpha1.ReasonNoReplicas))
		})

		It("updates DesiredReplicas after spec change", func() {
			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			patch := client.MergeFrom(fleet.DeepCopy())
			fleet.Spec.Scaling.Replicas = 5
			Expect(k8sClient.Patch(context.Background(), fleet, patch)).To(Succeed())

			Expect(reconcileFleet(fleetName)).To(Succeed())

			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			Expect(fleet.Status.DesiredReplicas).To(Equal(int32(5)))
		})
	})

	Context("PrioritizeAllowed=true", func() {
		const fleetName = "fleet-priority-test"

		BeforeEach(func() {
			fleet := makeFleet(fleetName, ns, 0)
			fleet.Spec.Scaling.PrioritizeAllowed = true
			fleet.Spec.Scaling.AgePriority = gameserverv1alpha1.OldestFirst
			Expect(k8sClient.Create(context.Background(), fleet)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // finalizer

			for _, name := range []string{"old-server", "mid-server", "new-allowed"} {
				s := makeServer(fleetName+"-"+name, ns)
				s.Labels = map[string]string{"fleet": fleetName}
				Expect(k8sClient.Create(context.Background(), s)).To(Succeed())
			}

			fleet2 := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet2)).To(Succeed())
			fleet2.Spec.Scaling.Replicas = 3
			Expect(k8sClient.Update(context.Background(), fleet2)).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed())
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("deletes the deletion-allowed server even though it is the newest", func() {
			allowedName := fleetName + "-new-allowed"
			deletion := MapDeletion{Allow: map[string]bool{allowedName: true}}

			reconciler := &FleetReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				Recorder:        NewFakeRecorder(),
				DeletionChecker: deletion,
			}

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: fleetName, Namespace: ns}, fleet)).To(Succeed())
			fleet.Spec.Scaling.Replicas = 2
			Expect(k8sClient.Update(context.Background(), fleet)).To(Succeed())

			_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: fleetName, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())

			remaining := serversForFleet(fleetName)
			Expect(remaining).To(HaveLen(2))
			remainingNames := make([]string, len(remaining))
			for i, s := range remaining {
				remainingNames[i] = s.Name
			}
			Expect(remainingNames).NotTo(ContainElement(allowedName))
		})
	})

	Context("Events", func() {
		newRecordingReconciler := func() (*FleetReconciler, *FakeRecorder) {
			rec := NewFakeRecorder()
			return &FleetReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: rec,
			}, rec
		}

		do := func(r *FleetReconciler, name string) {
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())
		}

		It("emits FleetScaledUp when servers are created to meet desired replicas", func() {
			const name = "fleet-ev-scaleup"
			Expect(k8sClient.Create(context.Background(), makeFleet(name, ns, 2))).To(Succeed())
			defer cleanupFleet(name)

			Expect(reconcileFleet(name)).To(Succeed()) // finalizer

			r, rec := newRecordingReconciler()
			do(r, name)
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonFleetScaledUp)))
		})

		It("emits FleetScaledDown when a server is removed to meet desired replicas", func() {
			const name = "fleet-ev-scaledown"
			Expect(k8sClient.Create(context.Background(), makeFleet(name, ns, 2))).To(Succeed())
			defer cleanupFleet(name)

			Expect(reconcileFleet(name)).To(Succeed()) // finalizer
			Expect(reconcileFleet(name)).To(Succeed()) // scale to 2
			Expect(serversForFleet(name)).To(HaveLen(2))

			fleet := &gameserverv1alpha1.Fleet{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, fleet)).To(Succeed())
			fleet.Spec.Scaling.Replicas = 1
			Expect(k8sClient.Update(context.Background(), fleet)).To(Succeed())

			r, rec := newRecordingReconciler()
			do(r, name)
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonFleetScaledDown)))
		})

		It("emits FleetNotReady on the first reconcile where the Ready condition is set to False", func() {
			const name = "fleet-ev-notready"
			Expect(k8sClient.Create(context.Background(), makeFleet(name, ns, 1))).To(Succeed())
			defer cleanupFleet(name)

			Expect(reconcileFleet(name)).To(Succeed()) // finalizer (no conditions set yet)

			r, rec := newRecordingReconciler()
			do(r, name) // scales and syncs conditions; no ready replicas → FleetNotReady
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonFleetNotReady)))
		})

		It("emits FleetReady when all replicas become ready", func() {
			const name = "fleet-ev-ready"
			Expect(k8sClient.Create(context.Background(), makeFleet(name, ns, 1))).To(Succeed())
			defer cleanupFleet(name)

			Expect(reconcileFleet(name)).To(Succeed()) // finalizer
			Expect(reconcileFleet(name)).To(Succeed()) // scale to 1, Ready=False stored in status

			servers := serversForFleet(name)
			Expect(servers).To(HaveLen(1))
			server := servers[0].DeepCopy()
			server.Status.Phase = gameserverv1alpha1.ServerPhaseReady
			Expect(k8sClient.Status().Update(context.Background(), server)).To(Succeed())

			r, rec := newRecordingReconciler()
			do(r, name) // ReadyReplicas=1=Desired, Ready transitions False→True
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonFleetReady)))
		})
	})

	Context("Error paths", func() {
		const fleetName = "fleet-err-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeFleet(fleetName, ns, 2))).To(Succeed())
			Expect(reconcileFleet(fleetName)).To(Succeed()) // adds finalizer
		})

		AfterEach(func() { cleanupFleet(fleetName) })

		It("returns an error when server creation fails during scale-up", func() {
			failReconciler := &FleetReconciler{
				Client:   FakeFailClient{Client: k8sClient, FailCreate: true},
				Scheme:   k8sClient.Scheme(),
				Recorder: NewFakeRecorder(),
			}
			_, err := failReconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: fleetName, Namespace: ns},
			})
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when listing servers fails", func() {
			failReconciler := &FleetReconciler{
				Client:   FakeFailClient{Client: k8sClient, FailList: true},
				Scheme:   k8sClient.Scheme(),
				Recorder: NewFakeRecorder(),
			}
			_, err := failReconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: fleetName, Namespace: ns},
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
