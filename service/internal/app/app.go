package app

import (
	"log"
	"net/http"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	kubeadapter "github.com/MirrorStudios/fallernetes-service/internal/adapters/kube"
	"github.com/MirrorStudios/fallernetes-service/internal/service"
)

type App struct {
	Mux     *http.ServeMux
	Service *service.OperatorService
}

func CreateApp() *App {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not create config: ", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal("Could not create dynamic client: ", err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("Could not create clientset: ", err)
	}

	adapter := kubeadapter.NewKubeAdapter(dynamicClient, clientSet)
	svc := service.NewOperatorService(adapter, adapter, adapter, adapter, adapter)

	return &App{
		Mux:     http.NewServeMux(),
		Service: svc,
	}
}
