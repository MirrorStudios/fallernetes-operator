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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
)

var _ = Describe("Fleet Webhook", func() {
	var (
		obj       *gameserverv1alpha1.Fleet
		oldObj    *gameserverv1alpha1.Fleet
		validator FleetCustomValidator
		defaulter FleetCustomDefaulter
	)

	BeforeEach(func() {
		obj = &gameserverv1alpha1.Fleet{}
		oldObj = &gameserverv1alpha1.Fleet{}
		validator = FleetCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		defaulter = FleetCustomDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating Fleet under Defaulting Webhook", func() {
		It("sets TimeOut to 40 minutes when nil", func() {
			obj.Spec.ServerSpec.TimeOut = nil
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.ServerSpec.TimeOut).NotTo(BeNil())
			Expect(obj.Spec.ServerSpec.TimeOut.Duration).To(Equal(time.Minute * 40))
		})

		It("does not override an already-set TimeOut", func() {
			obj.Spec.ServerSpec.TimeOut = &metav1.Duration{Duration: time.Minute * 10}
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.ServerSpec.TimeOut.Duration).To(Equal(time.Minute * 10))
		})
	})

	Context("When creating or updating Fleet under Validating Webhook", func() {
		It("rejects a Fleet with negative replicas", func() {
			obj.Spec.Scaling.Replicas = -1
			obj.Spec.Scaling.AgePriority = gameserverv1alpha1.OldestFirst
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("rejects a Fleet with an invalid AgePriority", func() {
			obj.Spec.Scaling.Replicas = 1
			obj.Spec.Scaling.AgePriority = "random_value"
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
		})

		It("admits a valid Fleet", func() {
			obj.Spec.Scaling.Replicas = 2
			obj.Spec.Scaling.AgePriority = gameserverv1alpha1.NewestFirst
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects an update that sets replicas to a negative value", func() {
			obj.Spec.Scaling.Replicas = -5
			obj.Spec.Scaling.AgePriority = gameserverv1alpha1.OldestFirst
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
		})
	})

})
