package builders

import (
	"github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CreateServerForFleet", func() {
	port := 8080
	img := "test:latest"

	baseFleet := v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-fleet",
			Namespace: "default",
			UID:       "fleet-uid-1234",
		},
		Spec: v1alpha1.FleetSpec{
			ServerSpec: v1alpha1.ServerSpec{
				SidecarSettings: &v1alpha1.SidecarSettings{
					Port:         &port,
					SidecarImage: &img,
				},
			},
		},
	}

	It("sets the fleet label on the server", func() {
		server := CreateServerForFleet(baseFleet, "default")
		Expect(server.Labels).To(HaveKeyWithValue("fleet", "my-fleet"))
	})

	It("preserves existing fleet labels", func() {
		fleet := baseFleet.DeepCopy()
		fleet.Labels = map[string]string{"existing": "label"}
		server := CreateServerForFleet(*fleet, "default")
		Expect(server.Labels).To(HaveKeyWithValue("existing", "label"))
		Expect(server.Labels).To(HaveKeyWithValue("fleet", "my-fleet"))
	})

	It("uses the fleet name as GenerateName prefix", func() {
		server := CreateServerForFleet(baseFleet, "default")
		Expect(server.GenerateName).To(Equal("my-fleet-"))
	})

	It("sets the namespace from the argument", func() {
		server := CreateServerForFleet(baseFleet, "other-ns")
		Expect(server.Namespace).To(Equal("other-ns"))
	})

	It("sets owner reference pointing to the fleet", func() {
		server := CreateServerForFleet(baseFleet, "default")
		Expect(server.OwnerReferences).To(HaveLen(1))
		Expect(server.OwnerReferences[0].Name).To(Equal("my-fleet"))
		Expect(string(server.OwnerReferences[0].UID)).To(Equal("fleet-uid-1234"))
		Expect(*server.OwnerReferences[0].Controller).To(BeTrue())
	})

	It("copies the fleet server spec into the server", func() {
		server := CreateServerForFleet(baseFleet, "default")
		Expect(server.Spec.SidecarSettings).NotTo(BeNil())
		Expect(*server.Spec.SidecarSettings.Port).To(Equal(8080))
	})
})
