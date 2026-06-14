package controller

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/autoscaler"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/utils"
)

var _ = Describe("GameTypeAutoscaler Controller", func() {
	const ns = "default"
	const autoscalerName = "autoscaler-test"
	const gametypeName = "autoscaler-gametype"

	webhookPath := "/scale"
	interval := metav1.Duration{Duration: 30 * time.Second}

	validAutoscaler := func() *gameserverv1alpha1.GameTypeAutoscaler {
		return &gameserverv1alpha1.GameTypeAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: autoscalerName, Namespace: ns},
			Spec: gameserverv1alpha1.GameTypeAutoscalerSpec{
				GameTypeName: gametypeName,
				AutoscalePolicy: gameserverv1alpha1.AutoscalePolicy{
					Type: gameserverv1alpha1.Webhook,
					WebhookAutoscalerSpec: gameserverv1alpha1.WebhookAutoscalerSpec{
						Path: &webhookPath,
					},
				},
				Sync: gameserverv1alpha1.Sync{
					Type: gameserverv1alpha1.FixedInterval,
					Time: &interval,
				},
			},
		}
	}

	newReconciler := func(webhook autoscaler.Webhook) (*GameTypeAutoscalerReconciler, *FakeRecorder) {
		rec := NewFakeRecorder()
		return &GameTypeAutoscalerReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: rec,
			Webhook:  webhook,
		}, rec
	}

	reconcileAutoscaler := func(webhook autoscaler.Webhook) (reconcile.Result, error) {
		r, _ := newReconciler(webhook)
		return r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
		})
	}

	BeforeEach(func() {
		Expect(k8sClient.Create(context.Background(), validAutoscaler())).To(Succeed())
	})

	AfterEach(func() {
		gt := &gameserverv1alpha1.GameType{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt); err == nil {
			_ = k8sClient.Delete(context.Background(), gt)
		}
		resource := &gameserverv1alpha1.GameTypeAutoscaler{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: autoscalerName, Namespace: ns}, resource); err == nil {
			_ = k8sClient.Delete(context.Background(), resource)
		}
	})

	It("requeues after 30 seconds without error when the referenced GameType does not exist", func() {
		result, err := reconcileAutoscaler(FakeWebhook{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(30 * time.Second))
	})

	Context("with an existing GameType", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gametypeName, ns, 2))).To(Succeed())
		})

		It("requeues after the sync interval when the webhook says no scale", func() {
			result, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: false}})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(interval.Duration))
		})

		It("updates the GameType replica count when the webhook says scale", func() {
			result, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 5}})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(interval.Duration))

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(5)))
		})

		It("requeues after one minute when the webhook returns an error", func() {
			result, err := reconcileAutoscaler(FakeWebhook{Err: errors.New("webhook unavailable")})
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))
		})
	})

	Context("Behavioral", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gametypeName, ns, 3))).To(Succeed())
		})

		It("passes the autoscaler spec and current gametype to the webhook", func() {
			capture := &FakeWebhookCapture{}
			r, _ := newReconciler(capture)
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(capture.CalledWithAutoscaler).NotTo(BeNil())
			Expect(capture.CalledWithAutoscaler.Spec.GameTypeName).To(Equal(gametypeName))
			Expect(capture.CalledWithGameType).NotTo(BeNil())
			Expect(capture.CalledWithGameType.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(3)))
		})

		It("does not update the GameType when the webhook says no scale is needed", func() {
			r, _ := newReconciler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: false}})
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(3)))
		})

		It("requeues after the sync interval even when scaling occurs", func() {
			r, _ := newReconciler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 5}})
			result, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(interval.Duration))
		})

		It("returns an error when the GameType update fails during scaling", func() {
			reconciler := &GameTypeAutoscalerReconciler{
				Client:   FakeFailClient{Client: k8sClient, FailUpdate: true},
				Scheme:   k8sClient.Scheme(),
				Recorder: NewFakeRecorder(),
				Webhook:  FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 5}},
			}
			_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to update gametype"))
		})
	})

	Context("Clamping", func() {
		ptr32 := func(v int32) *int32 { return &v }

		makeGameTypeWithBounds := func(name, ns string, replicas int32, min, max *int32) *gameserverv1alpha1.GameType {
			gt := makeGameType(name, ns, replicas)
			gt.Spec.FleetSpec.Scaling.MinReplicas = min
			gt.Spec.FleetSpec.Scaling.MaxReplicas = max
			return gt
		}

		It("clamps up to minReplicas when the webhook returns a value below min", func() {
			Expect(k8sClient.Create(context.Background(), makeGameTypeWithBounds(gametypeName, ns, 3, ptr32(5), nil))).To(Succeed())

			_, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 2}})
			Expect(err).NotTo(HaveOccurred())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(5)))
		})

		It("clamps down to maxReplicas when the webhook returns a value above max", func() {
			Expect(k8sClient.Create(context.Background(), makeGameTypeWithBounds(gametypeName, ns, 3, nil, ptr32(10)))).To(Succeed())

			_, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 15}})
			Expect(err).NotTo(HaveOccurred())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(10)))
		})

		It("applies the webhook value as-is when neither bound is set", func() {
			Expect(k8sClient.Create(context.Background(), makeGameType(gametypeName, ns, 3))).To(Succeed())

			_, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 7}})
			Expect(err).NotTo(HaveOccurred())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(7)))
		})

		It("only enforces minReplicas when maxReplicas is not set — values above min pass through", func() {
			Expect(k8sClient.Create(context.Background(), makeGameTypeWithBounds(gametypeName, ns, 3, ptr32(5), nil))).To(Succeed())

			_, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 20}})
			Expect(err).NotTo(HaveOccurred())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(20)))
		})

		It("only enforces maxReplicas when minReplicas is not set — values below max pass through", func() {
			Expect(k8sClient.Create(context.Background(), makeGameTypeWithBounds(gametypeName, ns, 3, nil, ptr32(8)))).To(Succeed())

			_, err := reconcileAutoscaler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 3}})
			Expect(err).NotTo(HaveOccurred())

			gt := &gameserverv1alpha1.GameType{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{Name: gametypeName, Namespace: ns}, gt)).To(Succeed())
			Expect(gt.Spec.FleetSpec.Scaling.Replicas).To(Equal(int32(3)))
		})
	})

	Context("Events", func() {
		It("emits GameTypeAutoscalerInvalidTarget when the GameType does not exist", func() {
			r, rec := newReconciler(FakeWebhook{})
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonGameTypeAutoscalerInvalidTarget)))
		})

		Context("with an existing GameType", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(context.Background(), makeGameType(gametypeName, ns, 2))).To(Succeed())
			})

			It("emits GameAutoscalerWebhook warning when the webhook returns an error", func() {
				r, rec := newReconciler(FakeWebhook{Err: errors.New("webhook down")})
				_, err := r.Reconcile(context.Background(), reconcile.Request{
					NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
				})
				Expect(err).To(HaveOccurred())
				Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonGameTypeAutoscalerWebhook)))
			})

			It("emits GameAutoscalerScale when the webhook requests a scale", func() {
				r, rec := newReconciler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: true, DesiredReplicas: 4}})
				_, err := r.Reconcile(context.Background(), reconcile.Request{
					NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(eventReasons(rec)).To(ContainElement(string(utils.ReasonGameTypeAutoscalerScale)))
			})

			It("emits no scale event when the webhook says no scale is needed", func() {
				r, rec := newReconciler(FakeWebhook{Response: autoscaler.AutoscaleResponse{Scale: false}})
				_, err := r.Reconcile(context.Background(), reconcile.Request{
					NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(eventReasons(rec)).NotTo(ContainElement(string(utils.ReasonGameTypeAutoscalerScale)))
			})
		})
	})
})
