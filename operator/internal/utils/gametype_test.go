package utils

import (
	"time"

	"github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeFleetWithAge(name string, offset time.Duration) v1alpha1.Fleet {
	return v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.NewTime(time.Now().Add(offset)),
		},
	}
}

var _ = Describe("GetOldestFleet", func() {
	It("returns nil for an empty slice", func() {
		Expect(GetOldestFleet(nil)).To(BeNil())
	})

	It("returns the only fleet when slice has one element", func() {
		fleets := []v1alpha1.Fleet{makeFleetWithAge("only", 0)}
		Expect(GetOldestFleet(fleets).Name).To(Equal("only"))
	})

	DescribeTable("picks the fleet with the earliest creation timestamp",
		func(names []string, offsets []time.Duration, expectedName string) {
			fleets := make([]v1alpha1.Fleet, len(names))
			for i, name := range names {
				fleets[i] = makeFleetWithAge(name, offsets[i])
			}
			Expect(GetOldestFleet(fleets).Name).To(Equal(expectedName))
		},
		Entry("oldest is first",
			[]string{"old", "new"},
			[]time.Duration{-2 * time.Hour, -1 * time.Hour},
			"old",
		),
		Entry("oldest is last",
			[]string{"new", "old"},
			[]time.Duration{-1 * time.Hour, -2 * time.Hour},
			"old",
		),
		Entry("oldest is in the middle",
			[]string{"mid", "old", "new"},
			[]time.Duration{-1 * time.Hour, -3 * time.Hour, 0},
			"old",
		),
	)
})

var _ = Describe("GetNewestFleet", func() {
	It("returns nil for an empty slice", func() {
		Expect(GetNewestFleet(nil)).To(BeNil())
	})

	It("returns the only fleet when slice has one element", func() {
		fleets := []v1alpha1.Fleet{makeFleetWithAge("only", 0)}
		Expect(GetNewestFleet(fleets).Name).To(Equal("only"))
	})

	DescribeTable("picks the fleet with the latest creation timestamp",
		func(names []string, offsets []time.Duration, expectedName string) {
			fleets := make([]v1alpha1.Fleet, len(names))
			for i, name := range names {
				fleets[i] = makeFleetWithAge(name, offsets[i])
			}
			Expect(GetNewestFleet(fleets).Name).To(Equal(expectedName))
		},
		Entry("newest is first",
			[]string{"new", "old"},
			[]time.Duration{0, -2 * time.Hour},
			"new",
		),
		Entry("newest is last",
			[]string{"old", "new"},
			[]time.Duration{-2 * time.Hour, 0},
			"new",
		),
		Entry("newest is in the middle",
			[]string{"old", "new", "mid"},
			[]time.Duration{-3 * time.Hour, 0, -1 * time.Hour},
			"new",
		),
	)
})

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
