package kubeclient

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/SDA-SE/sdase-image-collector/internal/cmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func CreateClientOrDie(kubeconfig, kubecontext, masterURL string) *kubernetes.Clientset {
	if kubeconfig == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
	}

	var (
		config *rest.Config
		err    error
	)

	if kubeconfig == "" {
		log.Info().Msg("Using inCluster-config based on serviceaccount-token")
		config, err = rest.InClusterConfig()
	} else {
		log.Info().Msg("Using kubeconfig")
		config, err = buildConfigFromFlags(masterURL, kubeconfig, kubecontext)
	}
	cmd.CheckError(err)

	return kubernetes.NewForConfigOrDie(config)
}

func buildConfigFromFlags(masterURL, kubeconfig, kubecontext string) (*rest.Config, error) {
	overrides := clientcmd.ConfigOverrides{}
	if kubecontext != "" {
		overrides.CurrentContext = kubecontext
	}
	if masterURL != "" {
		overrides.ClusterInfo = api.Cluster{Server: masterURL}
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&overrides,
	).ClientConfig()
}
