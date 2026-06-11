package utils

import (
	"context"
	"fmt"

	"github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/operator/internal/sidecar"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FindDeleteServer is used to find the server that should be deleted.
// It is based on the specs agepriority field.
func FindDeleteServer(ctx context.Context, fleet *v1alpha1.Fleet, servers *v1alpha1.ServerList, client client.Client, checker sidecar.FleetDeletionChecker) (*v1alpha1.Server, error) {
	strategy := fleet.Spec.Scaling.AgePriority
	deleteFirst := fleet.Spec.Scaling.PrioritizeAllowed

	if strategy == v1alpha1.OldestFirst {
		return getOldestServer(ctx, servers, deleteFirst, &client, checker)
	}

	if strategy == v1alpha1.NewestFirst {
		return getNewestServer(ctx, servers, deleteFirst, &client, checker)

	}
	return nil, fmt.Errorf("invalid scaling strategy: %s", strategy)
}

// getOldestServer gets the server of the fleet, that is the oldest
// If deleteFirst is enabled, then it tries to get the server that can be deleted, but that is the oldest out of those.
// If it cannot find any where deletion is allowed, it returns the oldest server.
func getOldestServer(ctx context.Context, servers *v1alpha1.ServerList, deleteFirst bool, client *client.Client, checker sidecar.FleetDeletionChecker) (*v1alpha1.Server, error) {
	var oldestServer *v1alpha1.Server
	var oldestTime *metav1.Time
	var oldestAllowedServer *v1alpha1.Server
	var oldestAllowTime *metav1.Time

	for i := range servers.Items {
		server := &servers.Items[i]
		if oldestTime == nil || server.CreationTimestamp.Before(oldestTime) {
			oldestTime = &server.CreationTimestamp
			oldestServer = server
		}
		if deleteFirst {
			allowed, err := checker.IsDeleteAllowed(ctx, server, client)
			if err != nil {
				return nil, err
			}
			if allowed {
				if oldestAllowTime == nil || server.CreationTimestamp.Before(oldestAllowTime) {
					oldestAllowTime = &server.CreationTimestamp
					oldestAllowedServer = server
				}
			}
		}
	}

	if oldestServer == nil {
		return nil, fmt.Errorf("no servers found")
	}

	if oldestAllowedServer != nil {
		return oldestAllowedServer, nil
	}

	return oldestServer, nil
}

// getNewestServer gets the server of the fleet, that is the newest
// If deleteFirst is enabled, then it tries to get the server that can be deleted, but that is the youngest out of those.
// If it cannot find any where deletion is allowed, it returns the youngest server.
func getNewestServer(ctx context.Context, servers *v1alpha1.ServerList, deleteFirst bool, client *client.Client, checker sidecar.FleetDeletionChecker) (*v1alpha1.Server, error) {

	var newestServer *v1alpha1.Server
	var newestTime *metav1.Time
	var newestAllowedServer *v1alpha1.Server
	var newestAllowTime *metav1.Time

	for i := range servers.Items {
		server := &servers.Items[i]
		if newestTime == nil || server.CreationTimestamp.After(newestTime.Time) {
			newestTime = &server.CreationTimestamp
			newestServer = server
		}
		if deleteFirst {
			allowed, err := checker.IsDeleteAllowed(ctx, server, client)
			if err != nil {
				return nil, err
			}
			if allowed {
				if newestAllowTime == nil || server.CreationTimestamp.After(newestAllowTime.Time) {
					newestAllowTime = &server.CreationTimestamp
					newestAllowedServer = server
				}
			}
		}
	}

	if newestServer == nil {
		return nil, fmt.Errorf("no servers found")
	}

	if newestAllowedServer != nil {
		return newestAllowedServer, nil
	}

	return newestServer, nil
}
