package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/sidecar"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeFleetDeleteChecker struct {
	DeletionState map[string]bool
}

func (f FakeFleetDeleteChecker) IsDeleteAllowed(_ context.Context, server *v1alpha1.Server, _ *client.Client) (bool, error) {
	return f.DeletionState[server.Name], nil
}

var _ sidecar.FleetDeletionChecker = FakeFleetDeleteChecker{}

func makeTestServers(offsets ...time.Duration) *v1alpha1.ServerList {
	base := time.Now()
	items := make([]v1alpha1.Server, len(offsets))
	for i, d := range offsets {
		items[i] = v1alpha1.Server{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("server%d", i+1),
				CreationTimestamp: metav1.Time{Time: base.Add(d)},
			},
		}
	}
	return &v1alpha1.ServerList{Items: items}
}

var _ = Describe("Fleet Deletion Utilities", func() {
	ctx := context.Background()

	Describe("getOldestServer", func() {
		DescribeTable("selects correct server without prioritization",
			func(offsets []time.Duration, expectedIndex int) {
				servers := makeTestServers(offsets...)
				server, err := getOldestServer(ctx, servers, false, nil, FakeFleetDeleteChecker{})
				Expect(err).NotTo(HaveOccurred())
				Expect(server.Name).To(Equal(servers.Items[expectedIndex].Name))
			},
			Entry("oldest of three is first created",
				[]time.Duration{0, time.Hour, time.Minute}, 0),
			Entry("oldest of two is first created",
				[]time.Duration{time.Hour, 0}, 1),
			Entry("single server is returned",
				[]time.Duration{0}, 0),
		)

		DescribeTable("prioritizes deletable servers",
			func(offsets []time.Duration, deletable map[string]bool, expectedName string) {
				servers := makeTestServers(offsets...)
				for i := range servers.Items {
					servers.Items[i].Name = fmt.Sprintf("server%d", i+1)
				}
				checker := FakeFleetDeleteChecker{DeletionState: deletable}
				server, err := getOldestServer(ctx, servers, true, nil, checker)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.Name).To(Equal(expectedName))
			},
			Entry("returns oldest deletable when some are not deletable",
				[]time.Duration{0, time.Minute, time.Hour},
				map[string]bool{"server2": true, "server3": true},
				"server2",
			),
			Entry("falls back to overall oldest when none are deletable",
				[]time.Duration{0, time.Minute, time.Hour},
				map[string]bool{},
				"server1",
			),
			Entry("returns the only deletable server regardless of age",
				[]time.Duration{0, time.Minute, time.Hour},
				map[string]bool{"server3": true},
				"server3",
			),
		)

		It("returns an error when the server list is empty", func() {
			_, err := getOldestServer(ctx, &v1alpha1.ServerList{}, false, nil, FakeFleetDeleteChecker{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("getNewestServer", func() {
		DescribeTable("selects correct server without prioritization",
			func(offsets []time.Duration, expectedIndex int) {
				servers := makeTestServers(offsets...)
				server, err := getNewestServer(ctx, servers, false, nil, FakeFleetDeleteChecker{})
				Expect(err).NotTo(HaveOccurred())
				Expect(server.Name).To(Equal(servers.Items[expectedIndex].Name))
			},
			Entry("newest of three is last created",
				[]time.Duration{0, time.Hour, time.Minute}, 1),
			Entry("newest of two is last created",
				[]time.Duration{0, time.Hour}, 1),
			Entry("single server is returned",
				[]time.Duration{0}, 0),
		)

		DescribeTable("prioritizes deletable servers",
			func(offsets []time.Duration, deletable map[string]bool, expectedName string) {
				servers := makeTestServers(offsets...)
				for i := range servers.Items {
					servers.Items[i].Name = fmt.Sprintf("server%d", i+1)
				}
				checker := FakeFleetDeleteChecker{DeletionState: deletable}
				server, err := getNewestServer(ctx, servers, true, nil, checker)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.Name).To(Equal(expectedName))
			},
			Entry("returns newest deletable when some are not deletable",
				[]time.Duration{0, time.Minute, time.Hour},
				map[string]bool{"server2": true, "server3": true},
				"server3",
			),
			Entry("falls back to overall newest when none are deletable",
				[]time.Duration{0, time.Minute, time.Hour},
				map[string]bool{},
				"server3",
			),
			Entry("returns the only deletable server regardless of age",
				[]time.Duration{0, time.Minute, time.Hour},
				map[string]bool{"server1": true},
				"server1",
			),
		)

		It("returns an error when the server list is empty", func() {
			_, err := getNewestServer(ctx, &v1alpha1.ServerList{}, false, nil, FakeFleetDeleteChecker{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FindDeleteServer", func() {
		makeFleet := func(priority v1alpha1.Priority, prioritizeAllowed bool) *v1alpha1.Fleet {
			return &v1alpha1.Fleet{
				Spec: v1alpha1.FleetSpec{
					Scaling: v1alpha1.FleetScaling{
						AgePriority:       priority,
						PrioritizeAllowed: prioritizeAllowed,
					},
				},
			}
		}

		It("uses oldest-first strategy when configured", func() {
			servers := makeTestServers(0, time.Hour, time.Minute)
			fleet := makeFleet(v1alpha1.OldestFirst, false)
			server, err := FindDeleteServer(ctx, fleet, servers, nil, FakeFleetDeleteChecker{})
			Expect(err).NotTo(HaveOccurred())
			Expect(server.Name).To(Equal(servers.Items[0].Name))
		})

		It("uses newest-first strategy when configured", func() {
			servers := makeTestServers(0, time.Hour, time.Minute)
			fleet := makeFleet(v1alpha1.NewestFirst, false)
			server, err := FindDeleteServer(ctx, fleet, servers, nil, FakeFleetDeleteChecker{})
			Expect(err).NotTo(HaveOccurred())
			Expect(server.Name).To(Equal(servers.Items[1].Name))
		})

		It("returns an error for an unknown strategy", func() {
			servers := makeTestServers(0)
			fleet := makeFleet(v1alpha1.Priority("unknown"), false)
			_, err := FindDeleteServer(ctx, fleet, servers, nil, FakeFleetDeleteChecker{})
			Expect(err).To(HaveOccurred())
		})
	})
})
