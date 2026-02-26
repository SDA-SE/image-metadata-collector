package kubeclient

import (
	"context"
	"maps"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"

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
	ImageType     string
}

const (
	ImageTypeJob           = "job"
	ImageTypeCronJob       = "cronjob"
	ImageTypeInitContainer = "init_container"
	ImageTypeOther         = "other"
)

// GetImages returns all images of all pods in the given namespaces
// The Labels & Annotations of Pods and Namespaces are merged
func (c *Client) GetImages(namespaces *[]Namespace) (*[]Image, error) {
	var images []Image

	for _, namespace := range *namespaces {

		// 1. Get Pods
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
			// Track regular and init containers separately so image type stays explicit.
			containerImageMap := map[string]string{}
			for _, container := range pod.Spec.Containers {
				containerImageMap[container.Name] = container.Image
			}

			// Get all init container images
			initContainerImageMap := map[string]string{}
			for _, initContainer := range pod.Spec.InitContainers {
				initContainerImageMap[initContainer.Name] = initContainer.Image
			}
			//	pod.Status.ContainerStatuses[0].Image=="minio/console:v0.19.4"

			// Create images for all init containers with status
			for _, status := range pod.Status.InitContainerStatuses {
				var image = CreateImageAndAppend(initContainerImageMap, status, namespace, labels, annotations, ImageTypeInitContainer)
				if (&Image{} != &image) {
					log.Info().Msgf("Adding image from init containers with Status: %s", image)
					images = append(images, image)
				}
			}

			// Create images for all containers with status
			for _, status := range pod.Status.ContainerStatuses {
				var image = CreateImageAndAppend(containerImageMap, status, namespace, labels, annotations, ImageTypeOther)
				if (&Image{} != &image) {
					log.Info().Msgf("Adding image from containers with Status: %s", image)
					images = append(images, image)
				}
			}

			// Add all remaining init container images for which no status exists.
			for _, imageName := range initContainerImageMap {

				image := Image{
					Image:         imageName,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
					ImageType:     ImageTypeInitContainer,
				}
				log.Info().Msgf("Adding image from init containers without Status: %s", image)
				images = append(images, image)
			}

			// Add all remaining container images for which no status exists
			for _, imageName := range containerImageMap {

				image := Image{
					Image:         imageName,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
					ImageType:     ImageTypeOther,
				}
				log.Info().Msgf("Adding image from containers without Status: %s", image)
				images = append(images, image)
			}
		}

		// 2. Get Jobs
		jobs, err := c.Clientset.BatchV1().Jobs(namespace.Name).List(context.Background(), metav1.ListOptions{})

		if err != nil {
			return nil, err
		}

		for _, job := range jobs.Items {

			// Merge Pod and Namespace Labels & Annotations
			labels := job.GetLabels()
			if labels == nil {
				labels = namespace.Labels
			} else {
				maps.Copy(labels, namespace.Labels)
			}

			annotations := job.GetAnnotations()
			if annotations == nil {
				annotations = namespace.Annotations
			} else {
				maps.Copy(annotations, namespace.Annotations)
			}

			// Reference: https://kubernetes.io/docs/concepts/workloads/controllers/job/
			for _, container := range job.Spec.Template.Spec.Containers {
				image := Image{
					Image:         container.Image,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
					ImageType:     ImageTypeJob,
				}
				log.Info().Msgf("Adding image from Jobs (without Status): %s", image)
				images = append(images, image)
			}

			// Get all init container images
			for _, initContainer := range job.Spec.Template.Spec.InitContainers {
				image := Image{
					Image:         initContainer.Image,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
					ImageType:     ImageTypeInitContainer,
				}
				log.Info().Msgf("Adding image from Jobs (without Status): %s", image)
				images = append(images, image)
			}
		}

		// 3. Get CronJobs
		cronJobs, err := c.Clientset.BatchV1().CronJobs(namespace.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, cronJob := range cronJobs.Items {
			// Merge Pod and Namespace Labels & Annotations
			labels := cronJob.GetLabels()
			if labels == nil {
				labels = namespace.Labels
			} else {
				maps.Copy(labels, namespace.Labels)
			}
			annotations := cronJob.GetAnnotations()
			if annotations == nil {
				annotations = namespace.Annotations
			} else {
				maps.Copy(annotations, namespace.Annotations)
			}

			// Reference: https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/
			for _, container := range cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers {
				image := Image{
					Image:         container.Image,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
					ImageType:     ImageTypeCronJob,
				}
				log.Info().Msgf("Adding image from Cron Jobs (without Status): %s", image)
				images = append(images, image)
			}

			// Get all init container images
			for _, initContainer := range cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers {
				image := Image{
					Image:         initContainer.Image,
					NamespaceName: namespace.Name,
					Labels:        labels,
					Annotations:   annotations,
					ImageType:     ImageTypeInitContainer,
				}
				log.Info().Msgf("Adding image from Cron Jobs (without Status): %s", image)
				images = append(images, image)
			}
		}
	}

	return &images, nil
}

func CreateImageAndAppend(containerImageMap map[string]string, status v1.ContainerStatus, namespace Namespace, labels map[string]string, annotations map[string]string, imageType string) Image {

	var imageName string
	containerImage := containerImageMap[status.Name]
	delete(containerImageMap, status.Name)

	// Don't create an image if no image name exists
	if containerImage == "" && status.Image == "" {
		return Image{}
	} else if status.Image != "" {
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
		ImageType:     imageType,
	}
	return image
}

func (c *Client) GetAllImagesForAllNamespaces() (*[]Image, error) {
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
	namespaceNames := make([]string, len(*namespaces))
	for i, ns := range *namespaces {
		namespaceNames[i] = ns.Name
	}
	log.Info().Msgf("All Images for namespaces %s have been parsed.", strings.Join(namespaceNames, ", "))
	return k8Images, nil
}
