/*
Copyright 2026.

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

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
)

var _ = Describe("Server Webhook", func() {
	var (
		obj       *gameserverv1alpha1.Server
		oldObj    *gameserverv1alpha1.Server
		validator ServerCustomValidator
		defaulter ServerCustomDefaulter
	)

	BeforeEach(func() {
		obj = &gameserverv1alpha1.Server{}
		oldObj = &gameserverv1alpha1.Server{}
		validator = ServerCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = ServerCustomDefaulter{DefaultSidecarImage: "unfamousthomas/fallernetes-sidecar:main"}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating Server under Defaulting Webhook", func() {
		It("sets SidecarSettings defaults when the field is nil", func() {
			obj.Spec.SidecarSettings = nil
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.SidecarSettings).NotTo(BeNil())
			Expect(*obj.Spec.SidecarSettings.Port).To(Equal(8080))
			Expect(*obj.Spec.SidecarSettings.SidecarImage).To(Equal("unfamousthomas/fallernetes-sidecar:main"))
		})

		It("does not override an already-set Port", func() {
			port := 9090
			obj.Spec.SidecarSettings = &gameserverv1alpha1.SidecarSettings{Port: &port}
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(*obj.Spec.SidecarSettings.Port).To(Equal(9090))
			Expect(*obj.Spec.SidecarSettings.SidecarImage).NotTo(BeEmpty())
		})

		It("does not override fully populated SidecarSettings", func() {
			port := 7777
			img := "my-custom-sidecar:v1"
			obj.Spec.SidecarSettings = &gameserverv1alpha1.SidecarSettings{Port: &port, SidecarImage: &img}
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(*obj.Spec.SidecarSettings.Port).To(Equal(7777))
			Expect(*obj.Spec.SidecarSettings.SidecarImage).To(Equal("my-custom-sidecar:v1"))
		})
	})

	Context("When creating or updating Server under Validating Webhook", func() {
		It("rejects a Server with sidecar port 0", func() {
			port := 0
			obj.Spec.SidecarSettings = &gameserverv1alpha1.SidecarSettings{Port: &port}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("rejects a Server with sidecar port above 65535", func() {
			port := 99999
			obj.Spec.SidecarSettings = &gameserverv1alpha1.SidecarSettings{Port: &port}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("admits a Server with no SidecarSettings", func() {
			obj.Spec.SidecarSettings = nil
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("admits a Server with a valid sidecar port", func() {
			port := 8080
			obj.Spec.SidecarSettings = &gameserverv1alpha1.SidecarSettings{Port: &port}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects an update that changes the pod spec", func() {
			oldObj.Spec.Pod = corev1.PodSpec{
				Containers: []corev1.Container{{Name: "game", Image: "game-server:v1"}},
			}
			obj.Spec.Pod = corev1.PodSpec{
				Containers: []corev1.Container{{Name: "game", Image: "game-server:v2"}},
			}
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("admits an update that leaves the pod spec unchanged", func() {
			pod := corev1.PodSpec{
				Containers: []corev1.Container{{Name: "game", Image: "game-server:v1"}},
			}
			oldObj.Spec.Pod = pod
			obj.Spec.Pod = pod
			obj.Spec.AllowForceDelete = true
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
