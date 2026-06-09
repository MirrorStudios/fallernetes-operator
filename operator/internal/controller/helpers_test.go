package controller

import (
	"context"
	"fmt"
	"sync"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	"github.com/MirrorStudios/fallernetes-operator/internal/sidecar"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func defaultPodSpec() corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "game-server", Image: "test-game:latest"},
		},
	}
}

// FakeFailClient embeds a real client and selectively fails specific operations.
// Using embedding means it automatically satisfies any future interface additions.
type FakeFailClient struct {
	client.Client
	FailUpdate   bool
	FailCreate   bool
	FailDelete   bool
	FailGet      bool
	FailList     bool
	FailPatch    bool
	FailGetOnPod bool
}

func (c FakeFailClient) Get(ctx context.Context, name types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if c.FailGet {
		return fmt.Errorf("fail get")
	}
	if c.FailGetOnPod {
		if _, ok := obj.(*corev1.Pod); ok {
			return fmt.Errorf("fail get on pod")
		}
	}
	return c.Client.Get(ctx, name, obj, opts...)
}

func (c FakeFailClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.FailCreate {
		return fmt.Errorf("fail create")
	}
	return c.Client.Create(ctx, obj, opts...)
}

func (c FakeFailClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.FailUpdate {
		return fmt.Errorf("fail update")
	}
	return c.Client.Update(ctx, obj, opts...)
}

func (c FakeFailClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.FailDelete {
		return fmt.Errorf("fail delete")
	}
	return c.Client.Delete(ctx, obj, opts...)
}

func (c FakeFailClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	if c.FailDelete {
		return fmt.Errorf("fail deleteAllOf")
	}
	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

func (c FakeFailClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	if c.FailList {
		return fmt.Errorf("fail list")
	}
	return c.Client.List(ctx, obj, opts...)
}

func (c FakeFailClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.FailPatch {
		return fmt.Errorf("fail patch")
	}
	return c.Client.Patch(ctx, obj, patch, opts...)
}

// FakeEvent holds data captured by FakeRecorder.
type FakeEvent struct {
	Object    runtime.Object
	EventType string
	Reason    string
	Message   string
}

// FakeRecorder is a thread-safe in-memory event recorder for use in tests.
type FakeRecorder struct {
	mu     sync.Mutex
	Events []FakeEvent
}

func NewFakeRecorder() *FakeRecorder {
	return &FakeRecorder{}
}

func (f *FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Events = append(f.Events, FakeEvent{Object: object, EventType: eventtype, Reason: reason, Message: message})
}

func (f *FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	f.Event(object, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (f *FakeRecorder) AnnotatedEventf(object runtime.Object, _ map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Events = append(f.Events, FakeEvent{Object: object, EventType: eventtype, Reason: reason, Message: fmt.Sprintf(messageFmt, args...)})
}

// FakeDeletion implements utils.Deletion for testing server deletion logic.
type FakeDeletion struct {
	Allow bool
	Err   error
}

func (f FakeDeletion) IsDeletionAllowed(_ *gameserverv1alpha1.Server, _ *corev1.Pod) (bool, error) {
	return f.Allow, f.Err
}

var _ sidecar.Deletion = FakeDeletion{}

// MapDeletion implements utils.FleetDeletionChecker with per-server control.
type MapDeletion struct {
	Allow map[string]bool
}

func (m MapDeletion) IsDeleteAllowed(_ context.Context, server *gameserverv1alpha1.Server, _ *client.Client) (bool, error) {
	return m.Allow[server.Name], nil
}

var _ utils.FleetDeletionChecker = MapDeletion{}

// Resource factory helpers

func defaultPort() *int {
	p := 8080
	return &p
}

func defaultImage() *string {
	img := "test-image:latest"
	return &img
}

func defaultSidecarSettings() *gameserverv1alpha1.SidecarSettings {
	return &gameserverv1alpha1.SidecarSettings{
		Port:         defaultPort(),
		SidecarImage: defaultImage(),
	}
}

func makeFleet(name, ns string, replicas int32) *gameserverv1alpha1.Fleet {
	return &gameserverv1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: gameserverv1alpha1.FleetSpec{
			Scaling: gameserverv1alpha1.FleetScaling{
				Replicas:          replicas,
				PrioritizeAllowed: false,
				AgePriority:       gameserverv1alpha1.OldestFirst,
			},
			ServerSpec: gameserverv1alpha1.ServerSpec{
				Pod:             defaultPodSpec(),
				SidecarSettings: defaultSidecarSettings(),
			},
		},
	}
}

func makeServer(name, ns string) *gameserverv1alpha1.Server {
	return &gameserverv1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: gameserverv1alpha1.ServerSpec{
			Pod:             defaultPodSpec(),
			SidecarSettings: defaultSidecarSettings(),
		},
	}
}

func makeGameType(name, ns string, replicas int32) *gameserverv1alpha1.GameType {
	return &gameserverv1alpha1.GameType{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: gameserverv1alpha1.GameTypeSpec{
			FleetSpec: gameserverv1alpha1.FleetSpec{
				Scaling: gameserverv1alpha1.FleetScaling{
					Replicas:          replicas,
					PrioritizeAllowed: false,
					AgePriority:       gameserverv1alpha1.OldestFirst,
				},
				ServerSpec: gameserverv1alpha1.ServerSpec{
					Pod:             defaultPodSpec(),
					SidecarSettings: defaultSidecarSettings(),
				},
			},
		},
	}
}

// clearFinalizers removes all finalizers from obj, silently ignoring missing objects.
func clearFinalizers(obj client.Object) {
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		return
	}
	patch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
	obj.SetFinalizers(nil)
	_ = k8sClient.Patch(ctx, obj, patch)
}
