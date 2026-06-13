package app

import (
	"log/slog"
	"net/http"
	"os"

	gameserverv1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubeadapter "github.com/MirrorStudios/fallernetes-operator/service/internal/adapters/kube"
	"github.com/MirrorStudios/fallernetes-operator/service/internal/service"
)

type App struct {
	Mux     *http.ServeMux
	Service *service.OperatorService
	Logger  *slog.Logger
}

func CreateApp(logger *slog.Logger) *App {
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("Could not create config", "error", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		logger.Error("Could not add client-go scheme", "error", err)
		os.Exit(1)
	}
	if err := gameserverv1alpha1.AddToScheme(scheme); err != nil {
		logger.Error("Could not add gameserver scheme", "error", err)
		os.Exit(1)
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("Could not create k8s client", "error", err)
		os.Exit(1)
	}

	adapter := kubeadapter.NewKubeAdapter(k8sClient)
	svc := service.NewOperatorService(adapter, adapter, adapter, adapter, adapter)

	return &App{
		Mux:     http.NewServeMux(),
		Service: svc,
		Logger:  logger,
	}
}
