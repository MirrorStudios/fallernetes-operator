//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MirrorStudios/fallernetes-operator/operator/test/utils"
)

func gameTypeTests() {
	It("should create a Fleet for a GameType", func() {
		const name = "e2e-gametype-test"
		cleanupGameType(name, testNamespace)

		By("applying a GameType manifest")
		tmpFile := writeGameTypeManifest(name, testNamespace, "busybox:latest", sidecarProtocolCmd)
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for a Fleet labeled with the gametype name to appear")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "fleets",
				"-l", fmt.Sprintf("gametype=%s", name),
				"-n", testNamespace,
				"-o", "jsonpath={.items[*].metadata.name}",
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(utils.GetNonEmptyLines(output)).NotTo(BeEmpty())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("verifying the GameType status records the fleet name")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "gametype", name,
				"-n", testNamespace,
				"-o", "jsonpath={.status.activeFleetName}",
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).NotTo(BeEmpty())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("should create a new Fleet when GameType pod spec changes", func() {
		const name = "e2e-gametype-rolling-test"
		cleanupGameType(name, testNamespace)

		By("applying initial GameType")
		tmpFile := writeGameTypeManifest(name, testNamespace, "busybox:1.35", "sleep 3600")
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the initial Fleet to appear")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "fleets",
				"-l", fmt.Sprintf("gametype=%s", name),
				"-n", testNamespace,
				"-o", "jsonpath={.items[*].metadata.name}",
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(utils.GetNonEmptyLines(output)).To(HaveLen(1))
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("patching the GameType to change the container image")
		cmd = exec.Command("kubectl", "patch", "gametype", name,
			"-n", testNamespace, "--type=merge",
			"-p", `{"spec":{"fleetSpec":{"spec":{"pod":{"containers":[{"name":"game-server","image":"busybox:1.36","command":["sh","-c","sleep 3600"]}]}}}}}`,
		)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the rolling update to produce a fleet with the new image")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "fleets",
				"-l", fmt.Sprintf("gametype=%s", name),
				"-n", testNamespace,
				"-o", "jsonpath={.items[*].spec.spec.pod.containers[*].image}",
			)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("busybox:1.36"))
		}, 3*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("should accept and reconcile a GameTypeAutoscaler without validation errors", func() {
		const autoscalerName = "e2e-autoscaler-test"
		const gameTypeName = "e2e-autoscaler-gametype"

		DeferCleanup(func() {
			cmd := exec.Command("kubectl", "delete", "gametypeautoscaler", autoscalerName, "-n", testNamespace,
				"--wait=false", "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})
		cleanupGameType(gameTypeName, testNamespace)

		By("creating a GameType for the autoscaler to target")
		tmpFile := writeGameTypeManifest(gameTypeName, testNamespace, "busybox:latest", sidecarProtocolCmd)
		cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("applying a GameTypeAutoscaler manifest")
		// The URL is unreachable; webhook call failures are expected and not tested here.
		// This test only verifies the CR is accepted as structurally valid.
		asYAML := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
kind: GameTypeAutoscaler
metadata:
  name: %s
  namespace: %s
spec:
  gameTypeName: %s
  policy:
    type: webhook
    webhook:
      url: http://localhost:9999
      path: scale
  sync:
    type: fixedinterval
    interval: 30s
`, autoscalerName, testNamespace, gameTypeName)
		tmpAS := writeManifest(autoscalerName, asYAML)
		cmd = exec.Command("kubectl", "apply", "-f", tmpAS)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "GameTypeAutoscaler should be accepted by the webhook")

		By("verifying the resource exists in the cluster")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "gametypeautoscaler", autoscalerName,
				"-n", testNamespace, "-o", "jsonpath={.metadata.name}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal(autoscalerName))
		}, time.Minute, 5*time.Second).Should(Succeed())

		By("checking that no validation error events were emitted for the autoscaler")
		validationReasons := []string{
			"GameAutoscalerInvalidTarget",
			"GameautoscalerInvalidAutoscalePolicy",
			"GameautoscalerInvalidSyncType",
		}
		Consistently(func(g Gomega) {
			for _, reason := range validationReasons {
				cmd := exec.Command("kubectl", "get", "events",
					"-n", testNamespace,
					"--field-selector", fmt.Sprintf("involvedObject.name=%s,reason=%s", autoscalerName, reason),
					"-o", "jsonpath={.items[*].reason}",
				)
				output, _ := utils.Run(cmd)
				g.Expect(output).To(BeEmpty())
			}
		}, 10*time.Second, 2*time.Second).Should(Succeed())
	})
}
