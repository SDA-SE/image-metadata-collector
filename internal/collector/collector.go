package collector

import (
	"encoding/json"
	"maps"
	"regexp"
	"strings"

	"github.com/SDA-SE/sdase-image-collector/internal/pkg/kubeclient"
	"github.com/SDA-SE/sdase-image-collector/internal/pkg/storage"

	"github.com/rs/zerolog/log"
)

type AnnotationNames struct {
	Base       string
	Scans      string
	Contact    string
	DefectDojo string
}

type CollectorImage struct {
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
	ImageId   string `json:"image_id"`

	// Fields from annotations and labels
	Environment            string   `json:"environment"`
	Product                string   `json:"product"`
	Description            string   `json:"description"`
	AppKubernetesIoName    string   `json:"app_kubernetes_io_name"`
	AppKubernetesIoVersion string   `json:"app_kubernetes_io_version"`
	ContainerType          string   `json:"container_type"`
	Skip                   bool     `json:"skip"`
	NamespaceFilter        string   `json:"namespace_filter"`
	NamespaceFilterNegated string   `json:"namespace_filter_negated"`
	EngagementTags         []string `json:"engagement_tags"`

	Team       string `json:"team"`
	Slack      string `json:"slack"`
	Rocketchat string `json:"rocketchat"`
	Email      string `json:"email"`

	IsScanBaseimageLifetime          bool  `json:"is_scan_baseimage_lifetime"`
	IsScanDependencyCheck            bool  `json:"is_scan_dependency_check"`
	IsScanDependencyTrack            bool  `json:"is_scan_dependency_track"`
	IsScanDistroless                 bool  `json:"is_scan_distroless"`
	IsScanLifetime                   bool  `json:"is_scan_lifetime"`
	IsScanMalware                    bool  `json:"is_scan_maleware"`
	IsScanNewVersion                 bool  `json:"is_scan_new_version"`
	IsScanRunAsRoot                  bool  `json:"is_scan_runasroot"`
	IsPotentiallyRunningAsRoot       bool  `json:"is_scan_potentially_running_as_root"`
	IsScanRunAsPrivileged            bool  `json:"is_scan_run_as_privileged"`
	IsPotentiallyRunningAsPrivileged bool  `json:"is_scan_potentially_running_as_privileged"`
	ScanLifetimeMaxDays              int64 `json:"is_scan_lifetime_max_days"`
}

// convertK8ImageToCollectorImage by considering the images labels, annotations and cluster wide defaults
func convertK8ImageToCollectorImage(k8Image kubeclient.Image, defaults *CollectorImage, annotationNames *AnnotationNames) *CollectorImage {

	tags := k8Image.Labels
	maps.Copy(tags, k8Image.Annotations)
	// fmt.Printf("%s -> tags=%s\n", k8Image.Image, tags)

	collectorImage := &CollectorImage{
		Namespace: k8Image.NamespaceName,
		Image:     k8Image.Image,
		ImageId:   k8Image.ImageId,

		Environment:            GetOrDefaultString(tags, annotationNames.Base+"environment", defaults.Environment),
		Product:                GetOrDefaultString(tags, annotationNames.Base+"product", defaults.Product),
		Description:            GetOrDefaultString(tags, annotationNames.Base+"description", defaults.Description),
		AppKubernetesIoName:    GetOrDefaultString(tags, "app.kubernetes.io/name", ""),
		AppKubernetesIoVersion: GetOrDefaultString(tags, "app.kubernetes.io/version", ""),
		ContainerType:          GetOrDefaultString(tags, annotationNames.Base+"container-type", defaults.ContainerType),
		Skip:                   GetOrDefaultBool(tags, annotationNames.Scans+"skip", defaults.Skip),
		NamespaceFilter:        GetOrDefaultString(tags, annotationNames.Scans+"namespace-filter", defaults.NamespaceFilter),
		NamespaceFilterNegated: GetOrDefaultString(tags, annotationNames.Scans+"negated-namespace-filter", defaults.NamespaceFilterNegated),
		EngagementTags:         GetOrDefaultStringSlice(tags, annotationNames.DefectDojo+"engagement-tags", defaults.EngagementTags),

		Team:       GetOrDefaultString(tags, annotationNames.Contact+"team", defaults.Team),
		Slack:      GetOrDefaultString(tags, annotationNames.Contact+"slack", defaults.Slack),
		Rocketchat: GetOrDefaultString(tags, annotationNames.Contact+"rocketchat", defaults.Rocketchat),
		Email:      GetOrDefaultString(tags, annotationNames.Contact+"email", defaults.Email),

		IsScanBaseimageLifetime:          GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-baseimage-lifetime", defaults.IsScanBaseimageLifetime),
		IsScanDependencyCheck:            GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-dependency-check", defaults.IsScanDependencyCheck),
		IsScanDependencyTrack:            GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-dependency-track", defaults.IsScanDependencyTrack),
		IsScanDistroless:                 GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-distroless", defaults.IsScanDistroless),
		IsScanLifetime:                   GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-lifetime", defaults.IsScanLifetime),
		IsScanMalware:                    GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-maleware", defaults.IsScanMalware),
		IsScanNewVersion:                 GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-new-version", defaults.IsScanNewVersion),
		IsScanRunAsRoot:                  GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-runasroot", defaults.IsScanRunAsRoot),
		IsPotentiallyRunningAsRoot:       GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-potentially-running-as-root", defaults.IsPotentiallyRunningAsRoot),
		IsScanRunAsPrivileged:            GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-run-as-privileged", defaults.IsScanRunAsPrivileged),
		IsPotentiallyRunningAsPrivileged: GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-potentially-running-as-privilaged", defaults.IsPotentiallyRunningAsPrivileged),
		ScanLifetimeMaxDays:              GetOrDefaultInt64(tags, annotationNames.Scans+"scan-lifetime-max-days", defaults.ScanLifetimeMaxDays),
	}

	return collectorImage

}

// isSkipImage by considering the images labels, annotations and deployment wide defaults
func isSkipImage(ci *CollectorImage) bool {
	isNamespaceFilter, _ := regexp.MatchString(ci.NamespaceFilter, ci.Namespace)
	if ci.NamespaceFilter == "" {
		isNamespaceFilter = false
	}

	isNamespaceFilterNegated, _ := regexp.MatchString(ci.NamespaceFilterNegated, ci.Namespace)
	if ci.NamespaceFilterNegated == "" {
		isNamespaceFilterNegated = false
	} else {
		isNamespaceFilterNegated = !isNamespaceFilterNegated
	}

	return ci.Skip || isNamespaceFilter || isNamespaceFilterNegated
}

// cleanCollectorImage applies replacement and other rules to specific fields
func cleanCollectorImage(ci *CollectorImage) {
	ci.Image = strings.Replace(ci.Image, "docker-pullable://", "", -1)
	ci.ImageId = strings.Replace(ci.ImageId, "docker-pullable://", "", -1)

	ci.Skip = isSkipImage(ci)
}

// Collect images from kubernetes, convert, clean and store them in the storage
func Collect(defaults *CollectorImage, annotationNames *AnnotationNames, k8client *kubeclient.Client) (*[]CollectorImage, error) {
	namespaces, err := k8client.GetNamespaces()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed to get namespaces")
		return nil, err
	}
	k8Images, err := k8client.GetImages(namespaces)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("failed to get images")
		return nil, err
	}

	var images []CollectorImage

	for _, k8Image := range *k8Images {
		collectorImage := convertK8ImageToCollectorImage(k8Image, defaults, annotationNames)
		cleanCollectorImage(collectorImage)
		images = append(images, *collectorImage)

	}

	return &images, nil
}

// Store stores images in the provided storager implementation
func Store(images *[]CollectorImage, name string, storage storage.Storager) error {

	data, err := json.MarshalIndent(images, "", "\t")
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not marshal json images")
		return err
	}

	filename := name + "-output.json"
	if err = storage.Upload([]byte(data), filename, name); err != nil {
		return err
	}

	return nil
}