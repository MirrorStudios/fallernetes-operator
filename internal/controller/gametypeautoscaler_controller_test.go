/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"github.com/MirrorStudios/fallernetes/internal/utils"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes/api/v1alpha1"
)

type TestWebhook struct {
	Scale    bool
	Replicas int
	Error    bool
}

func (t *TestWebhook) SendScaleWebhookRequest(autoscaler *gameserverv1alpha1.GameTypeAutoscaler, gametype *gameserverv1alpha1.GameType) (utils.AutoscaleResponse, error) {
	if t.Error {
		return utils.AutoscaleResponse{}, fmt.Errorf("random error with webhook")
	}
	return utils.AutoscaleResponse{
		Scale:           t.Scale,
		DesiredReplicas: t.Replicas,
	}, nil
}

var duration = metav1.Duration{Duration: 5 * time.Second}
var path = "/scale"

const resourceName = "test-resource"
const namespace = "default"

var basicGameTypeAutoscaler = gameserverv1alpha1.GameTypeAutoscalerSpec{
	GameTypeName: resourceName,
	AutoscalePolicy: gameserverv1alpha1.AutoscalePolicy{
		Type: gameserverv1alpha1.Webhook,
		WebhookAutoscalerSpec: gameserverv1alpha1.WebhookAutoscalerSpec{
			Path: &path,
			Service: &gameserverv1alpha1.Service{
				Name:      "some-random-service",
				Namespace: metav1.NamespaceDefault,
				Port:      8080,
			},
		},
	},
	Sync: gameserverv1alpha1.Sync{
		Type: gameserverv1alpha1.FixedInterval,
		Time: &duration,
	},
}

var _ = Describe("GameTypeAutoscaler Controller", func() {

	Context("When reconciling a resource", func() {

		ctx := context.Background()

		autoscalerNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}
		gameTypeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		BeforeEach(func() {
			By("creating a new game to match the autoscaler")
			gametype := &gameserverv1alpha1.GameType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
			}
			GameTypeAutoscaler := &gameserverv1alpha1.GameTypeAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				}}

			err := k8sClient.Get(ctx, gameTypeNamespacedName, gametype)
			if err != nil && errors.IsNotFound(err) {
				resource := &gameserverv1alpha1.GameType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: basicGametypeSpec,
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Eventually(func() error {
					return k8sClient.Get(ctx, gameTypeNamespacedName, &gameserverv1alpha1.GameType{})
				}, time.Second*5, time.Millisecond*100).Should(Succeed())

			}
			By("creating the custom resource for the Kind GameTypeAutoscaler")
			err = k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			if err != nil && errors.IsNotFound(err) {
				resource := &gameserverv1alpha1.GameTypeAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: basicGameTypeAutoscaler,
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

		})
		AfterEach(func() {
			By("Check if GameTypeAutoscaler exists")
			GameTypeAutoscaler := &gameserverv1alpha1.GameTypeAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
			}
			err := k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			Expect(err).To(BeNil())

			By("Cleanup the specific GameTypeAutoscaler instance")
			err = k8sClient.Delete(ctx, GameTypeAutoscaler)
			Expect(err).To(BeNil())

			Eventually(func() error {
				err := k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("error deleting game autoscaler: %w", err)
				}
				return nil
			}, time.Second*5, time.Millisecond*100).Should(Succeed())
			By("Check if autoscaler was deleted")
			err = k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			Expect(err).To(Not(BeNil()))

			By("Check if game exists")
			game := &gameserverv1alpha1.GameType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gameTypeNamespacedName.Name,
					Namespace: namespace,
				},
			}

			err = k8sClient.Get(ctx, gameTypeNamespacedName, game)
			Expect(err).To(BeNil())

			By("Cleanup the specific game instance")
			err = k8sClient.Delete(ctx, game)
			Expect(err).To(BeNil())

			Eventually(func() error {
				err := k8sClient.Get(ctx, gameTypeNamespacedName, game)
				if !errors.IsNotFound(err) {
					return fmt.Errorf("error deleting game: %w", err)
				}
				return nil
			}, time.Second*5, time.Millisecond*100).Should(Succeed())

			By("Check if game was deleted")
			err = k8sClient.Get(ctx, gameTypeNamespacedName, game)
			Expect(err).To(Not(BeNil()))
		})

		It("should successfully reconcile the GameTypeAutoscaler", func() {
			By("create the reconciler for GameTypeAutoscaler")
			hook := &TestWebhook{
				Scale:    false,
				Replicas: 1,
				Error:    false,
			}
			controllerReconciler := &GameTypeAutoscalerReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Webhook:  hook,
				Recorder: NewFakeRecorder(),
			}

			By("first reconciling for GameTypeAutoscaler")
			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).To(BeNil())
			Expect(res.RequeueAfter).To(BeEquivalentTo(5 * time.Second))

			By("second reconciling for GameTypeAutoscaler")
			res, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).To(BeNil())
			Expect(res.RequeueAfter).To(BeEquivalentTo(5 * time.Second))

			By("reconcile not existing autoscaler")
			res, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      "some-random-scaler",
			},
			})

			Expect(err).To(Not(BeNil()))
			Expect(err).To(Not(Succeed()))

			By("Reconcile with webhook error")
			hook.Error = true
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).To(Not(BeNil()))
			hook.Error = false
		})

		It("Reconcile with scaling", func() {
			By("Setup reconciler")
			hook := &TestWebhook{
				Scale:    false,
				Replicas: 1,
				Error:    false,
			}
			controllerReconciler := &GameTypeAutoscalerReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Webhook:  hook,
				Recorder: NewFakeRecorder(),
			}

			By("Setup hook")
			hook.Scale = true
			hook.Replicas = 10

			By("Reconcile and check time")
			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).To(BeNil())
			Expect(res.RequeueAfter).To(BeEquivalentTo(5 * time.Second))
			By("Check updated game")
			updatedGameType := gameserverv1alpha1.GameType{}
			err = k8sClient.Get(ctx, gameTypeNamespacedName, &updatedGameType)
			Expect(err).To(BeNil())
			Expect(updatedGameType.Spec.FleetSpec.Scaling.Replicas).Should(BeEquivalentTo(10))
		})

		It("Reconcile with invalid types", func() {
			By("Setup reconciler")
			hook := &TestWebhook{
				Scale:    false,
				Replicas: 1,
				Error:    false,
			}
			controllerReconciler := &GameTypeAutoscalerReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Webhook:  hook,
				Recorder: NewFakeRecorder(),
			}

			GameTypeAutoscaler := &gameserverv1alpha1.GameTypeAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				}}

			By("Reconcile")
			err := k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			Expect(err).To(BeNil())

			By("Invalid gametype reconcile")
			err = k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			Expect(err).To(BeNil())
			GameTypeAutoscaler.Spec.AutoscalePolicy.Type = gameserverv1alpha1.Webhook
			GameTypeAutoscaler.Spec.GameTypeName = "thisdoesnotexist"
			err = k8sClient.Update(ctx, GameTypeAutoscaler)
			Expect(err).To(BeNil())
			Eventually(func() error {
				autoscaler := gameserverv1alpha1.GameTypeAutoscaler{}
				err := k8sClient.Get(ctx, autoscalerNamespacedName, &autoscaler)
				if err != nil {
					return err
				}
				if autoscaler.Spec.GameTypeName != "thisdoesnotexist" {
					return fmt.Errorf("still correct game")
				}
				return nil
			}, time.Second*5, time.Millisecond*100).Should(Succeed())
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).ToNot(BeNil())
		})

		It("should fail update", func() {
			fakeClient := FakeFailClient{
				client:       k8sClient,
				FailUpdate:   true,
				FailCreate:   false,
				FailDelete:   false,
				FailGet:      false,
				FailList:     false,
				FailPatch:    false,
				FailGetOnPod: false,
			}
			hook := &TestWebhook{
				Scale:    false,
				Replicas: 1,
				Error:    false,
			}
			controllerReconciler := &GameTypeAutoscalerReconciler{
				Client:   fakeClient,
				Scheme:   fakeClient.Scheme(),
				Webhook:  hook,
				Recorder: NewFakeRecorder(),
			}
			hook.Scale = true
			hook.Replicas = 10
			hook.Error = false

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      resourceName,
				},
			})
			Expect(err).To(Not(BeNil()))
		})

		It("Should emit the correct events", func() {
			recorder := NewFakeRecorder()

			GameTypeAutoscaler := &gameserverv1alpha1.GameTypeAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				}}

			hook := &TestWebhook{
				Scale:    false,
				Replicas: 1,
				Error:    false,
			}
			controllerReconciler := &GameTypeAutoscalerReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Webhook:  hook,
				Recorder: recorder,
			}

			originalGametype := resourceName
			By("Update to invalid GameTypeName")
			err := k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			Expect(err).To(BeNil())
			GameTypeAutoscaler.Spec.GameTypeName = originalGametype + "-1"
			err = k8sClient.Update(ctx, GameTypeAutoscaler)
			Expect(err).To(BeNil())

			By("Reconcile after update")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).ToNot(BeNil())
			By("Check for fail to find gametype event")
			hasGametypeErrorEvent := false
			for _, event := range recorder.Events {
				if event.Message == "Failed to find the gametype" {
					hasGametypeErrorEvent = true
					break
				}
			}
			Expect(hasGametypeErrorEvent).To(BeTrue())

			By("Reset game name")
			GameTypeAutoscaler.Spec.GameTypeName = originalGametype
			err = k8sClient.Update(ctx, GameTypeAutoscaler)
			Expect(err).To(BeNil())
			err = k8sClient.Get(ctx, autoscalerNamespacedName, GameTypeAutoscaler)
			Expect(err).To(BeNil())

			By("Check if scale event is emitted")
			hook.Scale = true
			hook.Replicas = 5
			hook.Error = false
			controllerReconciler.Webhook = hook
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: autoscalerNamespacedName})
			Expect(err).To(BeNil())
			hasScaleEvent := false
			for _, event := range recorder.Events {
				if event.Message == "Scaling game to 5" {
					hasScaleEvent = true
					break
				}
			}
			Expect(hasScaleEvent).To(BeTrue())
		})
	})
})
