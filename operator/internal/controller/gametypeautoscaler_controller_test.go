package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/MirrorStudios/fallernetes-operator/internal/autoscaler"
	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
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

	newReconciler := func(webhook autoscaler.Webhook) *GameTypeAutoscalerReconciler {
		return &GameTypeAutoscalerReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: NewFakeRecorder(),
			Webhook:  webhook,
		}
	}

	reconcileAutoscaler := func(webhook autoscaler.Webhook) (reconcile.Result, error) {
		return newReconciler(webhook).Reconcile(context.Background(), reconcile.Request{
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

	It("requeues without error when the referenced GameType does not exist", func() {
		result, err := reconcileAutoscaler(FakeWebhook{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Requeue).To(BeTrue())
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
			result, err := reconcileAutoscaler(FakeWebhook{Err: fmt.Errorf("webhook unavailable")})
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))
		})
	})
})
