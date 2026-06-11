package sidecar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/MirrorStudios/fallernetes-operator/operator/internal/sidecar/gen"
)

func newClient(pod *v1.Pod, port string) (*gen.ClientWithResponses, error) {
	return gen.NewClientWithResponses(
		buildPodBaseAddress(pod, port),
		gen.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	)
}

// IsDeleteAllowed asks the sidecar whether the pod may be deleted.
func IsDeleteAllowed(pod *v1.Pod, port string) (bool, error) {
	client, err := newClient(pod, port)
	if err != nil {
		return false, err
	}

	resp, err := client.GetAllowDeleteWithResponse(context.Background())
	if err != nil {
		return false, err
	}
	if resp.StatusCode() != http.StatusOK {
		return false, errors.New("GET allow_delete returned: " + resp.Status())
	}
	if resp.JSON200 == nil {
		return false, errors.New("GET allow_delete: empty response body")
	}
	return resp.JSON200.Allowed, nil
}

// RequestShutdown tells the sidecar that the operator has requested shutdown.
func RequestShutdown(pod *v1.Pod, port string) error {
	client, err := newClient(pod, port)
	if err != nil {
		return err
	}

	resp, err := client.SetShutdownWithResponse(context.Background(), gen.SetShutdownJSONRequestBody{Shutdown: true})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.New("POST shutdown returned: " + resp.Status())
	}
	return nil
}

func buildPodBaseAddress(pod *v1.Pod, port string) string {
	return fmt.Sprintf("http://%s:%s/", pod.Status.PodIP, port)
}
