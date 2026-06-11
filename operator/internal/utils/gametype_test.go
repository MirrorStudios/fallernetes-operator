package utils

import (
	"time"

	"github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
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
