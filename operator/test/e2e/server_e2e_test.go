//go:build e2e
// +build e2e

package e2e

import (
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MirrorStudios/fallernetes-operator/operator/test/utils"
)

func serverTests() {
	It("should create a Running pod for a Server", func() {
		const name = "e2e-server-test"
		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "server", name, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		By("applying a Server manifest")
		tmpFile := writeServerManifest(name, testNamespace, sidecarProtocolCmd)
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		podName := name + "-pod"
		By("waiting for the pod to appear")
		Eventually(waitForPod(podName, testNamespace), 2*time.Minute, 5*time.Second).Should(Succeed())

		By("waiting for the pod to reach Running phase")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pod", podName,
				"-n", testNamespace,
				"-o", "jsonpath={.status.phase}",
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Running"))
		}, 3*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("should inject the sidecar container into the Server pod", func() {
		const name = "e2e-server-sidecar-test"
		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "server", name, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		By("applying a Server manifest")
		tmpFile := writeServerManifest(name, testNamespace, "sleep 3600")
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		podName := name + "-pod"
		By("waiting for the pod to appear")
		Eventually(waitForPod(podName, testNamespace), 2*time.Minute, 5*time.Second).Should(Succeed())

		By("verifying the sidecar container is present in the pod spec")
		cmd = exec.Command("kubectl", "get", "pod", podName,
			"-n", testNamespace,
			"-o", "jsonpath={.spec.containers[*].name}",
		)
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("fallernetes-sidecar"))
	})

	It("should remove the pod when a Server is deleted", func() {
		const name = "e2e-server-del-test"
		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "server", name, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		By("applying a Server manifest")
		tmpFile := writeServerManifest(name, testNamespace, "sleep 3600")
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		podName := name + "-pod"
		By("waiting for the pod to exist")
		Eventually(waitForPod(podName, testNamespace), 2*time.Minute, 5*time.Second).Should(Succeed())

		By("deleting the Server")
		cmd = exec.Command("kubectl", "delete", "server", name, "-n", testNamespace, "--wait=false")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the pod to be gone")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pod", podName, "-n", testNamespace)
			_, err := utils.Run(cmd)
			g.Expect(err).To(HaveOccurred())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})
}
