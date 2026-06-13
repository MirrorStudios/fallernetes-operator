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

func fleetTests() {
	It("should create Running pods for a Fleet", func() {
		const name = "e2e-fleet-test"
		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "fleet", name, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		By("applying a Fleet manifest")
		tmpFile := writeFleetManifest(name, testNamespace, 2)
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for 2 pods to appear with the fleet label")
		Eventually(waitForFleetPods(name, testNamespace, 2), 2*time.Minute, 5*time.Second).Should(Succeed())

		By("verifying all pods reach Running phase")
		Eventually(waitForFleetPodsRunning(name, testNamespace), 3*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("should remove all Servers and Pods when a Fleet is deleted", func() {
		const name = "e2e-fleet-del-test"
		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "fleet", name, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		By("creating a Fleet with 2 replicas")
		tmpFile := writeFleetManifest(name, testNamespace, 2)
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for 2 pods to be Running")
		Eventually(waitForFleetPods(name, testNamespace, 2), 2*time.Minute, 5*time.Second).Should(Succeed())
		Eventually(waitForFleetPodsRunning(name, testNamespace), 3*time.Minute, 5*time.Second).Should(Succeed())

		By("deleting the Fleet")
		cmd = exec.Command("kubectl", "delete", "fleet", name, "-n", testNamespace, "--wait=false")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for all Pods with the fleet label to disappear")
		Eventually(waitForFleetPods(name, testNamespace, 0), 3*time.Minute, 5*time.Second).Should(Succeed())

		By("waiting for all Server CRs with the fleet label to disappear")
		Eventually(waitForFleetServers(name, testNamespace, 0), 3*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("should remove excess pods when a Fleet is scaled down", func() {
		const name = "e2e-fleet-scaledown-test"
		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "fleet", name, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		By("applying a Fleet with 3 replicas")
		tmpFile := writeFleetManifest(name, testNamespace, 3)
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for 3 pods to appear")
		Eventually(waitForFleetPods(name, testNamespace, 3), 2*time.Minute, 5*time.Second).Should(Succeed())

		By("patching fleet replicas down to 1")
		cmd = exec.Command("kubectl", "patch", "fleet", name,
			"-n", testNamespace,
			"--type=merge",
			"-p", `{"spec":{"scaling":{"replicas":1}}}`,
		)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting until only 1 pod remains")
		Eventually(waitForFleetPods(name, testNamespace, 1), 2*time.Minute, 5*time.Second).Should(Succeed())

		By("waiting until only 1 Server CR remains")
		Eventually(waitForFleetServers(name, testNamespace, 1), 2*time.Minute, 5*time.Second).Should(Succeed())
	})
}
