package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
)

var _ = Describe("GameTypeAutoscaler Controller", func() {
	const ns = "default"
	const autoscalerName = "autoscaler-test"

	webhookPath := "/scale"
	interval := metav1.Duration{Duration: 30 * time.Second}

	validAutoscaler := func() *gameserverv1alpha1.GameTypeAutoscaler {
		return &gameserverv1alpha1.GameTypeAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: autoscalerName, Namespace: ns},
			Spec: gameserverv1alpha1.GameTypeAutoscalerSpec{
				GameTypeName: "my-gametype",
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

	BeforeEach(func() {
		Expect(k8sClient.Create(context.Background(), validAutoscaler())).To(Succeed())
	})

	AfterEach(func() {
		resource := &gameserverv1alpha1.GameTypeAutoscaler{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: autoscalerName, Namespace: ns}, resource); err == nil {
			_ = k8sClient.Delete(context.Background(), resource)
		}
	})

	It("reconciles without error (stub controller)", func() {
		reconciler := &GameTypeAutoscalerReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Name: autoscalerName, Namespace: ns},
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
