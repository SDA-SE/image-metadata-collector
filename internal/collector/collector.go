package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
	"strings"

	"github.com/SDA-SE/image-metadata-collector/internal/pkg/kubeclient"

	"github.com/rs/zerolog/log"

	"github.com/CycloneDX/cyclonedx-go"
)

type AnnotationNames struct {
	Base       string
	Scans      string
	Contact    string
	DefectDojo string
}

type Owner struct {
	Role string `json:"role"`
	Uuid string `json:"uuid"`
	Name string `json:"name"`
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

	Team     string  `json:"team"`
	TeamUuid string  `json:"team_uuid"`
	Slack    string  `json:"slack"`
	Email    string  `json:"email"`
	Owners   []Owner `json:"owners"`

	IsScanBaseimageLifetime          bool  `json:"is_scan_baseimage_lifetime"`
	IsScanDependencyCheck            bool  `json:"is_scan_dependency_check"`
	IsScanDependencyTrack            bool  `json:"is_scan_dependency_track"`
	IsScanDistroless                 bool  `json:"is_scan_distroless"`
	IsScanLifetime                   bool  `json:"is_scan_lifetime"`
	IsScanMalware                    bool  `json:"is_scan_malware"`
	IsScanNewVersion                 bool  `json:"is_scan_new_version"`
	IsScanRunAsRoot                  bool  `json:"is_scan_runasroot"`
	IsPotentiallyRunningAsRoot       bool  `json:"is_scan_potentially_running_as_root"`
	IsScanRunAsPrivileged            bool  `json:"is_scan_run_as_privileged"`
	IsPotentiallyRunningAsPrivileged bool  `json:"is_scan_potentially_running_as_privileged"`
	ScanLifetimeMaxDays              int64 `json:"scan_lifetime_max_days"`
}

type RunConfig struct {
	ImageFilter     []string
	NamespaceToTeam []string
}

// convertK8ImageToCollectorImage by considering the images labels, annotations and cluster wide defaults
func convertK8ImageToCollectorImage(k8Image kubeclient.Image, defaults *CollectorImage, annotationNames *AnnotationNames) *CollectorImage {
	tags := k8Image.Labels
	if tags == nil {
		tags = k8Image.Annotations
	} else {
		maps.Copy(tags, k8Image.Annotations)
	}

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
		NamespaceFilterNegated: GetOrDefaultString(tags, annotationNames.Scans+"negated_namespace_filter", defaults.NamespaceFilterNegated),
		EngagementTags:         GetOrDefaultStringSlice(tags, annotationNames.DefectDojo+"engagement-tags", defaults.EngagementTags),

		Team:     GetOrDefaultString(tags, annotationNames.Contact+"team", defaults.Team),
		TeamUuid: GetOrDefaultString(tags, annotationNames.Contact+"team_uuid", defaults.TeamUuid),
		Slack:    GetOrDefaultString(tags, annotationNames.Contact+"slack", defaults.Slack),
		Email:    GetOrDefaultString(tags, annotationNames.Contact+"email", defaults.Email),
		Owners:   GetOrDefaultOwners(tags, annotationNames.Contact+"owners", defaults.Owners),

		IsScanBaseimageLifetime:          GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-baseimage-lifetime", defaults.IsScanBaseimageLifetime),
		IsScanDependencyCheck:            GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-dependency-check", defaults.IsScanDependencyCheck),
		IsScanDependencyTrack:            GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-dependency-track", defaults.IsScanDependencyTrack),
		IsScanDistroless:                 GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-distroless", defaults.IsScanDistroless),
		IsScanLifetime:                   GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-lifetime", defaults.IsScanLifetime),
		IsScanMalware:                    GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-malware", defaults.IsScanMalware),
		IsScanNewVersion:                 GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-new-version", defaults.IsScanNewVersion),
		IsScanRunAsRoot:                  GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-runasroot", defaults.IsScanRunAsRoot),
		IsPotentiallyRunningAsRoot:       GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-potentially-running-as-root", defaults.IsPotentiallyRunningAsRoot),
		IsScanRunAsPrivileged:            GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-run-as-privileged", defaults.IsScanRunAsPrivileged),
		IsPotentiallyRunningAsPrivileged: GetOrDefaultBool(tags, annotationNames.Scans+"is-scan-potentially-running-as-privileged", defaults.IsPotentiallyRunningAsPrivileged),
		ScanLifetimeMaxDays:              GetOrDefaultInt64(tags, annotationNames.Scans+"scan-lifetime-max-days", defaults.ScanLifetimeMaxDays),
	}

	return collectorImage

}

func isSkipImage(ci *CollectorImage, imageFilter *RunConfig) bool {
	return isSkipImageByNamespace(ci) || isSkipImageByImageFilter(ci, imageFilter)
}

func isSkipImageByImageFilter(ci *CollectorImage, runConfig *RunConfig) bool {
	for _, imageFilter := range runConfig.ImageFilter {
		log.Debug().Msgf("image %s (imagefilter %s)", ci.Image, imageFilter)
		matched, err := regexp.MatchString(imageFilter, ci.Image)
		if matched && err == nil {
			return true
		}
	}

	return false
}

// considering the images labels, annotations and deployment wide defaults
func isSkipImageByNamespace(ci *CollectorImage) bool {
	isNamespaceFilter, _ := regexp.MatchString(ci.NamespaceFilter, ci.Namespace)
	if ci.NamespaceFilter == "" {
		isNamespaceFilter = false
	}

	isNamespaceFilterNegated := false
	isNamespaceFilterMatch, _ := regexp.MatchString(ci.NamespaceFilterNegated, ci.Namespace)
	if ci.NamespaceFilterNegated != "" {
		isNamespaceFilterNegated = isNamespaceFilterMatch
	}

	return ci.Skip || isNamespaceFilter || isNamespaceFilterNegated
}

// applies replacement and other rules to specific fields
func cleanCollectorImage(ci *CollectorImage, imageFilter *RunConfig) {
	ci.Image = strings.ReplaceAll(ci.Image, "docker-pullable://", "")
	ci.ImageId = cleanCollectorImageId(ci)

	ci.Skip = isSkipImage(ci, imageFilter)
}

func cleanCollectorImageId(ci *CollectorImage) string {
	var imageId = strings.ReplaceAll(ci.ImageId, "docker-pullable://", "")
	if imageId == "" {
		log.Info().Msgf("ImageId is empty for image %s (ns %s). Using image name as imageId", ci.Image, ci.Namespace)
		imageId = ci.Image
	}
	return imageId
}

// images from kubernetes, convert, clean and store them in the storage
func ConvertImages(k8Images *[]kubeclient.Image, defaults *CollectorImage, annotationNames *AnnotationNames, runConfig *RunConfig) (*[]CollectorImage, error) {
	var images []CollectorImage

	for _, k8Image := range *k8Images {
		collectorImage := convertK8ImageToCollectorImage(k8Image, defaults, annotationNames)
		cleanCollectorImage(collectorImage, runConfig)
		images = append(images, *collectorImage)

	}

	return &images, nil
}

// TODO: Write Tests. Not written yet due to upcomming refactor
// stores images in the provided storager implementation
func Store(images *[]CollectorImage, storage io.Writer, jsonMarshal JsonMarshal) error {

	if images == nil {
		err := errors.New("cannot marshal nil")
		log.Fatal().Stack().Err(err)
		return err
	}

	data, err := jsonMarshal(images)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Could not marshal json images")
		return err
	}

	if _, err = storage.Write(data); err != nil {
		return err
	}

	return nil
}

func CycloneDXMarshal(v interface{}) ([]byte, error) {
	images, ok := v.(*[]CollectorImage)
	if !ok {
		return nil, fmt.Errorf("invalid type, expected *[]CollectorImage")
	}

	bom := cyclonedx.BOM{
		BOMFormat:   "CycloneDX",
		SpecVersion: cyclonedx.SpecVersion1_5,
		Version:     1,
		Components:  &[]cyclonedx.Component{},
	}

	for _, img := range *images {
		component := cyclonedx.Component{
			Type:       cyclonedx.ComponentTypeContainer,
			Name:       img.Image,
			Version:    img.AppKubernetesIoVersion,
			PackageURL: img.ImageId,
			Properties: &[]cyclonedx.Property{
				{Name: "namespace", Value: img.Namespace},
				{Name: "environment", Value: img.Environment},
				{Name: "product", Value: img.Product},
				{Name: "description", Value: img.Description},
				{Name: "app_kubernetes_io_name", Value: img.AppKubernetesIoName},
				{Name: "container_type", Value: img.ContainerType},
				{Name: "team", Value: img.Team},
				{Name: "slack", Value: img.Slack},
				{Name: "email", Value: img.Email},
				{Name: "is_scan_baseimage_lifetime", Value: fmt.Sprintf("%v", img.IsScanBaseimageLifetime)},
				{Name: "is_scan_dependency_check", Value: fmt.Sprintf("%v", img.IsScanDependencyCheck)},
				{Name: "is_scan_dependency_track", Value: fmt.Sprintf("%v", img.IsScanDependencyTrack)},
				{Name: "is_scan_distroless", Value: fmt.Sprintf("%v", img.IsScanDistroless)},
				{Name: "is_scan_lifetime", Value: fmt.Sprintf("%v", img.IsScanLifetime)},
				{Name: "is_scan_malware", Value: fmt.Sprintf("%v", img.IsScanMalware)},
				{Name: "is_scan_new_version", Value: fmt.Sprintf("%v", img.IsScanNewVersion)},
				{Name: "is_scan_runasroot", Value: fmt.Sprintf("%v", img.IsScanRunAsRoot)},
				{Name: "is_scan_potentially_running_as_root", Value: fmt.Sprintf("%v", img.IsPotentiallyRunningAsRoot)},
				{Name: "is_scan_run_as_privileged", Value: fmt.Sprintf("%v", img.IsScanRunAsPrivileged)},
				{Name: "is_scan_potentially_running_as_privileged", Value: fmt.Sprintf("%v", img.IsPotentiallyRunningAsPrivileged)},
				{Name: "scan_lifetime_max_days", Value: fmt.Sprintf("%d", img.ScanLifetimeMaxDays)},
			},
		}
		*bom.Components = append(*bom.Components, component)
	}

	return json.MarshalIndent(bom, "", "  ")
}
