package kubeclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.yaml.in/yaml/v3"
)

const (
	inClusterHostEnv = "KUBERNETES_SERVICE_HOST"
	inClusterPortEnv = "KUBERNETES_SERVICE_PORT"
	requestTimeout   = 30 * time.Second
)

var (
	defaultKubeconfigPathFunc = defaultKubeconfigPath
	serviceAccountTokenPath   = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	serviceAccountCAPath      = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type KubeConfig struct {
	ConfigFile string
	Context    string
	MasterUrl  string
}

type Client struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

func NewClient(cfg *KubeConfig) *Client {
	client, err := newClientFromConfig(cfg)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Couldn't build kubernetes client")
	}

	return client
}

func newClientFromConfig(cfg *KubeConfig) (*Client, error) {
	if cfg == nil {
		cfg = &KubeConfig{}
	}

	kubeconfig := cfg.ConfigFile
	if kubeconfig == "" {
		if candidate := defaultKubeconfigPathFunc(); candidate != "" {
			if _, err := os.Stat(candidate); err == nil {
				kubeconfig = candidate
			}
		}
	}

	if kubeconfig == "" {
		log.Info().Msg("Using inCluster-config based on serviceaccount-token")
		return newInClusterClient()
	}

	log.Info().Msg("Using kubeconfig")
	return newClientFromKubeconfig(cfg.MasterUrl, kubeconfig, cfg.Context)
}

func defaultKubeconfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".kube", "config")
}

func newInClusterClient() (*Client, error) {
	host := strings.TrimSpace(os.Getenv(inClusterHostEnv))
	port := strings.TrimSpace(os.Getenv(inClusterPortEnv))
	if host == "" || port == "" {
		return nil, errors.New("in-cluster configuration requires KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT")
	}

	token, err := readTrimmedFile(serviceAccountTokenPath)
	if err != nil {
		return nil, fmt.Errorf("read service account token: %w", err)
	}

	caPEM, err := os.ReadFile(serviceAccountCAPath)
	if err != nil {
		return nil, fmt.Errorf("read service account CA: %w", err)
	}

	httpClient, err := newHTTPClient(tlsMaterial{
		caPEM: caPEM,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:     fmt.Sprintf("https://%s:%s", host, port),
		bearerToken: token,
		httpClient:  httpClient,
	}, nil
}

func newClientFromKubeconfig(masterURL, kubeconfigPath, kubeContext string) (*Client, error) {
	configData, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("read kubeconfig: %w", err)
	}

	var cfg kubeconfigDocument
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}

	contextName := cfg.CurrentContext
	if kubeContext != "" {
		contextName = kubeContext
	}
	if contextName == "" {
		return nil, errors.New("kubeconfig does not define a current context and no --kube-context override was provided")
	}

	selectedContext, err := cfg.findContext(contextName)
	if err != nil {
		return nil, err
	}

	cluster, err := cfg.findCluster(selectedContext.Context.Cluster)
	if err != nil {
		return nil, err
	}

	user, err := cfg.findUser(selectedContext.Context.User)
	if err != nil {
		return nil, err
	}

	serverURL := strings.TrimSpace(cluster.Cluster.Server)
	if masterURL != "" {
		serverURL = masterURL
	}
	if serverURL == "" {
		return nil, errors.New("kubeconfig cluster server is empty")
	}

	httpClient, bearerToken, err := newHTTPClientFromKubeconfig(filepath.Dir(kubeconfigPath), cluster.Cluster, user.User)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:     strings.TrimRight(serverURL, "/"),
		bearerToken: bearerToken,
		httpClient:  httpClient,
	}, nil
}

type Namespace struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

func (c *Client) GetNamespaces() (*[]Namespace, error) {
	var response namespaceListResponse
	if err := c.getJSON("/api/v1/namespaces", &response); err != nil {
		return nil, err
	}

	namespaces := make([]Namespace, 0, len(response.Items))
	for _, item := range response.Items {
		namespaces = append(namespaces, Namespace{
			Name:        item.Metadata.Name,
			Labels:      item.Metadata.Labels,
			Annotations: item.Metadata.Annotations,
		})
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

// GetImages returns all images of all pods in the given namespaces.
// The Labels & Annotations of workloads and namespaces are merged.
func (c *Client) GetImages(namespaces *[]Namespace) (*[]Image, error) {
	var images []Image

	for _, namespace := range *namespaces {
		pods, err := c.listPods(namespace.Name)
		if err != nil {
			return nil, err
		}
		for _, pod := range pods.Items {
			labels := mergeMaps(pod.Metadata.Labels, namespace.Labels)
			annotations := mergeMaps(pod.Metadata.Annotations, namespace.Annotations)
			images = append(images, extractPodImages(namespace, labels, annotations, pod)...)
		}

		jobs, err := c.listJobs(namespace.Name)
		if err != nil {
			return nil, err
		}
		for _, job := range jobs.Items {
			labels := mergeMaps(job.Metadata.Labels, namespace.Labels)
			annotations := mergeMaps(job.Metadata.Annotations, namespace.Annotations)
			images = append(images, extractTemplateImages(namespace, labels, annotations, job.Spec.Template.Spec, ImageTypeJob)...)
		}

		cronJobs, err := c.listCronJobs(namespace.Name)
		if err != nil {
			return nil, err
		}
		for _, cronJob := range cronJobs.Items {
			labels := mergeMaps(cronJob.Metadata.Labels, namespace.Labels)
			annotations := mergeMaps(cronJob.Metadata.Annotations, namespace.Annotations)
			images = append(images, extractTemplateImages(namespace, labels, annotations, cronJob.Spec.JobTemplate.Spec.Template.Spec, ImageTypeCronJob)...)
		}
	}

	return &images, nil
}

func extractPodImages(namespace Namespace, labels map[string]string, annotations map[string]string, pod podItem) []Image {
	var images []Image

	containerImageMap := make(map[string]string, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		containerImageMap[container.Name] = container.Image
	}

	initContainerImageMap := make(map[string]string, len(pod.Spec.InitContainers))
	for _, initContainer := range pod.Spec.InitContainers {
		initContainerImageMap[initContainer.Name] = initContainer.Image
	}

	for _, status := range pod.Status.InitContainerStatuses {
		image := CreateImageAndAppend(initContainerImageMap, status, namespace, labels, annotations, ImageTypeInitContainer)
		if !isZeroImage(image) {
			log.Info().Msgf("Adding image from init containers with Status: %+v", image)
			images = append(images, image)
		}
	}

	for _, status := range pod.Status.ContainerStatuses {
		image := CreateImageAndAppend(containerImageMap, status, namespace, labels, annotations, ImageTypeOther)
		if !isZeroImage(image) {
			log.Info().Msgf("Adding image from containers with Status: %+v", image)
			images = append(images, image)
		}
	}

	for _, imageName := range initContainerImageMap {
		image := Image{
			Image:         imageName,
			NamespaceName: namespace.Name,
			Labels:        labels,
			Annotations:   annotations,
			ImageType:     ImageTypeInitContainer,
		}
		log.Info().Msgf("Adding image from init containers without Status: %+v", image)
		images = append(images, image)
	}

	for _, imageName := range containerImageMap {
		image := Image{
			Image:         imageName,
			NamespaceName: namespace.Name,
			Labels:        labels,
			Annotations:   annotations,
			ImageType:     ImageTypeOther,
		}
		log.Info().Msgf("Adding image from containers without Status: %+v", image)
		images = append(images, image)
	}

	return images
}

func extractTemplateImages(namespace Namespace, labels map[string]string, annotations map[string]string, spec podSpec, imageType string) []Image {
	var images []Image

	for _, container := range spec.Containers {
		image := Image{
			Image:         container.Image,
			NamespaceName: namespace.Name,
			Labels:        labels,
			Annotations:   annotations,
			ImageType:     imageType,
		}
		log.Info().Msgf("Adding image from workload template: %+v", image)
		images = append(images, image)
	}

	for _, initContainer := range spec.InitContainers {
		image := Image{
			Image:         initContainer.Image,
			NamespaceName: namespace.Name,
			Labels:        labels,
			Annotations:   annotations,
			ImageType:     ImageTypeInitContainer,
		}
		log.Info().Msgf("Adding image from workload init container: %+v", image)
		images = append(images, image)
	}

	return images
}

func isZeroImage(image Image) bool {
	return image.Image == "" &&
		image.ImageId == "" &&
		image.NamespaceName == "" &&
		image.Labels == nil &&
		image.Annotations == nil &&
		image.ImageType == ""
}

func CreateImageAndAppend(containerImageMap map[string]string, status containerStatus, namespace Namespace, labels map[string]string, annotations map[string]string, imageType string) Image {
	containerImage := containerImageMap[status.Name]
	delete(containerImageMap, status.Name)

	var imageName string
	switch {
	case status.Image != "":
		imageName = status.Image
	case containerImage != "":
		imageName = containerImage
	default:
		return Image{}
	}

	return Image{
		Image:         imageName,
		ImageId:       status.ImageID,
		NamespaceName: namespace.Name,
		Labels:        labels,
		Annotations:   annotations,
		ImageType:     imageType,
	}
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

func (c *Client) listPods(namespace string) (*podListResponse, error) {
	var response podListResponse
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods", url.PathEscape(namespace))
	if err := c.getJSON(path, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) listJobs(namespace string) (*jobListResponse, error) {
	var response jobListResponse
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", url.PathEscape(namespace))
	if err := c.getJSON(path, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) listCronJobs(namespace string) (*cronJobListResponse, error) {
	var response cronJobListResponse
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/cronjobs", url.PathEscape(namespace))
	if err := c.getJSON(path, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) getJSON(path string, target any) error {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	if c.bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("kubernetes API GET %s returned status %d: %s", path, response.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode kubernetes API response for %s: %w", path, err)
	}

	return nil
}

func mergeMaps(workload map[string]string, namespace map[string]string) map[string]string {
	switch {
	case workload == nil && namespace == nil:
		return nil
	case workload == nil:
		return maps.Clone(namespace)
	case namespace == nil:
		return maps.Clone(workload)
	default:
		merged := maps.Clone(workload)
		maps.Copy(merged, namespace)
		return merged
	}
}

func readTrimmedFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

type tlsMaterial struct {
	caPEM                []byte
	clientCertificatePEM []byte
	clientKeyPEM         []byte
	insecureSkipVerify   bool
	serverName           string
}

func newHTTPClientFromKubeconfig(baseDir string, cluster kubeconfigCluster, user kubeconfigUser) (*http.Client, string, error) {
	if user.Exec != nil {
		return nil, "", errors.New("unsupported kubeconfig auth: exec plugins are not supported")
	}
	if user.AuthProvider != nil {
		return nil, "", errors.New("unsupported kubeconfig auth: auth-provider plugins are not supported")
	}
	if user.Username != "" || user.Password != "" {
		return nil, "", errors.New("unsupported kubeconfig auth: basic auth is not supported")
	}

	caPEM, err := resolveFileOrData(baseDir, cluster.CertificateAuthority, cluster.CertificateAuthorityData, "certificate-authority")
	if err != nil {
		return nil, "", err
	}

	clientCertificatePEM, err := resolveFileOrData(baseDir, user.ClientCertificate, user.ClientCertificateData, "client-certificate")
	if err != nil {
		return nil, "", err
	}

	clientKeyPEM, err := resolveFileOrData(baseDir, user.ClientKey, user.ClientKeyData, "client-key")
	if err != nil {
		return nil, "", err
	}

	bearerToken := strings.TrimSpace(user.Token)
	if user.TokenFile != "" {
		bearerToken, err = readTrimmedFile(resolvePath(baseDir, user.TokenFile))
		if err != nil {
			return nil, "", fmt.Errorf("read tokenFile: %w", err)
		}
	}

	httpClient, err := newHTTPClient(tlsMaterial{
		caPEM:                caPEM,
		clientCertificatePEM: clientCertificatePEM,
		clientKeyPEM:         clientKeyPEM,
		insecureSkipVerify:   cluster.InsecureSkipTLSVerify,
		serverName:           cluster.TLSServerName,
	})
	if err != nil {
		return nil, "", err
	}

	return httpClient, bearerToken, nil
}

func newHTTPClient(material tlsMaterial) (*http.Client, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: material.insecureSkipVerify,
	}

	if len(material.caPEM) > 0 {
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(material.caPEM) {
			return nil, errors.New("failed to parse CA certificate data")
		}
		tlsConfig.RootCAs = rootCAs
	}

	if material.serverName != "" {
		tlsConfig.ServerName = material.serverName
	}

	if len(material.clientCertificatePEM) > 0 || len(material.clientKeyPEM) > 0 {
		if len(material.clientCertificatePEM) == 0 || len(material.clientKeyPEM) == 0 {
			return nil, errors.New("client certificate authentication requires both certificate and key")
		}

		certificate, err := tls.X509KeyPair(material.clientCertificatePEM, material.clientKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Timeout:   requestTimeout,
		Transport: transport,
	}, nil
}

func resolveFileOrData(baseDir, filePath, inlineData, fieldName string) ([]byte, error) {
	if inlineData != "" {
		decoded, err := base64.StdEncoding.DecodeString(inlineData)
		if err != nil {
			return nil, fmt.Errorf("decode %s data: %w", fieldName, err)
		}
		return decoded, nil
	}

	if filePath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(resolvePath(baseDir, filePath))
	if err != nil {
		return nil, fmt.Errorf("read %s file: %w", fieldName, err)
	}

	return data, nil
}

func resolvePath(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(baseDir, path)
}

type kubeconfigDocument struct {
	CurrentContext string                   `yaml:"current-context"`
	Clusters       []namedKubeconfigCluster `yaml:"clusters"`
	Contexts       []namedKubeconfigContext `yaml:"contexts"`
	Users          []namedKubeconfigUser    `yaml:"users"`
}

type namedKubeconfigCluster struct {
	Name    string            `yaml:"name"`
	Cluster kubeconfigCluster `yaml:"cluster"`
}

type kubeconfigCluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
	TLSServerName            string `yaml:"tls-server-name"`
}

type namedKubeconfigContext struct {
	Name    string            `yaml:"name"`
	Context kubeconfigContext `yaml:"context"`
}

type kubeconfigContext struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type namedKubeconfigUser struct {
	Name string         `yaml:"name"`
	User kubeconfigUser `yaml:"user"`
}

type kubeconfigUser struct {
	Token                 string         `yaml:"token"`
	TokenFile             string         `yaml:"tokenFile"`
	ClientCertificate     string         `yaml:"client-certificate"`
	ClientCertificateData string         `yaml:"client-certificate-data"`
	ClientKey             string         `yaml:"client-key"`
	ClientKeyData         string         `yaml:"client-key-data"`
	Username              string         `yaml:"username"`
	Password              string         `yaml:"password"`
	Exec                  map[string]any `yaml:"exec"`
	AuthProvider          map[string]any `yaml:"auth-provider"`
}

func (k kubeconfigDocument) findCluster(name string) (*namedKubeconfigCluster, error) {
	for _, cluster := range k.Clusters {
		if cluster.Name == name {
			return &cluster, nil
		}
	}

	return nil, fmt.Errorf("kubeconfig cluster %q not found", name)
}

func (k kubeconfigDocument) findContext(name string) (*namedKubeconfigContext, error) {
	for _, kubeContext := range k.Contexts {
		if kubeContext.Name == name {
			return &kubeContext, nil
		}
	}

	return nil, fmt.Errorf("kubeconfig context %q not found", name)
}

func (k kubeconfigDocument) findUser(name string) (*namedKubeconfigUser, error) {
	for _, user := range k.Users {
		if user.Name == name {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("kubeconfig user %q not found", name)
}

type metadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type namespaceListResponse struct {
	Items []struct {
		Metadata metadata `json:"metadata"`
	} `json:"items"`
}

type podListResponse struct {
	Items []podItem `json:"items"`
}

type podItem struct {
	Metadata metadata  `json:"metadata"`
	Spec     podSpec   `json:"spec"`
	Status   podStatus `json:"status"`
}

type podSpec struct {
	Containers     []containerSpec `json:"containers"`
	InitContainers []containerSpec `json:"initContainers"`
}

type containerSpec struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type podStatus struct {
	ContainerStatuses     []containerStatus `json:"containerStatuses"`
	InitContainerStatuses []containerStatus `json:"initContainerStatuses"`
}

type containerStatus struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	ImageID string `json:"imageID"`
}

type jobListResponse struct {
	Items []jobItem `json:"items"`
}

type jobItem struct {
	Metadata metadata `json:"metadata"`
	Spec     jobSpec  `json:"spec"`
}

type jobSpec struct {
	Template podTemplate `json:"template"`
}

type podTemplate struct {
	Spec podSpec `json:"spec"`
}

type cronJobListResponse struct {
	Items []cronJobItem `json:"items"`
}

type cronJobItem struct {
	Metadata metadata    `json:"metadata"`
	Spec     cronJobSpec `json:"spec"`
}

type cronJobSpec struct {
	JobTemplate cronJobTemplate `json:"jobTemplate"`
}

type cronJobTemplate struct {
	Spec jobSpec `json:"spec"`
}
