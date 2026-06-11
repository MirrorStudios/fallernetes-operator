package app

import (
	"log"
	"net/http"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		log.Fatal("Could not add client-go scheme: ", err)
	}
	if err := gameserverv1alpha1.AddToScheme(scheme); err != nil {
		log.Fatal("Could not add gameserver scheme: ", err)
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal("Could not create k8s client: ", err)
	}

	adapter := kubeadapter.NewKubeAdapter(k8sClient)
	svc := service.NewOperatorService(adapter, adapter, adapter, adapter, adapter)

	return &App{
		Mux:     http.NewServeMux(),
		Service: svc,
	}
}
