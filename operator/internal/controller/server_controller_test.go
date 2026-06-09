package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
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
