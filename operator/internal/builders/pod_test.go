package builders

import (
	"os"

	"github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GetNewPod", func() {
	port := 9000
	img := "sidecar:v1"

	baseServer := &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-server",
			Namespace: "default",
			UID:       "server-uid-abc",
		},
		Spec: v1alpha1.ServerSpec{
			SidecarSettings: &v1alpha1.SidecarSettings{
				Port:         &port,
				SidecarImage: &img,
				LogDebug:     true,
			},
		},
	}

	It("names the pod <server>-pod", func() {
		pod := GetNewPod(baseServer, "default")
		Expect(pod.Name).To(Equal("my-server-pod"))
	})

	It("places the pod in the given namespace", func() {
		pod := GetNewPod(baseServer, "other-ns")
		Expect(pod.Namespace).To(Equal("other-ns"))
	})

	It("sets the server label on the pod", func() {
		pod := GetNewPod(baseServer, "default")
		Expect(pod.Labels).To(HaveKeyWithValue("server", "my-server"))
	})

	It("sets an owner reference to the server", func() {
		pod := GetNewPod(baseServer, "default")
		Expect(pod.OwnerReferences).To(HaveLen(1))
		Expect(pod.OwnerReferences[0].Name).To(Equal("my-server"))
		Expect(string(pod.OwnerReferences[0].UID)).To(Equal("server-uid-abc"))
	})

	It("includes the fallernetes-sidecar container", func() {
		pod := GetNewPod(baseServer, "default")
		names := make([]string, len(pod.Spec.Containers))
		for i, c := range pod.Spec.Containers {
			names[i] = c.Name
		}
		Expect(names).To(ContainElement("fallernetes-sidecar"))
	})

	It("sets the sidecar image from SidecarSettings", func() {
		pod := GetNewPod(baseServer, "default")
		var sidecar *struct{ Image string }
		for _, c := range pod.Spec.Containers {
			if c.Name == "fallernetes-sidecar" {
				sidecar = &struct{ Image string }{Image: c.Image}
				break
			}
		}
		Expect(sidecar).NotTo(BeNil())
		Expect(sidecar.Image).To(Equal("sidecar:v1"))
	})

	It("injects PORT env var matching SidecarSettings.Port", func() {
		pod := GetNewPod(baseServer, "default")
		for _, c := range pod.Spec.Containers {
			if c.Name == "fallernetes-sidecar" {
				found := false
				for _, env := range c.Env {
					if env.Name == "PORT" {
						Expect(env.Value).To(Equal("9000"))
						found = true
					}
				}
				Expect(found).To(BeTrue(), "PORT env var not found in sidecar container")
			}
		}
	})

	It("injects SERVER_NAME into all containers", func() {
		pod := GetNewPod(baseServer, "default")
		for _, c := range pod.Spec.Containers {
			found := false
			for _, env := range c.Env {
				if env.Name == "SERVER_NAME" {
					Expect(env.Value).To(Equal("my-server"))
					found = true
				}
			}
			Expect(found).To(BeTrue(), "SERVER_NAME env var missing from container %s", c.Name)
		}
	})

	It("injects FLEET_NAME when the server has a fleet label", func() {
		server := baseServer.DeepCopy()
		server.Labels = map[string]string{"fleet": "my-fleet"}
		pod := GetNewPod(server, "default")

		for _, c := range pod.Spec.Containers {
			found := false
			for _, env := range c.Env {
				if env.Name == "FLEET_NAME" {
					Expect(env.Value).To(Equal("my-fleet"))
					found = true
				}
			}
			Expect(found).To(BeTrue(), "FLEET_NAME env var missing from container %s", c.Name)
		}
	})

	It("injects SERVER_CAPACITY when GameInfo.Capacity is set", func() {
		server := baseServer.DeepCopy()
		cap := 50
		server.Spec.GameInfo = &v1alpha1.GameInfo{Capacity: &cap}
		pod := GetNewPod(server, "default")

		for _, c := range pod.Spec.Containers {
			found := false
			for _, env := range c.Env {
				if env.Name == "SERVER_CAPACITY" {
					Expect(env.Value).To(Equal("50"))
					found = true
				}
			}
			Expect(found).To(BeTrue(), "SERVER_CAPACITY env var missing from container %s", c.Name)
		}
	})

	It("does not inject SERVER_CAPACITY when GameInfo is nil", func() {
		pod := GetNewPod(baseServer, "default")
		for _, c := range pod.Spec.Containers {
			for _, env := range c.Env {
				Expect(env.Name).NotTo(Equal("SERVER_CAPACITY"))
			}
		}
	})

	It("omits ImagePullSecrets when IMAGE_PULL_SECRET_NAME is unset", func() {
		Expect(os.Unsetenv("IMAGE_PULL_SECRET_NAME")).To(Succeed())
		pod := GetNewPod(baseServer, "default")
		Expect(pod.Spec.ImagePullSecrets).To(BeEmpty())
	})

	It("adds ImagePullSecrets when IMAGE_PULL_SECRET_NAME is set", func() {
		Expect(os.Setenv("IMAGE_PULL_SECRET_NAME", "my-pull-secret")).To(Succeed())
		DeferCleanup(os.Unsetenv, "IMAGE_PULL_SECRET_NAME")
		pod := GetNewPod(baseServer, "default")
		Expect(pod.Spec.ImagePullSecrets).To(HaveLen(1))
		Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("my-pull-secret"))
	})
})
