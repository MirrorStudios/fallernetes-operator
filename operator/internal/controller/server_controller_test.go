package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
