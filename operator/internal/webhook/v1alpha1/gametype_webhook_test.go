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

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	// TODO (user): Add any additional imports if needed
)

var _ = Describe("GameType Webhook", func() {
	var (
		obj       *gameserverv1alpha1.GameType
		oldObj    *gameserverv1alpha1.GameType
		validator GameTypeCustomValidator
		defaulter GameTypeCustomDefaulter
	)

	BeforeEach(func() {
		obj = &gameserverv1alpha1.GameType{}
		oldObj = &gameserverv1alpha1.GameType{}
		validator = GameTypeCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = GameTypeCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating GameType under Defaulting Webhook", func() {
		// TODO (user): Add logic for defaulting webhooks
		// Example:
		// It("Should apply defaults when a required field is empty", func() {
		//     By("simulating a scenario where defaults should be applied")
		//     obj.SomeFieldWithDefault = ""
		//     By("calling the Default method to apply defaults")
		//     defaulter.Default(ctx, obj)
		//     By("checking that the default values are set")
		//     Expect(obj.SomeFieldWithDefault).To(Equal("default_value"))
		// })
	})

	Context("When creating or updating GameType under Validating Webhook", func() {
		It("rejects a GameType where minReplicas exceeds maxReplicas", func() {
			minReplicas, maxReplicas := int32(10), int32(5)
			obj.Spec.FleetSpec.Scaling.MinReplicas = &minReplicas
			obj.Spec.FleetSpec.Scaling.MaxReplicas = &maxReplicas
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("admits a GameType where minReplicas equals maxReplicas", func() {
			bound := int32(5)
			obj.Spec.FleetSpec.Scaling.MinReplicas = &bound
			obj.Spec.FleetSpec.Scaling.MaxReplicas = &bound
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("admits a GameType where minReplicas is less than maxReplicas", func() {
			minReplicas, maxReplicas := int32(2), int32(10)
			obj.Spec.FleetSpec.Scaling.MinReplicas = &minReplicas
			obj.Spec.FleetSpec.Scaling.MaxReplicas = &maxReplicas
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects an update that sets minReplicas above maxReplicas", func() {
			minReplicas, maxReplicas := int32(20), int32(10)
			obj.Spec.FleetSpec.Scaling.MinReplicas = &minReplicas
			obj.Spec.FleetSpec.Scaling.MaxReplicas = &maxReplicas
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})
	})

})
