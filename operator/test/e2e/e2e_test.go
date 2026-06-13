//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// namespace where the project is deployed in
const namespace = "operator-system"

// serviceAccountName created for the project
const serviceAccountName = "operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "operator-metrics-binding"

// testNamespace is the namespace used for Fleet test workloads (distinct from the operator namespace).
const testNamespace = "default"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		// In Ginkgo v2, DeferCleanup registered in the last It of an Ordered container runs after
		// AfterAll, not before it. Explicitly delete all remaining fleets here so AfterAll does not
		// depend on DeferCleanup having already submitted the delete requests.
		By("deleting any remaining fleets in default namespace")
		cmd := exec.Command("kubectl", "delete", "fleet", "--all", "-n", "default",
			"--wait=false", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		// Wait for fleet resources to be fully deleted (finalizers removed) before undeploying.
		// Undeploying while finalizers are pending blocks kubectl delete (waiting on webhook/namespace).
		By("waiting for all fleets in default namespace to be fully deleted")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "fleets",
				"-n", "default",
				"-o", "jsonpath={.items[*].metadata.name}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(strings.TrimSpace(output)).To(BeEmpty())
		}, 3*time.Minute, 5*time.Second).Should(Succeed())

		By("cleaning up the curl pod for metrics")
		cmd = exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				By("getting the name of the controller-manager pod")
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				By("validating the pod's status")
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("ensuring the controller pod is ready")
			verifyControllerPodReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "Controller pod not ready")
			}
			Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

			By("waiting for the webhook service endpoints to be ready")
			verifyWebhookEndpointsReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpointslices.discovery.k8s.io", "-n", namespace,
					"-l", "kubernetes.io/service-name=operator-webhook-service",
					"-o", "jsonpath={range .items[*]}{range .endpoints[*]}{.addresses[*]}{end}{end}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Webhook endpoints should exist")
				g.Expect(output).ShouldNot(BeEmpty(), "Webhook endpoints not yet ready")
			}
			Eventually(verifyWebhookEndpointsReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying the mutating webhook server is ready")
			verifyMutatingWebhookReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "mutatingwebhookconfigurations.admissionregistration.k8s.io",
					"operator-mutating-webhook-configuration",
					"-o", "jsonpath={.webhooks[0].clientConfig.caBundle}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "MutatingWebhookConfiguration should exist")
				g.Expect(output).ShouldNot(BeEmpty(), "Mutating webhook CA bundle not yet injected")
			}
			Eventually(verifyMutatingWebhookReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying the validating webhook server is ready")
			verifyValidatingWebhookReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "validatingwebhookconfigurations.admissionregistration.k8s.io",
					"operator-validating-webhook-configuration",
					"-o", "jsonpath={.webhooks[0].clientConfig.caBundle}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "ValidatingWebhookConfiguration should exist")
				g.Expect(output).ShouldNot(BeEmpty(), "Validating webhook CA bundle not yet injected")
			}
			Eventually(verifyValidatingWebhookReady, 3*time.Minute, time.Second).Should(Succeed())

			By("waiting additional time for webhook server to stabilize")
			time.Sleep(5 * time.Second)

			// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": [
								"for i in $(seq 1 30); do curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
							],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		It("should provisioned cert-manager", func() {
			By("validating that cert-manager has the certificate Secret")
			verifyCertManager := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(verifyCertManager).Should(Succeed())
		})

		It("should have CA injection for mutating webhooks", func() {
			By("checking CA injection for mutating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"mutatingwebhookconfigurations.admissionregistration.k8s.io",
					"operator-mutating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				mwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		It("should have CA injection for validating webhooks", func() {
			By("checking CA injection for validating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"validatingwebhookconfigurations.admissionregistration.k8s.io",
					"operator-validating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				vwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		It("should create a Running pod for a Server", func() {
			const testServerName = "e2e-server-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "server", testServerName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
			})

			By("applying a Server manifest")
			tmpFile := writeServerManifest(testServerName, testNamespace,
				`while true; do if wget -qO- http://localhost:8080/shutdown 2>/dev/null | grep -q 'true'; then wget -qO- --post-data='{"allowed":true}' http://localhost:8080/allow_delete 2>/dev/null; exit 0; fi; sleep 2; done`)
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the pod to appear")
			podName := testServerName + "-pod"
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", podName,
					"-n", testNamespace,
					"-o", "jsonpath={.metadata.name}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(podName))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

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
			const testServerName = "e2e-server-sidecar-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "server", testServerName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
			})

			By("applying a Server manifest")
			tmpFile := writeServerManifest(testServerName, testNamespace, "sleep 3600")
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			podName := testServerName + "-pod"

			By("waiting for the pod to appear")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", podName,
					"-n", testNamespace,
					"-o", "jsonpath={.metadata.name}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(podName))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

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
			const testServerName = "e2e-server-del-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "server", testServerName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
			})

			By("applying a Server manifest")
			tmpFile := writeServerManifest(testServerName, testNamespace, "sleep 3600")
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			podName := testServerName + "-pod"

			By("waiting for the pod to exist")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", podName,
					"-n", testNamespace,
					"-o", "jsonpath={.metadata.name}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(podName))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("deleting the Server")
			cmd = exec.Command("kubectl", "delete", "server", testServerName, "-n", testNamespace, "--wait=false")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the pod to be gone")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", podName, "-n", testNamespace)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred())
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create a Fleet for a GameType", func() {
			const testGTName = "e2e-gametype-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "gametype", testGTName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "fleets",
						"-l", fmt.Sprintf("gametype=%s", testGTName),
						"-n", testNamespace,
						"-o", "jsonpath={.items[*].metadata.name}",
					)
					output, _ := utils.Run(cmd)
					g.Expect(strings.TrimSpace(output)).To(BeEmpty())
				}, 2*time.Minute, 5*time.Second).Should(Succeed())
			})

			By("applying a GameType manifest")
			gtYAML := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
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
          image: busybox:latest
          command: ["sh", "-c", "while true; do if wget -qO- http://localhost:8080/shutdown 2>/dev/null | grep -q 'true'; then wget -qO- --post-data='{\"allowed\":true}' http://localhost:8080/allow_delete 2>/dev/null; exit 0; fi; sleep 2; done"]
`, testGTName, testNamespace, sidecarImage)
			tmpFile := filepath.Join(os.TempDir(), "e2e-gametype.yaml")
			Expect(os.WriteFile(tmpFile, []byte(gtYAML), 0644)).To(Succeed())
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for a Fleet labeled with the gametype name to appear")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "fleets",
					"-l", fmt.Sprintf("gametype=%s", testGTName),
					"-n", testNamespace,
					"-o", "jsonpath={.items[*].metadata.name}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(utils.GetNonEmptyLines(output)).NotTo(BeEmpty())
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying the GameType status records the fleet name")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "gametype", testGTName,
					"-n", testNamespace,
					"-o", "jsonpath={.status.activeFleetName}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).NotTo(BeEmpty())
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should create a new Fleet when GameType pod spec changes", func() {
			const testGTName = "e2e-gametype-rolling-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "gametype", testGTName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "fleets",
						"-l", fmt.Sprintf("gametype=%s", testGTName),
						"-n", testNamespace,
						"-o", "jsonpath={.items[*].metadata.name}",
					)
					output, _ := utils.Run(cmd)
					g.Expect(strings.TrimSpace(output)).To(BeEmpty())
				}, 2*time.Minute, 5*time.Second).Should(Succeed())
			})

			By("applying initial GameType")
			gtYAML := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
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
          image: busybox:1.35
          command: ["sh", "-c", "sleep 3600"]
`, testGTName, testNamespace, sidecarImage)
			tmpFile := filepath.Join(os.TempDir(), "e2e-gametype-rolling.yaml")
			Expect(os.WriteFile(tmpFile, []byte(gtYAML), 0644)).To(Succeed())
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the initial Fleet to appear")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "fleets",
					"-l", fmt.Sprintf("gametype=%s", testGTName),
					"-n", testNamespace,
					"-o", "jsonpath={.items[*].metadata.name}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(utils.GetNonEmptyLines(output)).To(HaveLen(1))
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("patching the GameType to change the container image")
			cmd = exec.Command("kubectl", "patch", "gametype", testGTName,
				"-n", testNamespace, "--type=merge",
				"-p", `{"spec":{"fleetSpec":{"spec":{"pod":{"containers":[{"name":"game-server","image":"busybox:1.36","command":["sh","-c","sleep 3600"]}]}}}}}`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for a second Fleet to appear (rolling update in progress)")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "fleets",
					"-l", fmt.Sprintf("gametype=%s", testGTName),
					"-n", testNamespace,
					"-o", "jsonpath={.items[*].metadata.name}",
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(utils.GetNonEmptyLines(output)).To(HaveLen(2))
			}, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should accept and reconcile a GameTypeAutoscaler without validation errors", func() {
			const testASName = "e2e-autoscaler-test"
			const testGTName = "e2e-autoscaler-gametype"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "gametypeautoscaler", testASName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
				cmd = exec.Command("kubectl", "delete", "gametype", testGTName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
				Eventually(func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "fleets",
						"-l", fmt.Sprintf("gametype=%s", testGTName),
						"-n", testNamespace,
						"-o", "jsonpath={.items[*].metadata.name}",
					)
					output, _ := utils.Run(cmd)
					g.Expect(strings.TrimSpace(output)).To(BeEmpty())
				}, 2*time.Minute, 5*time.Second).Should(Succeed())
			})

			By("creating a GameType for the autoscaler to target")
			gtYAML := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
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
          image: busybox:latest
          command: ["sh", "-c", "while true; do if wget -qO- http://localhost:8080/shutdown 2>/dev/null | grep -q 'true'; then wget -qO- --post-data='{\"allowed\":true}' http://localhost:8080/allow_delete 2>/dev/null; exit 0; fi; sleep 2; done"]
`, testGTName, testNamespace, sidecarImage)
			gtFile := filepath.Join(os.TempDir(), "e2e-autoscaler-gametype.yaml")
			Expect(os.WriteFile(gtFile, []byte(gtYAML), 0644)).To(Succeed())
			cmd := exec.Command("kubectl", "apply", "-f", gtFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("applying a GameTypeAutoscaler manifest")
			// url points to an unreachable host; webhook call failures are expected and not tested here.
			// Correct field names from the types: spec.policy (not autoscalePolicy), spec.sync.interval
			// (not sync.time), and lowercase enum values: webhook / fixedinterval.
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
`, testASName, testNamespace, testGTName)
			tmpFile := filepath.Join(os.TempDir(), "e2e-autoscaler.yaml")
			Expect(os.WriteFile(tmpFile, []byte(asYAML), 0644)).To(Succeed())
			cmd = exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "GameTypeAutoscaler should be accepted by the webhook")

			By("verifying the resource exists in the cluster")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "gametypeautoscaler", testASName,
					"-n", testNamespace, "-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(testASName))
			}, time.Minute, 5*time.Second).Should(Succeed())

			By("checking that no validation error events were emitted for the autoscaler")
			// GameautoscalerWebhook warnings are expected (no server in e2e); only check that the
			// CR fields were accepted as valid by the controller.
			validationReasons := []string{
				"GameAutoscalerInvalidTarget",
				"GameautoscalerInvalidAutoscalePolicy",
				"GameautoscalerInvalidSyncType",
			}
			Consistently(func(g Gomega) {
				for _, reason := range validationReasons {
					cmd := exec.Command("kubectl", "get", "events",
						"-n", testNamespace,
						"--field-selector", fmt.Sprintf("involvedObject.name=%s,reason=%s", testASName, reason),
						"-o", "jsonpath={.items[*].reason}",
					)
					output, _ := utils.Run(cmd)
					g.Expect(output).To(BeEmpty())
				}
			}, 10*time.Second, 2*time.Second).Should(Succeed())
		})

		It("should create Running pods for a Fleet", func() {
			const testFleetName = "e2e-fleet-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "fleet", testFleetName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
			})

			By("applying a sample Fleet manifest")
			tmpFile := writeFleetManifest(testFleetName, testNamespace, 2)
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply Fleet manifest")

			By("waiting for 2 pods to appear with the fleet label")
			Eventually(waitForFleetPods(testFleetName, testNamespace, 2), 2*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying all pods reach Running phase")
			Eventually(waitForFleetPodsRunning(testFleetName, testNamespace), 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should remove all Servers and Pods when a Fleet is deleted", func() {
			const testFleetName = "e2e-fleet-del-test"

			By("creating a Fleet with 2 replicas")
			tmpFile := writeFleetManifest(testFleetName, testNamespace, 2)
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply Fleet manifest")

			By("waiting for 2 pods to be Running")
			Eventually(waitForFleetPods(testFleetName, testNamespace, 2), 2*time.Minute, 5*time.Second).Should(Succeed())
			Eventually(waitForFleetPodsRunning(testFleetName, testNamespace), 3*time.Minute, 5*time.Second).Should(Succeed())

			By("deleting the Fleet")
			cmd = exec.Command("kubectl", "delete", "fleet", testFleetName, "-n", testNamespace, "--wait=false")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for all Pods with the fleet label to disappear")
			Eventually(waitForFleetPods(testFleetName, testNamespace, 0), 3*time.Minute, 5*time.Second).Should(Succeed())

			By("waiting for all Server CRs with the fleet label to disappear")
			Eventually(waitForFleetServers(testFleetName, testNamespace, 0), 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should remove excess pods when a Fleet is scaled down", func() {
			const testFleetName = "e2e-fleet-scaledown-test"

			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "fleet", testFleetName, "-n", testNamespace,
					"--wait=false", "--ignore-not-found=true")
				_, _ = utils.Run(cmd)
			})

			By("applying a Fleet with 3 replicas")
			tmpFile := writeFleetManifest(testFleetName, testNamespace, 3)
			cmd := exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for 3 pods to appear")
			Eventually(waitForFleetPods(testFleetName, testNamespace, 3), 2*time.Minute, 5*time.Second).Should(Succeed())

			By("patching fleet replicas down to 1")
			cmd = exec.Command("kubectl", "patch", "fleet", testFleetName,
				"-n", testNamespace,
				"--type=merge",
				"-p", `{"spec":{"scaling":{"replicas":1}}}`,
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting until only 1 pod remains")
			Eventually(waitForFleetPods(testFleetName, testNamespace, 1), 2*time.Minute, 5*time.Second).Should(Succeed())

			By("waiting until only 1 Server CR remains")
			Eventually(waitForFleetServers(testFleetName, testNamespace, 1), 2*time.Minute, 5*time.Second).Should(Succeed())
		})

	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	By("creating temporary file to store the token request")
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		By("executing kubectl command to create the token")
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		By("parsing the JSON output to extract the token")
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

// waitForFleetPods returns a Gomega assertion function that passes when exactly count pods
// with the given fleet label exist. Use count=0 to assert all pods are gone.
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

// waitForFleetServers returns a Gomega assertion function that passes when exactly count Server CRs
// with the given fleet label exist. Use count=0 to assert all servers are gone.
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
// with the given fleet label is in the Running phase.
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

// writeServerManifest writes a Server manifest to a temp file and returns its path.
func writeServerManifest(name, ns, command string) string {
	yaml := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
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
`, name, ns, sidecarImage, command)
	path := filepath.Join(os.TempDir(), name+".yaml")
	Expect(os.WriteFile(path, []byte(yaml), 0644)).To(Succeed())
	return path
}

// writeFleetManifest writes a Fleet manifest to a temp file and returns its path.
// No force-delete timeout is set: deletion must be approved via the sidecar protocol so that
// tests catch regressions in the sidecar communication path.
func writeFleetManifest(name, ns string, replicas int) string {
	filename := name + ".yaml"
	yaml := fmt.Sprintf(`apiVersion: gameserver.falloria.com/v1alpha1
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
        command: ["sh", "-c", "while true; do if wget -qO- http://localhost:8080/shutdown 2>/dev/null | grep -q 'true'; then wget -qO- --post-data='{\"allowed\":true}' http://localhost:8080/allow_delete 2>/dev/null; exit 0; fi; sleep 2; done"]
`, name, ns, replicas, sidecarImage)
	path := filepath.Join(os.TempDir(), filename)
	Expect(os.WriteFile(path, []byte(yaml), 0644)).To(Succeed())
	return path
}
