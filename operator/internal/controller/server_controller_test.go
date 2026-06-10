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

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/internal/utils"
)

var _ = Describe("Server Controller", func() {
	const ns = "default"

	newServerReconciler := func(deletion FakeDeletion) *ServerReconciler {
		return &ServerReconciler{
			Client:          k8sClient,
			Scheme:          k8sClient.Scheme(),
			Recorder:        NewFakeRecorder(),
			DeletionAllowed: deletion,
		}
	}

	reconcileServer := func(name string, deletion FakeDeletion) error {
		_, err := newServerReconciler(deletion).Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
		})
		return err
	}

	getPod := func(serverName string) (*corev1.Pod, error) {
		pod := &corev1.Pod{}
		err := k8sClient.Get(context.Background(), types.NamespacedName{
			Name:      serverName + "-pod",
			Namespace: ns,
		}, pod)
		return pod, err
	}

	cleanupServer := func(name string) {
		pod := &corev1.Pod{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name + "-pod", Namespace: ns}, pod); err == nil {
			clearFinalizers(pod)
			_ = k8sClient.Delete(context.Background(), pod)
		}
		server := &gameserverv1alpha1.Server{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, server); err == nil {
			clearFinalizers(server)
			_ = k8sClient.Delete(context.Background(), server)
		}
	}

	allowed := FakeDeletion{Allow: true}
	blocked := FakeDeletion{Allow: false}

	Context("Finalizer management", func() {
		const serverName = "server-fin-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("adds the server finalizer on first reconcile", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())
			Expect(server.Finalizers).To(ContainElement(ServerFinalizer))
		})
	})

	Context("Pod lifecycle", func() {
		const serverName = "server-pod-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // add finalizer
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("creates a pod named <server>-pod", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal(serverName + "-pod"))
		})

		It("sets the server label on the pod", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Labels).To(HaveKeyWithValue("server", serverName))
		})

		It("adds the finalizer to the pod on a subsequent reconcile", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // create pod
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // add pod finalizer

			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Finalizers).To(ContainElement(ServerFinalizer))
		})

		It("sets the sidecar container on the pod", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())

			containerNames := make([]string, len(pod.Spec.Containers))
			for i, c := range pod.Spec.Containers {
				containerNames[i] = c.Name
			}
			Expect(containerNames).To(ContainElement("fallernetes-sidecar"))
		})
	})

	Context("Deletion flow", func() {
		const serverName = "server-del-test"

		setupServer := func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // create pod
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // pod finalizer
		}

		AfterEach(func() { cleanupServer(serverName) })

		It("keeps the server alive when deletion is not allowed by sidecar", func() {
			setupServer()

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), server)).To(Succeed())

			// Reconcile: deletion blocked
			Expect(reconcileServer(serverName, blocked)).To(Succeed())

			// Server still present (finalizer not removed)
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())
			Expect(server.Finalizers).To(ContainElement(ServerFinalizer))
		})

		It("removes the server when deletion is allowed by sidecar", func() {
			setupServer()

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), server)).To(Succeed())

			// handleDeletion succeeds: pod deleted, server finalizer removed
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			// Server finalizer should be stripped so GC can remove it
			updated := &gameserverv1alpha1.Server{}
			if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, updated); err == nil {
				Expect(updated.Finalizers).NotTo(ContainElement(ServerFinalizer))
			}
		})

		It("deletes the pod when deletion is allowed", func() {
			setupServer()

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), server)).To(Succeed())

			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			// Pod should be gone (or at least have no finalizer blocking deletion)
			pod, err := getPod(serverName)
			if err == nil {
				Expect(pod.Finalizers).NotTo(ContainElement(ServerFinalizer))
			}
		})
	})

	Context("Sidecar environment variables", func() {
		const serverName = "server-env-test"

		BeforeEach(func() {
			capacity := 10
			server := makeServer(serverName, ns)
			server.Spec.GameInfo = &gameserverv1alpha1.GameInfo{Capacity: &capacity}
			Expect(k8sClient.Create(context.Background(), server)).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // add finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // create pod
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("injects PORT, SERVER_NAME, and SERVER_CAPACITY into the sidecar container", func() {
			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())

			var sidecar *corev1.Container
			for i := range pod.Spec.Containers {
				if pod.Spec.Containers[i].Name == "fallernetes-sidecar" {
					sidecar = &pod.Spec.Containers[i]
					break
				}
			}
			Expect(sidecar).NotTo(BeNil(), "sidecar container not found in pod spec")

			envMap := make(map[string]string)
			for _, e := range sidecar.Env {
				envMap[e.Name] = e.Value
			}

			Expect(envMap).To(HaveKey("PORT"))
			Expect(envMap).To(HaveKey("SERVER_NAME"))
			Expect(envMap["SERVER_NAME"]).To(Equal(serverName))
			Expect(envMap).To(HaveKey("SERVER_CAPACITY"))
		})
	})

	Context("Idempotency", func() {
		const serverName = "server-idempotent-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // create pod
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("does not create a second pod when reconciled again", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			podList := &corev1.PodList{}
			Expect(k8sClient.List(context.Background(), podList,
				client.InNamespace(ns),
				client.MatchingLabels{"server": serverName},
			)).To(Succeed())
			Expect(podList.Items).To(HaveLen(1))
		})
	})

	Context("Owner reference", func() {
		const serverName = "server-ownerref-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // create pod
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("sets the Server as the pod's owner reference", func() {
			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.OwnerReferences).NotTo(BeEmpty())

			owner := pod.OwnerReferences[0]
			Expect(owner.Name).To(Equal(serverName))
			Expect(owner.Kind).To(Equal("Server"))
		})
	})

	Context("Self-healing", func() {
		const serverName = "server-selfheal-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // create pod
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // pod finalizer
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("recreates the pod after it is manually deleted", func() {
			// Confirm pod exists
			pod, err := getPod(serverName)
			Expect(err).NotTo(HaveOccurred())

			// Simulate external deletion: strip finalizer first so the delete goes through
			clearFinalizers(pod)
			Expect(k8sClient.Delete(context.Background(), pod)).To(Succeed())

			// Pod should be gone
			_, err = getPod(serverName)
			Expect(err).To(HaveOccurred())

			// Reconcile detects missing pod and recreates it
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			_, err = getPod(serverName)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Status and conditions", func() {
		const serverName = "server-status-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // adds server finalizer
		})

		AfterEach(func() {
			cleanupServer(serverName)
			// Wait for the pod to be fully removed from etcd before the next test starts.
			// Without this, a pod left in a brief "terminating" state (DeletionTimestamp set,
			// no finalizers) causes the next test's ensurePodFinalizer to fail with
			// "no new finalizers can be added if the object is being deleted".
			Eventually(func() bool {
				p := &corev1.Pod{}
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName + "-pod", Namespace: ns}, p)
				return err != nil && client.IgnoreNotFound(err) == nil
			}, "10s", "200ms").Should(BeTrue())
		})

		It("sets Phase=Pending and initial conditions (Unknown) right after pod creation", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // creates pod, writes pending status

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())

			Expect(server.Status.Phase).To(Equal(gameserverv1alpha1.ServerPhasePending))
			Expect(server.Status.ObservedGeneration).To(Equal(server.Generation))

			readyCond := findCond(server.Status.Conditions, gameserverv1alpha1.ConditionReady)
			Expect(readyCond).NotTo(BeNil())
			Expect(readyCond.Status).To(Equal(metav1.ConditionUnknown))
			Expect(readyCond.Reason).To(Equal(gameserverv1alpha1.ReasonPodNotRunning))

			scheduledCond := findCond(server.Status.Conditions, gameserverv1alpha1.ConditionPodScheduled)
			Expect(scheduledCond).NotTo(BeNil())
			Expect(scheduledCond.Status).To(Equal(metav1.ConditionUnknown))
			Expect(scheduledCond.Reason).To(Equal(gameserverv1alpha1.ReasonPodPending))
		})

		It("reflects non-running pod state after full status sync", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // creates pod
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // adds pod finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // calls syncServerStatus

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())

			Expect(server.Status.Phase).To(Equal(gameserverv1alpha1.ServerPhasePending))

			// Pod has no NodeName in envtest, so PodScheduled=False
			scheduledCond := findCond(server.Status.Conditions, gameserverv1alpha1.ConditionPodScheduled)
			Expect(scheduledCond).NotTo(BeNil())
			Expect(scheduledCond.Status).To(Equal(metav1.ConditionFalse))
			Expect(scheduledCond.Reason).To(Equal(gameserverv1alpha1.ReasonPodNotScheduled))

			// Pod is not Running, so Ready=False
			readyCond := findCond(server.Status.Conditions, gameserverv1alpha1.ConditionReady)
			Expect(readyCond).NotTo(BeNil())
			Expect(readyCond.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCond.Reason).To(Equal(gameserverv1alpha1.ReasonPodNotRunning))
		})

		It("sets Phase=Ready and Ready=True when pod is Running with a NodeName", func() {
			// Pre-create the pod with NodeName set. Kubernetes forbids patching spec.nodeName on
			// an existing pod, but allows it to be set at creation time.
			// TerminationGracePeriodSeconds must be 0: envtest has no kubelet, so a pod bound to
			// a node with the default 30s grace period would get stuck in Terminating on cleanup.
			gracePeriod := int64(0)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serverName + "-pod",
					Namespace: ns,
					Labels:    map[string]string{"server": serverName},
				},
				Spec: corev1.PodSpec{
					NodeName:                      "fake-node",
					TerminationGracePeriodSeconds: &gracePeriod,
					Containers:                    []corev1.Container{{Name: "game-server", Image: "test:latest"}},
				},
			}
			Expect(k8sClient.Create(context.Background(), pod)).To(Succeed())

			// Reconcile: ensurePodExists finds the pre-created pod, ensurePodFinalizer adds finalizer
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			// Simulate kubelet marking the pod as Running
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName + "-pod", Namespace: ns}, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(context.Background(), pod)).To(Succeed())

			// Reconcile: pod finalizer already present, calls syncServerStatus
			Expect(reconcileServer(serverName, allowed)).To(Succeed())

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())

			Expect(server.Status.Phase).To(Equal(gameserverv1alpha1.ServerPhaseReady))
			Expect(server.Status.PodPhase).To(Equal(corev1.PodRunning))

			readyCond := findCond(server.Status.Conditions, gameserverv1alpha1.ConditionReady)
			Expect(readyCond).NotTo(BeNil())
			Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCond.Reason).To(Equal(gameserverv1alpha1.ReasonPodRunning))

			scheduledCond := findCond(server.Status.Conditions, gameserverv1alpha1.ConditionPodScheduled)
			Expect(scheduledCond).NotTo(BeNil())
			Expect(scheduledCond.Status).To(Equal(metav1.ConditionTrue))
			Expect(scheduledCond.Reason).To(Equal(gameserverv1alpha1.ReasonPodScheduled))
		})

		It("reflects the pod's current phase in PodPhase", func() {
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // creates pod
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // adds pod finalizer

			pod := &corev1.Pod{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName + "-pod", Namespace: ns}, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodPending
			Expect(k8sClient.Status().Update(context.Background(), pod)).To(Succeed())

			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // calls syncServerStatus

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: serverName, Namespace: ns}, server)).To(Succeed())
			Expect(server.Status.PodPhase).To(Equal(corev1.PodPending))
		})
	})

	Context("Events", func() {
		newRecordingReconciler := func() (*ServerReconciler, *FakeRecorder) {
			rec := NewFakeRecorder()
			return &ServerReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				Recorder:        rec,
				DeletionAllowed: allowed,
			}, rec
		}

		do := func(r *ServerReconciler, name string) {
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())
		}

		It("emits ServerFinalizerAdded on first reconcile", func() {
			const name = "server-ev-fin-add"
			Expect(k8sClient.Create(context.Background(), makeServer(name, ns))).To(Succeed())
			defer cleanupServer(name)

			r, rec := newRecordingReconciler()
			do(r, name)
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonServerFinalizerAdded)))
		})

		It("emits ServerPodCreated when the pod is first created", func() {
			const name = "server-ev-pod-create"
			Expect(k8sClient.Create(context.Background(), makeServer(name, ns))).To(Succeed())
			defer cleanupServer(name)

			Expect(reconcileServer(name, allowed)).To(Succeed()) // adds server finalizer

			r, rec := newRecordingReconciler()
			do(r, name)
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonServerPodCreated)))
		})

		It("emits ServerPodFinalizerAdded when the pod finalizer is applied", func() {
			const name = "server-ev-pod-fin"
			Expect(k8sClient.Create(context.Background(), makeServer(name, ns))).To(Succeed())
			defer cleanupServer(name)

			Expect(reconcileServer(name, allowed)).To(Succeed()) // server finalizer
			Expect(reconcileServer(name, allowed)).To(Succeed()) // pod created

			r, rec := newRecordingReconciler()
			do(r, name)
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonServerPodFinalizerAdded)))
		})

		It("emits ServerReady when pod transitions from Pending to Running", func() {
			const name = "server-ev-ready"
			Expect(k8sClient.Create(context.Background(), makeServer(name, ns))).To(Succeed())
			defer cleanupServer(name)

			Expect(reconcileServer(name, allowed)).To(Succeed()) // server finalizer
			Expect(reconcileServer(name, allowed)).To(Succeed()) // pod created, phase=Pending
			Expect(reconcileServer(name, allowed)).To(Succeed()) // pod finalizer

			pod := &corev1.Pod{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: name + "-pod", Namespace: ns}, pod)).To(Succeed())
			pod.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(context.Background(), pod)).To(Succeed())

			r, rec := newRecordingReconciler()
			do(r, name)
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonServerReady)))
		})

		It("emits ServerPodFinalizerRemoved and ServerFinalizerRemoved during deletion", func() {
			const name = "server-ev-del"
			Expect(k8sClient.Create(context.Background(), makeServer(name, ns))).To(Succeed())
			defer cleanupServer(name)

			Expect(reconcileServer(name, allowed)).To(Succeed()) // server finalizer
			Expect(reconcileServer(name, allowed)).To(Succeed()) // pod created
			Expect(reconcileServer(name, allowed)).To(Succeed()) // pod finalizer

			server := &gameserverv1alpha1.Server{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, server)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), server)).To(Succeed())

			r, rec := newRecordingReconciler()
			do(r, name)
			reasons := eventReasons(rec)
			Expect(reasons).To(ContainElement(string(utils.ReasonServerPodFinalizerRemoved)))
			Expect(reasons).To(ContainElement(string(utils.ReasonServerFinalizerRemoved)))
		})
	})

	Context("Error paths", func() {
		const serverName = "server-err-test"

		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeServer(serverName, ns))).To(Succeed())
			Expect(reconcileServer(serverName, allowed)).To(Succeed()) // finalizer
		})

		AfterEach(func() { cleanupServer(serverName) })

		It("returns an error when pod creation fails", func() {
			failReconciler := &ServerReconciler{
				Client:          FakeFailClient{Client: k8sClient, FailCreate: true},
				Scheme:          k8sClient.Scheme(),
				Recorder:        NewFakeRecorder(),
				DeletionAllowed: allowed,
			}
			_, err := failReconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: serverName, Namespace: ns},
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
