package kubeclient

import (
	"context"
	"maps"
	"os"

	"github.com/rs/zerolog/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type KubeConfig struct {
	ConfigFile string
	Context    string
	MasterUrl  string
}

type Client struct {
	Clientset kubernetes.Interface
}

func NewClient(cfg *KubeConfig) *Client {
	kubeconfig := cfg.ConfigFile

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
		config, err = buildConfigFromFlags(cfg.MasterUrl, kubeconfig, cfg.Context)
	}
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Couldn't build config from flags")
	}

	client := &Client{Clientset: kubernetes.NewForConfigOrDie(config)}

	return client

}

// TODO: Move this into the NewClient function
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

type Namespace struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

func (c *Client) GetNamespaces() (*[]Namespace, error) {
	k8Namespaces, err := c.Clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var namespaces []Namespace
	for _, k8Namespace := range k8Namespaces.Items {
		namespace := Namespace{
			Name:        k8Namespace.GetName(),
			Labels:      k8Namespace.GetLabels(),
			Annotations: k8Namespace.GetAnnotations(),
		}
		namespaces = append(namespaces, namespace)
	}
	return &namespaces, nil
}

type Image struct {
	Image         string
	ImageId       string
	NamespaceName string
	Labels        map[string]string
	Annotations   map[string]string
}

// GetImages returns all images of all pods in the given namespaces
// The Labels & Annotations of Pods and Namespaces are merged
func (c *Client) GetImages(namespaces *[]Namespace) (*[]Image, error) {
	var images []Image

	for _, namespace := range *namespaces {
		pods, err := c.Clientset.CoreV1().Pods(namespace.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {

			// Merge Pod and Namespace Labels & Annotations
			labels := pod.GetLabels()
			if labels == nil {
				labels = namespace.Labels
			} else {
				maps.Copy(labels, namespace.Labels)
			}
			annotations := pod.GetAnnotations()
			if annotations == nil {
				annotations = namespace.Annotations
			} else {
				maps.Copy(annotations, namespace.Annotations)
			}

			// Get all container images
			containerImageMap := map[string]string{}
			for _, container := range pod.Spec.Containers {
				containerImageMap[container.Name] = container.Image
			}

			// Create images for all containers with status
			for _, status := range pod.Status.ContainerStatuses {
				var imageName string
				containerImage := containerImageMap[status.Name]
				delete(containerImageMap, status.Name)

				// Don't create an image if no image name exists
				if containerImage == "" && status.Image == "" {
					continue
				} else if containerImage == "" {
					imageName = status.Image
				} else {
					imageName = containerImage
				}

				image := Image{
					Image:         imageName,
					ImageId:       status.ImageID,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
				}
				images = append(images, image)
			}

			// Add all remaining container images for which no status exists
			for _, imageName := range containerImageMap {

				image := Image{
					Image:         imageName,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
				}
				images = append(images, image)
			}
		}
	}

	return &images, nil
}

// GetAllImages retrieve all Images for all Namespacesk
func (c *Client) GetAllImages() (*[]Image, error) {
	namespaces, err := c.GetNamespaces()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed to get namespaces")
		return nil, err
	}
	k8Images, err := c.GetImages(namespaces)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed to get images")
		return nil, err
	}

	return k8Images, nil
}
