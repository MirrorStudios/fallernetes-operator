package builders

import (
	"github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GetFleetObjectForType", func() {
	port := 8080
	img := "test:latest"

	baseGameType := &v1alpha1.GameType{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-gametype",
			Namespace: "default",
			UID:       "gt-uid-5678",
		},
		Spec: v1alpha1.GameTypeSpec{
			FleetSpec: v1alpha1.FleetSpec{
				Scaling: v1alpha1.FleetScaling{Replicas: 2},
				ServerSpec: v1alpha1.ServerSpec{
					SidecarSettings: &v1alpha1.SidecarSettings{
						Port:         &port,
						SidecarImage: &img,
					},
				},
			},
		},
	}

	It("uses gametype name as GenerateName prefix", func() {
		fleet := GetFleetObjectForType(baseGameType)
		Expect(fleet.GenerateName).To(Equal("my-gametype-"))
	})

	It("sets the gametype label on the fleet", func() {
		fleet := GetFleetObjectForType(baseGameType)
		Expect(fleet.Labels).To(HaveKeyWithValue("gametype", "my-gametype"))
	})

	It("preserves existing gametype labels", func() {
		gt := baseGameType.DeepCopy()
		gt.Labels = map[string]string{"env": "prod"}
		fleet := GetFleetObjectForType(gt)
		Expect(fleet.Labels).To(HaveKeyWithValue("env", "prod"))
		Expect(fleet.Labels).To(HaveKeyWithValue("gametype", "my-gametype"))
	})

	It("sets the namespace from the gametype", func() {
		fleet := GetFleetObjectForType(baseGameType)
		Expect(fleet.Namespace).To(Equal("default"))
	})

	It("sets owner reference pointing to the gametype", func() {
		fleet := GetFleetObjectForType(baseGameType)
		Expect(fleet.OwnerReferences).To(HaveLen(1))
		Expect(fleet.OwnerReferences[0].Name).To(Equal("my-gametype"))
		Expect(string(fleet.OwnerReferences[0].UID)).To(Equal("gt-uid-5678"))
		Expect(*fleet.OwnerReferences[0].Controller).To(BeTrue())
	})

	It("copies the fleet spec from the gametype", func() {
		fleet := GetFleetObjectForType(baseGameType)
		Expect(fleet.Spec.Scaling.Replicas).To(Equal(int32(2)))
		Expect(*fleet.Spec.ServerSpec.SidecarSettings.Port).To(Equal(8080))
	})
})
