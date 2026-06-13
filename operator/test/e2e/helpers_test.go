//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MirrorStudios/fallernetes-operator/operator/test/utils"
)

// sidecarProtocolCmd is the game-server command that cooperates with the sidecar
// shutdown/allow_delete protocol. Used wherever a pod needs to run through the
// full deletion flow rather than just sleeping.
const sidecarProtocolCmd = `while true; do if wget -qO- http://localhost:8080/shutdown 2>/dev/null | grep -q 'true'; then wget -qO- --post-data='{"allowed":true}' http://localhost:8080/allow_delete 2>/dev/null; exit 0; fi; sleep 2; done`

// tokenRequest is a minimal representation of the Kubernetes TokenRequest API response.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

// serviceAccountToken requests a short-lived token for the controller-manager service account.
func serviceAccountToken() (string, error) {
	const body = `{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest"}`
	reqFile := filepath.Join("/tmp", serviceAccountName+"-token-request")
	if err := os.WriteFile(reqFile, []byte(body), 0o644); err != nil {
		return "", err
	}

	var out string
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "create", "--raw",
			fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/%s/token", namespace, serviceAccountName),
			"-f", reqFile)
		raw, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())
		var tr tokenRequest
		g.Expect(json.Unmarshal(raw, &tr)).To(Succeed())
		out = tr.Status.Token
	}).Should(Succeed())
	return out, nil
}

// getMetricsOutput returns the logs from the curl-metrics pod.
func getMetricsOutput() (string, error) {
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// waitForPod returns a Gomega assertion function that passes once the named pod exists.
func waitForPod(name, ns string) func(Gomega) {
	return func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pod", name,
			"-n", ns,
			"-o", "jsonpath={.metadata.name}",
		)
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(Equal(name))
	}
}

// waitForFleetPods returns a Gomega assertion function that passes when exactly count pods
// with the fleet label exist. Use count=0 to assert all pods are gone.
func waitForFleetPods(fleetName, ns string, count int) func(Gomega) {
	return func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("fleet=%s", fleetName),
			"-n", ns,
			"-o", "jsonpath={.items[*].metadata.name}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.Fields(output)).To(HaveLen(count))
	}
}

// waitForFleetServers returns a Gomega assertion function that passes when exactly count
// Server CRs with the fleet label exist. Use count=0 to assert all servers are gone.
func waitForFleetServers(fleetName, ns string, count int) func(Gomega) {
	return func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "servers",
			"-l", fmt.Sprintf("fleet=%s", fleetName),
			"-n", ns,
			"-o", "jsonpath={.items[*].metadata.name}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.Fields(output)).To(HaveLen(count))
	}
}

// waitForFleetPodsRunning returns a Gomega assertion function that passes when every pod
// with the fleet label is in the Running phase.
func waitForFleetPodsRunning(fleetName, ns string) func(Gomega) {
	return func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("fleet=%s", fleetName),
			"-n", ns,
			"-o", "jsonpath={.items[*].status.phase}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		for _, phase := range strings.Fields(output) {
			g.Expect(phase).To(Equal("Running"))
		}
	}
}

// cleanupGameType registers a DeferCleanup that deletes the named GameType and waits
// for all of its Fleets to be fully removed before returning.
func cleanupGameType(name, ns string) {
	DeferCleanup(func() {
		cmd := exec.Command("kubectl", "delete", "gametype", name, "-n", ns,
			"--wait=false", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "fleets",
				"-l", fmt.Sprintf("gametype=%s", name),
				"-n", ns,
				"-o", "jsonpath={.items[*].metadata.name}",
			)
			output, _ := utils.Run(cmd)
			g.Expect(strings.TrimSpace(output)).To(BeEmpty())
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})
}

// writeManifest writes arbitrary YAML content to a temp file named after name and returns the path.
func writeManifest(name, content string) string {
	path := filepath.Join(os.TempDir(), name+".yaml")
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}

// writeServerManifest writes a Server manifest to a temp file and returns the path.
// The sidecar image comes from the suite-level sidecarImage variable.
func writeServerManifest(name, ns, command string) string {
	escapedCmd := strings.ReplaceAll(command, `"`, `\"`)
	content := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
kind: Server
metadata:
  name: %s
  namespace: %s
spec:
  sidecar:
    port: 8080
    image: %s
  pod:
    containers:
    - name: game-server
      image: busybox:latest
      command: ["sh", "-c", "%s"]
`, name, ns, sidecarImage, escapedCmd)
	path := filepath.Join(os.TempDir(), name+".yaml")
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}

// writeGameTypeManifest writes a GameType manifest to a temp file and returns the path.
// image is the game-server container image; command is the shell command to run.
func writeGameTypeManifest(name, ns, image, command string) string {
	escapedCmd := strings.ReplaceAll(command, `"`, `\"`)
	content := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
kind: GameType
metadata:
  name: %s
  namespace: %s
spec:
  fleetSpec:
    scaling:
      replicas: 1
      agePriority: oldest_first
      prioritizeAllowed: false
    spec:
      sidecar:
        port: 8080
        image: %s
      pod:
        containers:
        - name: game-server
          image: %s
          command: ["sh", "-c", "%s"]
`, name, ns, sidecarImage, image, escapedCmd)
	path := filepath.Join(os.TempDir(), name+".yaml")
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}

// writeFleetManifest writes a Fleet manifest to a temp file and returns the path.
// No force-delete timeout is set: deletion must be approved via the sidecar protocol.
func writeFleetManifest(name, ns string, replicas int) string {
	escapedCmd := strings.ReplaceAll(sidecarProtocolCmd, `"`, `\"`)
	content := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
kind: Fleet
metadata:
  name: %s
  namespace: %s
spec:
  scaling:
    replicas: %d
    agePriority: oldest_first
    prioritizeAllowed: false
  spec:
    sidecar:
      port: 8080
      image: %s
    pod:
      containers:
      - name: game-server
        image: busybox:latest
        command: ["sh", "-c", "%s"]
`, name, ns, replicas, sidecarImage, escapedCmd)
	path := filepath.Join(os.TempDir(), name+".yaml")
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}
